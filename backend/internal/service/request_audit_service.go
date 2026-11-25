package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// SettingKeyRequestAudit 网关请求审计设置（JSON）。
const SettingKeyRequestAudit = "request_audit_settings"

const (
	RequestAuditDefaultRetentionDays = 7
	RequestAuditDefaultMaxBodyBytes  = 1 << 20
	RequestAuditMinMaxBodyBytes      = 4 << 10
	RequestAuditMaxMaxBodyBytes      = 16 << 20
	RequestAuditMaxRetentionDays     = 365

	requestAuditQueueCapacity    = 512
	requestAuditQueueByteBudget  = 256 << 20
	requestAuditBatchSize        = 20
	requestAuditFlushInterval    = time.Second
	requestAuditSettingsCacheTTL = 15 * time.Second

	requestAuditRetentionInterval     = time.Hour
	requestAuditRetentionStartupDelay = 2 * time.Minute
	requestAuditRetentionBatchSize    = 2000

	// requestAuditLookupSlack 按时间兜底关联 usage 记录时允许的时钟/落库偏差。
	requestAuditLookupSlack = 60 * time.Second
)

var ErrRequestAuditNotFound = infraerrors.NotFound("REQUEST_AUDIT_NOT_FOUND", "request audit record not found")

// RequestAuditSettings 管理员可调的请求审计设置。
type RequestAuditSettings struct {
	Enabled       bool `json:"enabled"`
	RetentionDays int  `json:"retention_days"`
	MaxBodyBytes  int  `json:"max_body_bytes"`
}

func DefaultRequestAuditSettings() RequestAuditSettings {
	return RequestAuditSettings{
		Enabled:       true,
		RetentionDays: RequestAuditDefaultRetentionDays,
		MaxBodyBytes:  RequestAuditDefaultMaxBodyBytes,
	}
}

// Normalize 把越界值夹到合法范围；RetentionDays=0 表示永久保留。
func (s RequestAuditSettings) Normalize() RequestAuditSettings {
	if s.RetentionDays < 0 {
		s.RetentionDays = 0
	}
	if s.RetentionDays > RequestAuditMaxRetentionDays {
		s.RetentionDays = RequestAuditMaxRetentionDays
	}
	if s.MaxBodyBytes <= 0 {
		s.MaxBodyBytes = RequestAuditDefaultMaxBodyBytes
	}
	if s.MaxBodyBytes < RequestAuditMinMaxBodyBytes {
		s.MaxBodyBytes = RequestAuditMinMaxBodyBytes
	}
	if s.MaxBodyBytes > RequestAuditMaxMaxBodyBytes {
		s.MaxBodyBytes = RequestAuditMaxMaxBodyBytes
	}
	return s
}

// RequestAuditLog 一次网关请求的完整输入/输出快照。
type RequestAuditLog struct {
	ID                  int64             `json:"id"`
	CreatedAt           time.Time         `json:"created_at"`
	RequestID           string            `json:"request_id"`
	ClientRequestID     string            `json:"client_request_id"`
	UserID              *int64            `json:"user_id,omitempty"`
	APIKeyID            *int64            `json:"api_key_id,omitempty"`
	AccountID           *int64            `json:"account_id,omitempty"`
	Method              string            `json:"method"`
	Path                string            `json:"path"`
	Model               string            `json:"model"`
	Stream              bool              `json:"stream"`
	StatusCode          int               `json:"status_code"`
	DurationMs          int64             `json:"duration_ms"`
	ClientIP            string            `json:"client_ip"`
	UserAgent           string            `json:"user_agent"`
	RequestHeaders      map[string]string `json:"request_headers"`
	RequestContentType  string            `json:"request_content_type"`
	RequestBody         string            `json:"request_body"`
	RequestBodySize     int64             `json:"request_body_size"`
	RequestTruncated    bool              `json:"request_truncated"`
	ResponseContentType string            `json:"response_content_type"`
	ResponseBody        string            `json:"response_body"`
	ResponseBodySize    int64             `json:"response_body_size"`
	ResponseTruncated   bool              `json:"response_truncated"`
}

// RequestAuditLookup 从 usage 记录关联审计记录的条件。
type RequestAuditLookup struct {
	ClientRequestID string
	RequestID       string
	APIKeyID        int64
	Target          time.Time
	From            time.Time
	To              time.Time
}

type RequestAuditRepository interface {
	BatchInsert(ctx context.Context, logs []*RequestAuditLog) (int64, error)
	FindForUsage(ctx context.Context, lookup RequestAuditLookup) (*RequestAuditLog, error)
	DeleteBefore(ctx context.Context, cutoff time.Time, batchSize int) (int64, error)
}

type requestAuditSettingsCache struct {
	settings RequestAuditSettings
	loadedAt time.Time
}

// RequestAuditService 网关请求审计：异步批量落库 + 保留期清理 + 按 usage 记录查询。
type RequestAuditService struct {
	repo        RequestAuditRepository
	settingRepo SettingRepository
	usageRepo   UsageLogRepository

	cache      atomic.Pointer[requestAuditSettingsCache]
	refreshing atomic.Bool

	queue       chan *RequestAuditCapture
	queuedBytes atomic.Int64

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	dropped     atomic.Uint64
	writeFailed atomic.Uint64
}

func NewRequestAuditService(repo RequestAuditRepository, settingRepo SettingRepository, usageRepo UsageLogRepository) *RequestAuditService {
	ctx, cancel := context.WithCancel(context.Background())
	return &RequestAuditService{
		repo:        repo,
		settingRepo: settingRepo,
		usageRepo:   usageRepo,
		queue:       make(chan *RequestAuditCapture, requestAuditQueueCapacity),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// ProvideRequestAuditService 创建服务并启动写入与清理协程；停止挂在 cmd/server 的 provideCleanup。
func ProvideRequestAuditService(repo RequestAuditRepository, settingRepo SettingRepository, usageRepo UsageLogRepository) *RequestAuditService {
	svc := NewRequestAuditService(repo, settingRepo, usageRepo)
	svc.Start()
	return svc
}

func (s *RequestAuditService) Start() {
	if s == nil || s.repo == nil {
		return
	}
	s.wg.Add(2)
	go s.runWriter()
	go s.runRetentionLoop()
}

func (s *RequestAuditService) Stop() {
	if s == nil {
		return
	}
	s.cancel()
	s.wg.Wait()
}

// Settings 返回当前设置，网关热路径每请求调用。
// 仅首次（无缓存）同步读库；缓存过期后立即返回旧值，并由单个后台协程刷新，
// 不会让请求等待数据库，也不会在过期瞬间并发打库。
func (s *RequestAuditService) Settings(ctx context.Context) RequestAuditSettings {
	if s == nil {
		return RequestAuditSettings{}
	}
	cached := s.cache.Load()
	if cached == nil {
		settings := s.loadSettings(ctx)
		s.cache.Store(&requestAuditSettingsCache{settings: settings, loadedAt: time.Now()})
		return settings
	}
	if time.Since(cached.loadedAt) >= requestAuditSettingsCacheTTL && s.refreshing.CompareAndSwap(false, true) {
		go func() {
			defer s.refreshing.Store(false)
			refreshCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			settings := s.loadSettings(refreshCtx)
			s.cache.Store(&requestAuditSettingsCache{settings: settings, loadedAt: time.Now()})
		}()
	}
	return cached.settings
}

func (s *RequestAuditService) loadSettings(ctx context.Context) RequestAuditSettings {
	settings := DefaultRequestAuditSettings()
	if s.settingRepo == nil {
		return settings
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyRequestAudit)
	if err != nil || strings.TrimSpace(raw) == "" {
		// 读取失败时沿用上次缓存，避免数据库抖动导致审计开关来回切换。
		if err != nil && !errors.Is(err, ErrSettingNotFound) {
			if cached := s.cache.Load(); cached != nil {
				return cached.settings
			}
		}
		return settings
	}
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return DefaultRequestAuditSettings()
	}
	return settings.Normalize()
}

func (s *RequestAuditService) UpdateSettings(ctx context.Context, settings RequestAuditSettings) (RequestAuditSettings, error) {
	settings = settings.Normalize()
	encoded, err := json.Marshal(settings)
	if err != nil {
		return settings, err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyRequestAudit, string(encoded)); err != nil {
		return settings, fmt.Errorf("save request audit settings: %w", err)
	}
	s.cache.Store(&requestAuditSettingsCache{settings: settings, loadedAt: time.Now()})
	return settings, nil
}

// Record 非阻塞入队原始抓取；队列条数或字节预算打满时丢弃并计数，绝不阻塞网关请求。
// 渲染与落库都在写入协程中进行。
func (s *RequestAuditService) Record(entry *RequestAuditCapture) {
	if s == nil || entry == nil || entry.Log == nil || s.repo == nil {
		return
	}
	select {
	case <-s.ctx.Done():
		return
	default:
	}
	size := entry.approxBytes()
	if s.queuedBytes.Add(size) > requestAuditQueueByteBudget {
		s.queuedBytes.Add(-size)
		s.dropped.Add(1)
		return
	}
	select {
	case s.queue <- entry:
	default:
		s.queuedBytes.Add(-size)
		s.dropped.Add(1)
	}
}

// GetForUsageLog 查询某条使用记录对应的请求审计详情。
func (s *RequestAuditService) GetForUsageLog(ctx context.Context, usageLogID int64) (*RequestAuditLog, error) {
	if s == nil || s.usageRepo == nil || s.repo == nil {
		return nil, ErrRequestAuditNotFound
	}
	usage, err := s.usageRepo.GetByID(ctx, usageLogID)
	if err != nil {
		return nil, err
	}
	if usage == nil {
		return nil, ErrRequestAuditNotFound
	}
	duration := time.Duration(0)
	if usage.DurationMs != nil && *usage.DurationMs > 0 {
		duration = time.Duration(*usage.DurationMs) * time.Millisecond
	}
	lookup := RequestAuditLookup{
		APIKeyID: usage.APIKeyID,
		// 审计记录的 created_at 是请求开始时间，usage 的 created_at 接近请求结束。
		Target: usage.CreatedAt.Add(-duration),
		From:   usage.CreatedAt.Add(-duration - requestAuditLookupSlack),
		To:     usage.CreatedAt.Add(requestAuditLookupSlack),
	}
	rid := strings.TrimSpace(usage.RequestID)
	switch {
	case strings.HasPrefix(rid, "client:"):
		lookup.ClientRequestID = strings.TrimPrefix(rid, "client:")
	case strings.HasPrefix(rid, "local:"):
		lookup.RequestID = strings.TrimPrefix(rid, "local:")
	}
	log, err := s.repo.FindForUsage(ctx, lookup)
	if err != nil {
		return nil, err
	}
	if log == nil {
		return nil, ErrRequestAuditNotFound
	}
	return log, nil
}

func (s *RequestAuditService) runWriter() {
	defer s.wg.Done()
	ticker := time.NewTicker(requestAuditFlushInterval)
	defer ticker.Stop()

	batch := make([]*RequestAuditLog, 0, requestAuditBatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		_, err := s.repo.BatchInsert(ctx, batch)
		cancel()
		if err != nil {
			s.writeFailed.Add(uint64(len(batch)))
			_, _ = fmt.Fprintf(os.Stderr, "time=%s level=WARN msg=\"request audit flush failed\" err=%v batch=%d\n",
				time.Now().Format(time.RFC3339Nano), err, len(batch))
		}
		for i := range batch {
			batch[i] = nil
		}
		batch = batch[:0]
	}
	add := func(item *RequestAuditCapture) {
		if item == nil {
			return
		}
		// 出队即释放字节预算：原始抓取在渲染后不再被引用。
		s.queuedBytes.Add(-item.approxBytes())
		log := s.renderSafely(item)
		if log == nil {
			return
		}
		batch = append(batch, log)
		if len(batch) >= requestAuditBatchSize {
			flush()
		}
	}

	for {
		select {
		case <-s.ctx.Done():
			for {
				select {
				case item := <-s.queue:
					add(item)
				default:
					flush()
					return
				}
			}
		case item := <-s.queue:
			add(item)
		case <-ticker.C:
			flush()
		}
	}
}

// renderSafely 渲染单条抓取；渲染异常只丢弃该条，不影响写入协程与网关。
func (s *RequestAuditService) renderSafely(item *RequestAuditCapture) (log *RequestAuditLog) {
	defer func() {
		if r := recover(); r != nil {
			s.writeFailed.Add(1)
			log = nil
			_, _ = fmt.Fprintf(os.Stderr, "time=%s level=WARN msg=\"request audit render panic\" err=%v\n",
				time.Now().Format(time.RFC3339Nano), r)
		}
	}()
	return renderRequestAuditCapture(item)
}

func (s *RequestAuditService) runRetentionLoop() {
	defer s.wg.Done()
	startup := time.NewTimer(requestAuditRetentionStartupDelay)
	defer startup.Stop()
	select {
	case <-s.ctx.Done():
		return
	case <-startup.C:
	}
	ticker := time.NewTicker(requestAuditRetentionInterval)
	defer ticker.Stop()
	s.runRetentionOnce()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.runRetentionOnce()
		}
	}
}

func (s *RequestAuditService) runRetentionOnce() {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Minute)
	defer cancel()
	days := s.Settings(ctx).RetentionDays
	if days <= 0 {
		return
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	for {
		deleted, err := s.repo.DeleteBefore(ctx, cutoff, requestAuditRetentionBatchSize)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "time=%s level=WARN msg=\"request audit retention cleanup failed\" err=%v\n",
				time.Now().Format(time.RFC3339Nano), err)
			return
		}
		if deleted == 0 {
			return
		}
	}
}
