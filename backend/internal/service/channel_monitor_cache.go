package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"sync"
	"time"
)

// 渠道监控「缓存率」：取监控所用 API Key 所在分组，汇总该分组在 usage_logs 近 cacheRateWindow 的真实用量，
//
//	缓存率 = cache_read_tokens / 输入侧 token（input + cache_creation + cache_read，不含输出）
//
// 仅当监控的 API Key 是本站 API Key 且已绑定分组时才有数据（否则返回 nil）。
const (
	cacheRateWindow   = 7 * 24 * time.Hour
	cacheRateCacheTTL = time.Minute
)

// ChannelMonitorCacheSettings 单个监控的缓存率展示设置。
type ChannelMonitorCacheSettings struct {
	MonitorID     int64 `json:"monitor_id"`
	ShowCacheRate bool  `json:"show_cache_rate"`
}

type cacheRateEntry struct {
	rate *float64
	at   time.Time
}

// ChannelMonitorCacheService 渠道监控缓存率。
type ChannelMonitorCacheService struct {
	db       *sql.DB
	monitors *ChannelMonitorService
	mu       sync.Mutex
	cached   map[int64]cacheRateEntry
}

func NewChannelMonitorCacheService(db *sql.DB, monitors *ChannelMonitorService) *ChannelMonitorCacheService {
	return &ChannelMonitorCacheService{db: db, monitors: monitors, cached: map[int64]cacheRateEntry{}}
}

// Cache 返回注入的缓存率服务（可能为 nil）。
func (s *ChannelMonitorService) Cache() *ChannelMonitorCacheService { return s.cache }

// SetCache 注入缓存率服务。
func (s *ChannelMonitorService) SetCache(c *ChannelMonitorCacheService) { s.cache = c }

func (s *ChannelMonitorCacheService) GetSettings(ctx context.Context, monitorID int64) (*ChannelMonitorCacheSettings, error) {
	out := &ChannelMonitorCacheSettings{MonitorID: monitorID}
	err := s.db.QueryRowContext(ctx,
		`SELECT show_cache_rate FROM channel_monitor_cache_settings WHERE monitor_id = $1`, monitorID,
	).Scan(&out.ShowCacheRate)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return out, nil
}

func (s *ChannelMonitorCacheService) SaveSettings(ctx context.Context, in *ChannelMonitorCacheSettings) (*ChannelMonitorCacheSettings, error) {
	if _, err := s.monitors.repo.GetByID(ctx, in.MonitorID); err != nil {
		return nil, err
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO channel_monitor_cache_settings (monitor_id, show_cache_rate, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (monitor_id) DO UPDATE SET show_cache_rate = EXCLUDED.show_cache_rate, updated_at = NOW()`,
		in.MonitorID, in.ShowCacheRate); err != nil {
		return nil, err
	}
	s.mu.Lock()
	delete(s.cached, in.MonitorID)
	s.mu.Unlock()
	return s.GetSettings(ctx, in.MonitorID)
}

// EnabledSet 返回开启了缓存率展示的监控 ID 集合。
func (s *ChannelMonitorCacheService) EnabledSet(ctx context.Context) (map[int64]bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT monitor_id FROM channel_monitor_cache_settings WHERE show_cache_rate = true`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// Rate 返回监控的缓存率（0-100）；无数据返回 nil。结果缓存 cacheRateCacheTTL，避免列表轮询反复聚合。
func (s *ChannelMonitorCacheService) Rate(ctx context.Context, monitorID int64) *float64 {
	s.mu.Lock()
	if e, ok := s.cached[monitorID]; ok && time.Since(e.at) < cacheRateCacheTTL {
		s.mu.Unlock()
		return e.rate
	}
	s.mu.Unlock()

	rate, err := s.compute(ctx, monitorID)
	if err != nil {
		slog.Warn("channel_monitor_cache: compute failed", "monitor_id", monitorID, "error", err)
		return nil
	}
	s.mu.Lock()
	s.cached[monitorID] = cacheRateEntry{rate: rate, at: time.Now()}
	s.mu.Unlock()
	return rate
}

func (s *ChannelMonitorCacheService) compute(ctx context.Context, monitorID int64) (*float64, error) {
	m, err := s.monitors.Get(ctx, monitorID) // Get 会解密 API Key
	if err != nil {
		return nil, err
	}
	if m.APIKeyDecryptFailed || m.APIKey == "" {
		return nil, nil
	}
	var cacheRead, total int64
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(u.cache_read_tokens), 0),
		       COALESCE(SUM(u.input_tokens + u.cache_creation_tokens + u.cache_read_tokens), 0)
		FROM usage_logs u
		WHERE u.group_id = (SELECT k.group_id FROM api_keys k WHERE k.key = $1 LIMIT 1)
		  AND u.created_at >= NOW() - make_interval(secs => $2)`,
		m.APIKey, cacheRateWindow.Seconds()).Scan(&cacheRead, &total)
	if err != nil {
		return nil, err
	}
	if total <= 0 {
		return nil, nil
	}
	rate := float64(cacheRead) / float64(total) * 100
	return &rate, nil
}
