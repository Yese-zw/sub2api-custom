package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	iqCheckEndpoint      = "https://manxue.ai/api/v1/tests"
	iqCheckBenchmark     = "candy"
	iqCheckPollInterval  = 3 * time.Second
	iqCheckMaxWait       = 10 * time.Minute
	iqCheckMaxBodyBytes  = 1 << 20
	iqCheckStatusPassed  = "passed"
	iqCheckStatusFailed  = "incorrect"
	iqCheckStatusError   = "error"
	iqCheckStatusRunning = "running"
)

// IQCheckTask is the normalized state of a candy ("糖果") benchmark task on manxue.ai.
type IQCheckTask struct {
	ID        string `json:"id"`
	Status    string `json:"status"` // running | succeeded | failed | cancelled
	Phase     string `json:"phase,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
	// Verdict is only set once Status is terminal: passed | incorrect | error.
	Verdict      string `json:"verdict,omitempty"`
	Answer       string `json:"answer,omitempty"`
	DurationMs   int64  `json:"duration_ms,omitempty"`
	InputTokens  int64  `json:"input_tokens,omitempty"`
	OutputTokens int64  `json:"output_tokens,omitempty"`
	Error        string `json:"error,omitempty"`
}

// Done reports whether the task reached a terminal state.
func (t *IQCheckTask) Done() bool {
	return t.Status == "succeeded" || t.Status == "failed" || t.Status == "cancelled"
}

// IQCheckService talks to the external 降智检测 service (candy benchmark only).
type IQCheckService struct {
	client   *http.Client
	endpoint string
}

func NewIQCheckService() *IQCheckService {
	return &IQCheckService{client: &http.Client{Timeout: 30 * time.Second}, endpoint: iqCheckEndpoint}
}

// ResolveIQCheckUpstream derives the upstream base_url/api_key the detector should call.
func ResolveIQCheckUpstream(account *Account) (baseURL, apiKey string, err error) {
	if account == nil {
		return "", "", errors.New("account not found")
	}
	if account.Type != AccountTypeAPIKey && account.Type != AccountTypeUpstream {
		return "", "", errors.New("智商检测仅支持 API Key / 上游类型账号")
	}
	apiKey = strings.TrimSpace(account.GetCredential("api_key"))
	if apiKey == "" {
		return "", "", errors.New("账号未配置 api_key")
	}
	baseURL = strings.TrimSpace(account.GetCredential("base_url"))
	if baseURL == "" && account.IsOpenAI() {
		baseURL = "https://api.openai.com"
	}
	if baseURL == "" {
		return "", "", errors.New("账号未配置上游地址 base_url")
	}
	u, perr := url.Parse(baseURL)
	if perr != nil || u.Host == "" {
		return "", "", fmt.Errorf("上游地址无效: %s", baseURL)
	}
	// The detector appends /responses to base_url; default to the conventional /v1 root.
	if p := strings.Trim(u.Path, "/"); p == "" {
		u.Path = "/v1"
	} else {
		u.Path = "/" + p
	}
	u.RawQuery, u.Fragment = "", ""
	return u.String(), apiKey, nil
}

// Start submits a candy benchmark task and returns its id.
func (s *IQCheckService) Start(ctx context.Context, baseURL, apiKey, model string) (*IQCheckTask, error) {
	payload := map[string]any{"benchmark": iqCheckBenchmark, "base_url": baseURL, "api_key": apiKey}
	if m := strings.TrimSpace(model); m != "" {
		payload["model"] = m
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return s.do(req)
}

// Get fetches the current task state.
func (s *IQCheckService) Get(ctx context.Context, id string) (*IQCheckTask, error) {
	if !validIQTaskID(id) {
		return nil, errors.New("invalid task id")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint+"/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}
	return s.do(req)
}

// Run starts a task and blocks until it finishes (used by scheduled tests).
func (s *IQCheckService) Run(ctx context.Context, baseURL, apiKey, model string) (*IQCheckTask, error) {
	task, err := s.Start(ctx, baseURL, apiKey, model)
	if err != nil {
		return nil, err
	}
	deadline := time.NewTimer(iqCheckMaxWait)
	defer deadline.Stop()
	ticker := time.NewTicker(iqCheckPollInterval)
	defer ticker.Stop()
	for !task.Done() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, errors.New("智商检测超时")
		case <-ticker.C:
		}
		next, err := s.Get(ctx, task.ID)
		if err != nil {
			return nil, err
		}
		task = next
	}
	return task, nil
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

func (s *IQCheckService) do(req *http.Request) (*IQCheckTask, error) {
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求智商检测服务失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, iqCheckMaxBodyBytes))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("智商检测服务返回 HTTP %d: %s", resp.StatusCode, truncateIQ(string(raw), 300))
	}
	return parseIQCheckTask(raw)
}

func truncateIQ(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func parseIQCheckTask(raw []byte) (*IQCheckTask, error) {
	var r struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		Phase     string `json:"phase"`
		ExpiresAt string `json:"expires_at"`
		Error     any    `json:"error"`
		Candy     *struct {
			Status       string `json:"status"`
			Answer       string `json:"answer"`
			DurationMs   int64  `json:"duration_ms"`
			InputTokens  int64  `json:"input_tokens"`
			OutputTokens int64  `json:"output_tokens"`
			Error        any    `json:"error"`
		} `json:"candy"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("解析智商检测响应失败: %w", err)
	}
	t := &IQCheckTask{ID: r.ID, Status: r.Status, Phase: r.Phase, ExpiresAt: r.ExpiresAt}
	if t.Status == "" {
		t.Status = iqCheckStatusRunning
	}
	t.Error = iqErrString(r.Error)
	if r.Candy != nil {
		t.Verdict = r.Candy.Status
		t.Answer = r.Candy.Answer
		t.DurationMs = r.Candy.DurationMs
		t.InputTokens = r.Candy.InputTokens
		t.OutputTokens = r.Candy.OutputTokens
		if e := iqErrString(r.Candy.Error); e != "" && t.Error == "" {
			t.Error = e
		}
	}
	if t.Done() && t.Verdict == "" {
		// Terminal without a candy verdict: treat as detection error.
		t.Verdict = iqCheckStatusError
	}
	if !t.Done() {
		t.Verdict = ""
	}
	return t, nil
}

func iqErrString(v any) string {
	switch e := v.(type) {
	case nil:
		return ""
	case string:
		return e
	case map[string]any:
		if m, ok := e["message"].(string); ok {
			return m
		}
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// IQCheckVerdictLabel returns a short Chinese summary for storing in scheduled results.
func IQCheckVerdictLabel(t *IQCheckTask) string {
	switch t.Verdict {
	case iqCheckStatusPassed:
		return "通过"
	case iqCheckStatusFailed:
		return "未通过（疑似降智）"
	default:
		if t.Error != "" {
			return "检测失败: " + t.Error
		}
		return "检测失败"
	}
}
