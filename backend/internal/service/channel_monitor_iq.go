package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/tidwall/gjson"
)

// 渠道监控的智商（降智）检测：
//   - candy（糖果）：完全在本服务内完成，用渠道自身的 endpoint / api key / 模型直接请求；
//   - pelican（鹈鹕）与 creative（自由创作）：由第三方检测服务 manxue.ai 生成并分类，
//     该服务会收到渠道的 base_url 与 api key，仅在管理员显式开启时使用。
//
// 只保留最近 iqRetention 内的成功结果，每个 (监控, 模型, 检测类型) 最多 iqMaxPerModel 条。
const (
	IQBenchmarkCandy    = "candy"
	IQBenchmarkPelican  = "pelican"
	IQBenchmarkCreative = "creative"

	iqRetention              = 2 * time.Hour
	iqMaxPerModel            = 10
	iqDefaultIntervalMinutes = 10
	iqMinIntervalMinutes     = 5
	iqMaxIntervalMinutes     = 1440
	iqMaxModels              = 20
	iqManxueEndpoint         = "https://manxue.ai/api/v1/tests"
	iqManxuePollInterval     = 3 * time.Second
	iqManxueMaxWait          = 10 * time.Minute
	iqRunTimeout             = 40 * time.Minute
	iqCandyTimeout           = 3 * time.Minute
	iqMaxHTMLBytes           = 2 << 20
	iqManxueMaxBodyBytes     = 8 << 20
)

var ErrChannelMonitorIQInvalid = infraerrors.BadRequest("CHANNEL_MONITOR_IQ_INVALID", "invalid iq settings")

// ChannelMonitorIQSettings 单个监控的智商检测设置。Models 为空表示检测该监控的全部模型。
type ChannelMonitorIQSettings struct {
	MonitorID       int64 `json:"monitor_id"`
	Enabled         bool  `json:"enabled"`
	CandyEnabled    bool  `json:"candy_enabled"`    // 糖果测试
	PelicanEnabled  bool  `json:"pelican_enabled"`  // 鹈鹕测试
	CreativeEnabled bool  `json:"creative_enabled"` // 鹈鹕测试的配置：开启后以自由创作替代鹈鹕请求

	IntervalMinutes int        `json:"interval_minutes"`
	Models          []string   `json:"models"`
	LastRunAt       *time.Time `json:"last_run_at"`
}

// ChannelMonitorIQResult 一次成功的智商检测结果。
type ChannelMonitorIQResult struct {
	ID         int64     `json:"id"`
	MonitorID  int64     `json:"monitor_id"`
	Model      string    `json:"model"`
	Benchmark  string    `json:"benchmark"`
	Verdict    string    `json:"verdict,omitempty"` // candy: passed / incorrect
	Quality    string    `json:"quality,omitempty"` // pelican/creative: normal / degraded / unknown / suspicious
	Label      string    `json:"label,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	Answer     string    `json:"answer,omitempty"`
	HasHTML    bool      `json:"has_html"`
	HTML       string    `json:"html,omitempty"` // 仅详情接口返回
	DurationMs int64     `json:"duration_ms"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}

// ChannelMonitorIQService 渠道监控智商检测。
type ChannelMonitorIQService struct {
	db       *sql.DB
	monitors *ChannelMonitorService
	client   *http.Client
	running  sync.Map // monitorID -> struct{}
	failures sync.Map // "monitorID|model|benchmark" -> *ChannelMonitorIQFailure（仅内存，成功后清除）
}

// ChannelMonitorIQFailure 最近一次失败的检测（结果表只存成功记录，失败原因另存内存供弹窗展示）。
type ChannelMonitorIQFailure struct {
	MonitorID int64     `json:"-"`
	Model     string    `json:"model"`
	Benchmark string    `json:"benchmark"`
	Message   string    `json:"message"`
	At        time.Time `json:"at"`
}

func iqFailureKey(monitorID int64, model, benchmark string) string {
	return fmt.Sprintf("%d|%s|%s", monitorID, model, benchmark)
}

func (s *ChannelMonitorIQService) recordFailure(monitorID int64, model, benchmark string, err error) {
	s.failures.Store(iqFailureKey(monitorID, model, benchmark), &ChannelMonitorIQFailure{
		MonitorID: monitorID, Model: model, Benchmark: benchmark,
		Message: truncateForIQ(sanitizeErrorMessage(err.Error()), 300), At: time.Now(),
	})
}

func (s *ChannelMonitorIQService) clearFailure(monitorID int64, model, benchmark string) {
	s.failures.Delete(iqFailureKey(monitorID, model, benchmark))
}

// ListFailures 返回某监控各 (模型, 检测类型) 最近一次失败原因，最新在前。
func (s *ChannelMonitorIQService) ListFailures(monitorID int64) []*ChannelMonitorIQFailure {
	out := []*ChannelMonitorIQFailure{}
	s.failures.Range(func(_, v any) bool {
		if f := v.(*ChannelMonitorIQFailure); f.MonitorID == monitorID {
			out = append(out, f)
		}
		return true
	})
	sort.Slice(out, func(i, j int) bool { return out[i].At.After(out[j].At) })
	return out
}

func NewChannelMonitorIQService(db *sql.DB, monitors *ChannelMonitorService) *ChannelMonitorIQService {
	return &ChannelMonitorIQService{db: db, monitors: monitors, client: &http.Client{Timeout: 30 * time.Second}}
}

// IQ 返回注入的智商检测服务（可能为 nil）。
func (s *ChannelMonitorService) IQ() *ChannelMonitorIQService { return s.iq }

// SetIQ 注入智商检测服务。
func (s *ChannelMonitorService) SetIQ(iq *ChannelMonitorIQService) { s.iq = iq }

// ---------- settings ----------

func defaultIQSettings(monitorID int64) *ChannelMonitorIQSettings {
	return &ChannelMonitorIQSettings{MonitorID: monitorID, CandyEnabled: true, PelicanEnabled: true, IntervalMinutes: iqDefaultIntervalMinutes, Models: []string{}}
}

func (s *ChannelMonitorIQService) GetSettings(ctx context.Context, monitorID int64) (*ChannelMonitorIQSettings, error) {
	out := defaultIQSettings(monitorID)
	var raw []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT enabled, candy_enabled, pelican_enabled, creative_enabled, interval_minutes, models, last_run_at
		FROM channel_monitor_iq_settings WHERE monitor_id = $1`, monitorID,
	).Scan(&out.Enabled, &out.CandyEnabled, &out.PelicanEnabled, &out.CreativeEnabled, &out.IntervalMinutes, &raw, &out.LastRunAt)
	if errors.Is(err, sql.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(raw, &out.Models)
	if out.Models == nil {
		out.Models = []string{}
	}
	return out, nil
}

func (s *ChannelMonitorIQService) SaveSettings(ctx context.Context, in *ChannelMonitorIQSettings) (*ChannelMonitorIQSettings, error) {
	if _, err := s.monitors.repo.GetByID(ctx, in.MonitorID); err != nil {
		return nil, err
	}
	if in.IntervalMinutes < iqMinIntervalMinutes || in.IntervalMinutes > iqMaxIntervalMinutes {
		return nil, ErrChannelMonitorIQInvalid.WithMetadata(map[string]string{
			"detail": fmt.Sprintf("interval_minutes must be between %d and %d", iqMinIntervalMinutes, iqMaxIntervalMinutes),
		})
	}
	if in.Enabled && !in.CandyEnabled && !in.PelicanEnabled {
		return nil, ErrChannelMonitorIQInvalid.WithMetadata(map[string]string{"detail": "at least one test must be selected"})
	}
	seen := map[string]struct{}{}
	models := make([]string, 0, len(in.Models))
	for _, m := range in.Models {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		if _, dup := seen[m]; dup {
			continue
		}
		seen[m] = struct{}{}
		models = append(models, m)
	}
	if len(models) > iqMaxModels {
		return nil, ErrChannelMonitorIQInvalid.WithMetadata(map[string]string{"detail": "too many models"})
	}
	rawModels, _ := json.Marshal(models)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO channel_monitor_iq_settings (monitor_id, enabled, candy_enabled, pelican_enabled, creative_enabled, interval_minutes, models, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, NOW())
		ON CONFLICT (monitor_id) DO UPDATE SET
			enabled = EXCLUDED.enabled, candy_enabled = EXCLUDED.candy_enabled,
			pelican_enabled = EXCLUDED.pelican_enabled, creative_enabled = EXCLUDED.creative_enabled,
			interval_minutes = EXCLUDED.interval_minutes, models = EXCLUDED.models, updated_at = NOW()`,
		in.MonitorID, in.Enabled, in.CandyEnabled, in.PelicanEnabled, in.CreativeEnabled, in.IntervalMinutes, string(rawModels))
	if err != nil {
		return nil, err
	}
	return s.GetSettings(ctx, in.MonitorID)
}

// EnabledSet 返回已开启智商检测的监控 ID 集合（用户端卡片据此显示「智商状态」按钮）。
func (s *ChannelMonitorIQService) EnabledSet(ctx context.Context) (map[int64]bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT monitor_id FROM channel_monitor_iq_settings WHERE enabled = true`)
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

// iqRingSegments 卡片智商环的分段数（取最近 N 条结果）。
const iqRingSegments = 10

// RecentOutcomes 返回每个已开启智商检测的监控最近 iqRingSegments 条结果是否通过
// （糖果：答对；鹈鹕/自由创作：quality=normal），按时间从旧到新排列。
func (s *ChannelMonitorIQService) RecentOutcomes(ctx context.Context) (map[int64][]bool, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT monitor_id, passed FROM (
			SELECT r.monitor_id,
			       (CASE WHEN r.benchmark = 'candy' THEN r.verdict = 'passed' ELSE r.quality = 'normal' END) AS passed,
			       ROW_NUMBER() OVER (PARTITION BY r.monitor_id ORDER BY r.created_at DESC, r.id DESC) AS rn
			FROM channel_monitor_iq_results r
			JOIN channel_monitor_iq_settings st ON st.monitor_id = r.monitor_id AND st.enabled = true
			WHERE r.created_at >= NOW() - make_interval(secs => $1)
			  AND ((r.benchmark = 'candy' AND st.candy_enabled) OR (r.benchmark <> 'candy' AND st.pelican_enabled))
		) t WHERE rn <= $2
		ORDER BY monitor_id, rn DESC`, iqRetention.Seconds(), iqRingSegments)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[int64][]bool{}
	for rows.Next() {
		var id int64
		var passed bool
		if err := rows.Scan(&id, &passed); err != nil {
			return nil, err
		}
		out[id] = append(out[id], passed)
	}
	return out, rows.Err()
}

// ---------- results ----------

// ListResults 返回最近 iqRetention 内的成功结果（不含 html），最新在前。
func (s *ChannelMonitorIQService) ListResults(ctx context.Context, monitorID int64) ([]*ChannelMonitorIQResult, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, monitor_id, model, benchmark, verdict, quality, label, reason, answer, (html <> '') AS has_html,
		       duration_ms, started_at, finished_at
		FROM channel_monitor_iq_results
		WHERE monitor_id = $1 AND created_at >= NOW() - make_interval(secs => $2)
		ORDER BY created_at DESC, id DESC`, monitorID, iqRetention.Seconds())
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []*ChannelMonitorIQResult{}
	for rows.Next() {
		r := &ChannelMonitorIQResult{}
		if err := rows.Scan(&r.ID, &r.MonitorID, &r.Model, &r.Benchmark, &r.Verdict, &r.Quality, &r.Label, &r.Reason,
			&r.Answer, &r.HasHTML, &r.DurationMs, &r.StartedAt, &r.FinishedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetResult 返回单条结果（含 html）。
func (s *ChannelMonitorIQService) GetResult(ctx context.Context, monitorID, resultID int64) (*ChannelMonitorIQResult, error) {
	r := &ChannelMonitorIQResult{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, monitor_id, model, benchmark, verdict, quality, label, reason, answer, html, (html <> ''),
		       duration_ms, started_at, finished_at
		FROM channel_monitor_iq_results
		WHERE id = $1 AND monitor_id = $2 AND created_at >= NOW() - make_interval(secs => $3)`,
		resultID, monitorID, iqRetention.Seconds(),
	).Scan(&r.ID, &r.MonitorID, &r.Model, &r.Benchmark, &r.Verdict, &r.Quality, &r.Label, &r.Reason, &r.Answer,
		&r.HTML, &r.HasHTML, &r.DurationMs, &r.StartedAt, &r.FinishedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrChannelMonitorNotFound
	}
	return r, err
}

func (s *ChannelMonitorIQService) saveResult(ctx context.Context, r *ChannelMonitorIQResult) error {
	if len(r.HTML) > iqMaxHTMLBytes {
		r.HTML = r.HTML[:iqMaxHTMLBytes]
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO channel_monitor_iq_results
			(monitor_id, model, benchmark, verdict, quality, label, reason, answer, html, duration_ms, started_at, finished_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		r.MonitorID, r.Model, r.Benchmark, r.Verdict, r.Quality, r.Label, r.Reason, r.Answer, r.HTML,
		r.DurationMs, r.StartedAt, r.FinishedAt); err != nil {
		return err
	}
	// 滚动保留：仅保留最近 iqRetention，且每个 (监控, 模型, 检测类型) 最多 iqMaxPerModel 条。
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM channel_monitor_iq_results
		WHERE monitor_id = $1 AND (
			created_at < NOW() - make_interval(secs => $2)
			OR id IN (
				SELECT id FROM (
					SELECT id, ROW_NUMBER() OVER (PARTITION BY model, benchmark ORDER BY created_at DESC, id DESC) AS rn
					FROM channel_monitor_iq_results WHERE monitor_id = $1
				) ranked WHERE rn > $3
			)
		)`, r.MonitorID, iqRetention.Seconds(), iqMaxPerModel)
	return err
}

// ---------- scheduling ----------

// Trigger 在一次探活完成后调用：若该监控开启了智商检测且距上次已超过设定间隔，则异步执行一轮。
func (s *ChannelMonitorIQService) Trigger(m *ChannelMonitor) {
	if s == nil || m == nil || m.APIKeyDecryptFailed || strings.TrimSpace(m.Endpoint) == "" || strings.TrimSpace(m.APIKey) == "" {
		return
	}
	if defaultCheckMode(m.CheckMode) == MonitorCheckModeQuota {
		return
	}
	if _, busy := s.running.LoadOrStore(m.ID, struct{}{}); busy {
		return
	}
	go func() {
		defer s.running.Delete(m.ID)
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("channel_monitor_iq: panic", "monitor_id", m.ID, "panic", rec)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), iqRunTimeout)
		defer cancel()
		s.runIfDue(ctx, m)
	}()
}

func (s *ChannelMonitorIQService) runIfDue(ctx context.Context, m *ChannelMonitor) {
	// 原子占位：仅当已开启且距上次运行超过间隔时更新 last_run_at，多实例下也只会有一个执行。
	var candy, pelican, creative bool
	var rawModels []byte
	err := s.db.QueryRowContext(ctx, `
		UPDATE channel_monitor_iq_settings SET last_run_at = NOW()
		WHERE monitor_id = $1 AND enabled = true
		  AND (last_run_at IS NULL OR last_run_at <= NOW() - make_interval(mins => interval_minutes))
		RETURNING candy_enabled, pelican_enabled, creative_enabled, models`, m.ID).Scan(&candy, &pelican, &creative, &rawModels)
	if errors.Is(err, sql.ErrNoRows) {
		return
	}
	if err != nil {
		slog.Warn("channel_monitor_iq: claim failed", "monitor_id", m.ID, "error", err)
		return
	}
	var selected []string
	_ = json.Unmarshal(rawModels, &selected)
	for _, model := range s.pickModels(m, selected) {
		if ctx.Err() != nil {
			return
		}
		s.runModel(ctx, m, model, candy, pelican, creative)
	}
}

// pickModels 取监控的 primary + extra 模型；selected 非空时只取其交集。
func (s *ChannelMonitorIQService) pickModels(m *ChannelMonitor, selected []string) []string {
	all := append([]string{m.PrimaryModel}, m.ExtraModels...)
	want := map[string]struct{}{}
	for _, x := range selected {
		want[x] = struct{}{}
	}
	seen := map[string]struct{}{}
	out := []string{}
	for _, model := range all {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		if _, dup := seen[model]; dup {
			continue
		}
		seen[model] = struct{}{}
		if len(want) > 0 {
			if _, ok := want[model]; !ok {
				continue
			}
		}
		out = append(out, model)
	}
	return out
}

// runModel 按设置运行所选测试。creative 是鹈鹕测试的配置：开启后只请求自由创作，不再请求鹈鹕。
func (s *ChannelMonitorIQService) runModel(ctx context.Context, m *ChannelMonitor, model string, candy, pelican, creative bool) {
	if candy {
		if r, err := s.runCandy(ctx, m, model); err != nil {
			slog.Warn("channel_monitor_iq: candy failed", "monitor_id", m.ID, "model", model, "error", err)
			s.recordFailure(m.ID, model, IQBenchmarkCandy, err)
		} else if err := s.saveResult(ctx, r); err != nil {
			slog.Warn("channel_monitor_iq: save candy failed", "monitor_id", m.ID, "error", err)
		} else {
			s.clearFailure(m.ID, model, IQBenchmarkCandy)
		}
	}
	if !pelican || ctx.Err() != nil {
		return
	}
	bm := IQBenchmarkPelican
	if creative {
		bm = IQBenchmarkCreative
	}
	r, err := s.runManxue(ctx, m, model, bm)
	if err != nil {
		slog.Warn("channel_monitor_iq: manxue failed", "monitor_id", m.ID, "model", model, "benchmark", bm, "error", err)
		s.recordFailure(m.ID, model, bm, err)
		return
	}
	if err := s.saveResult(ctx, r); err != nil {
		slog.Warn("channel_monitor_iq: save result failed", "monitor_id", m.ID, "benchmark", bm, "error", err)
		return
	}
	s.clearFailure(m.ID, model, bm)
}

// ---------- candy (local) ----------

func (s *ChannelMonitorIQService) runCandy(ctx context.Context, m *ChannelMonitor, model string) (*ChannelMonitorIQResult, error) {
	if _, ok := providerAdapters[m.Provider]; !ok {
		return nil, fmt.Errorf("provider %q not supported", m.Provider)
	}
	// 只沿用渠道的 api_mode 与自定义请求头；不套用探活用的 body 覆盖，改为放宽输出上限。
	opts := &CheckOptions{
		APIMode:          m.APIMode,
		ExtraHeaders:     m.ExtraHeaders,
		BodyOverrideMode: MonitorBodyOverrideModeMerge,
		BodyOverride:     iqBodyOverride(m.Provider, m.APIMode),
	}
	callCtx, cancel := context.WithTimeout(ctx, iqCandyTimeout)
	defer cancel()
	started := time.Now()
	text, raw, status, err := callProvider(callCtx, m.Provider, m.Endpoint, m.APIKey, model, iqCheckQuestion, opts)
	finished := time.Now()
	if err != nil {
		return nil, sanitizeIQError(err.Error())
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", status, truncateForIQ(sanitizeErrorMessage(raw), 200))
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("empty answer")
	}
	verdict := IQVerdictIncorrect
	if iqAnswerPasses(text) {
		verdict = IQVerdictPassed
	}
	return &ChannelMonitorIQResult{
		MonitorID: m.ID, Model: model, Benchmark: IQBenchmarkCandy, Verdict: verdict, Answer: text,
		DurationMs: finished.Sub(started).Milliseconds(), StartedAt: started, FinishedAt: finished,
	}, nil
}

// iqBodyOverride 放宽输出长度上限，让推理型模型有足够空间作答。
func iqBodyOverride(provider, apiMode string) map[string]any {
	switch {
	case provider == MonitorProviderOpenAI && apiMode == MonitorAPIModeResponses:
		return map[string]any{"max_output_tokens": 4096}
	case provider == MonitorProviderGemini:
		return map[string]any{"generationConfig": map[string]any{"maxOutputTokens": 4096}}
	default:
		return map[string]any{"max_tokens": 4096}
	}
}

func sanitizeIQError(msg string) error {
	return errors.New(truncateForIQ(sanitizeErrorMessage(msg), 300))
}

// ---------- pelican / creative (manxue.ai) ----------

func iqManxueBaseURL(endpoint string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || u.Host == "" {
		return "", errors.New("invalid endpoint")
	}
	if u.Scheme != "https" {
		return "", errors.New("endpoint must be https for manxue.ai")
	}
	if strings.Trim(u.Path, "/") == "" {
		u.Path = "/v1"
	} else {
		u.Path = "/" + strings.Trim(u.Path, "/")
	}
	u.RawQuery, u.Fragment = "", ""
	return u.String(), nil
}

func (s *ChannelMonitorIQService) runManxue(ctx context.Context, m *ChannelMonitor, model, benchmark string) (*ChannelMonitorIQResult, error) {
	if m.Provider != MonitorProviderOpenAI {
		return nil, fmt.Errorf("provider %q not supported by %s", m.Provider, benchmark)
	}
	baseURL, err := iqManxueBaseURL(m.Endpoint)
	if err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(map[string]any{
		"benchmark": benchmark, "base_url": baseURL, "api_key": m.APIKey, "model": model,
	})
	started := time.Now()
	body, err := s.manxueDo(ctx, http.MethodPost, iqManxueEndpoint, payload)
	if err != nil {
		return nil, err
	}
	id := gjson.GetBytes(body, "id").String()
	if !validIQTaskID(id) {
		return nil, errors.New("manxue.ai returned no task id")
	}
	deadline := time.NewTimer(iqManxueMaxWait)
	defer deadline.Stop()
	ticker := time.NewTicker(iqManxuePollInterval)
	defer ticker.Stop()
	for {
		switch gjson.GetBytes(body, "status").String() {
		case "succeeded":
			return parseManxueResult(m.ID, model, benchmark, body, started, time.Now())
		case "failed", "cancelled":
			msg := firstNonEmpty(gjson.GetBytes(body, "error.message").String(), gjson.GetBytes(body, "error").String(), gjson.GetBytes(body, "status").String())
			return nil, fmt.Errorf("manxue.ai task %s: %s", gjson.GetBytes(body, "status").String(), truncateForIQ(msg, 200))
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, errors.New("manxue.ai task timed out")
		case <-ticker.C:
		}
		if body, err = s.manxueDo(ctx, http.MethodGet, iqManxueEndpoint+"/"+url.PathEscape(id), nil); err != nil {
			return nil, err
		}
	}
}

func (s *ChannelMonitorIQService) manxueDo(ctx context.Context, method, endpoint string, payload []byte) ([]byte, error) {
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("manxue.ai request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, iqManxueMaxBodyBytes))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("manxue.ai HTTP %d: %s", resp.StatusCode, truncateForIQ(string(raw), 200))
	}
	return raw, nil
}

func parseManxueResult(monitorID int64, model, benchmark string, body []byte, started, finished time.Time) (*ChannelMonitorIQResult, error) {
	j := string(body)
	html := firstNonEmpty(gjson.Get(j, "result.html").String(), gjson.Get(j, "html").String())
	r := &ChannelMonitorIQResult{
		MonitorID: monitorID, Model: model, Benchmark: benchmark, HTML: html,
		Quality:    gjson.Get(j, "assessment.quality").String(),
		Label:      gjson.Get(j, "assessment.label").String(),
		Reason:     gjson.Get(j, "assessment.reason").String(),
		DurationMs: finished.Sub(started).Milliseconds(), StartedAt: started, FinishedAt: finished,
	}
	if r.Quality == "" && r.Label == "" && html == "" {
		return nil, errors.New("manxue.ai returned an empty result")
	}
	return r, nil
}

func validIQTaskID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
