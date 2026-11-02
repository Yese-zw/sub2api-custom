package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/tidwall/gjson"
)

// 智商检测（糖果测试）：完全在本服务内完成，直接用账号自身的上游地址、密钥和代理
// 请求一道固定的题目，再按固定规则判分，不依赖任何第三方检测服务。
//
// 判分规则：回答中出现独立的 21 即通过（该题的标准答案为 21）。

const iqCheckQuestion = "在一个黑色的袋子里放有三种口味的糖果，每种糖果有两种不同的形状（圆形和五角星形，不同的形状靠手感可以分辨）。" +
	"现已知不同口味的糖和不同形状的数量统计如下表。参赛者需要在活动前决定摸出的糖果数目，那么，最少取出多少个糖果才能保证手中同时拥有不同形状的苹果味和桃子味的糖？" +
	"（同时手中有圆形苹果味匹配五角星桃子味糖果，或者有圆形桃子味匹配五角星苹果味糖果都满足要求）\n" +
	"形状 | 苹果味 | 桃子味 | 西瓜味\n圆形 | 7 | 9 | 8\n五角星形 | 7 | 6 | 4\n" +
	"不许联网，自己计算。只回答最少取出的糖果总数，使用一个整数，不要解释。"

const (
	IQVerdictPassed    = "passed"
	IQVerdictIncorrect = "incorrect"
	IQVerdictError     = "error"

	iqCheckTimeout      = 5 * time.Minute
	iqCheckMaxBodyBytes = 4 << 20
	iqCheckTaskTTL      = time.Hour
)

// IQCheckTask is the state of one IQ check run.
type IQCheckTask struct {
	ID     string `json:"id"`
	Status string `json:"status"` // running | succeeded | failed
	// Verdict is only set once finished: passed | incorrect | error.
	Verdict      string `json:"verdict,omitempty"`
	Answer       string `json:"answer,omitempty"`
	DurationMs   int64  `json:"duration_ms,omitempty"`
	InputTokens  int64  `json:"input_tokens,omitempty"`
	OutputTokens int64  `json:"output_tokens,omitempty"`
	Error        string `json:"error,omitempty"`

	startedAt time.Time
}

var iqCheckTasks = struct {
	sync.Mutex
	m map[string]*IQCheckTask
}{m: map[string]*IQCheckTask{}}

// IQCheckVerdictLabel returns a short Chinese summary for storing in scheduled results.
func IQCheckVerdictLabel(t *IQCheckTask) string {
	switch t.Verdict {
	case IQVerdictPassed:
		return "通过"
	case IQVerdictIncorrect:
		return "未通过（疑似降智）"
	default:
		if t.Error != "" {
			return "检测失败: " + t.Error
		}
		return "检测失败"
	}
}

// StartIQCheck validates the account synchronously, then runs the check in the background.
func (s *AccountTestService) StartIQCheck(ctx context.Context, accountID int64, modelID string) (*IQCheckTask, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, errors.New("account not found")
	}
	req, err := s.buildIQCheckRequest(context.Background(), account, modelID)
	if err != nil {
		return nil, err
	}

	idBytes := make([]byte, 12)
	_, _ = rand.Read(idBytes)
	task := &IQCheckTask{ID: hex.EncodeToString(idBytes), Status: "running", startedAt: time.Now()}

	iqCheckTasks.Lock()
	for id, t := range iqCheckTasks.m {
		if time.Since(t.startedAt) > iqCheckTaskTTL {
			delete(iqCheckTasks.m, id)
		}
	}
	iqCheckTasks.m[task.ID] = task
	iqCheckTasks.Unlock()

	go func() {
		runCtx, cancel := context.WithTimeout(context.Background(), iqCheckTimeout)
		defer cancel()
		result := s.executeIQCheck(runCtx, account, req)
		iqCheckTasks.Lock()
		defer iqCheckTasks.Unlock()
		task.Status, task.Verdict, task.Answer = result.Status, result.Verdict, result.Answer
		task.DurationMs, task.InputTokens, task.OutputTokens = result.DurationMs, result.InputTokens, result.OutputTokens
		task.Error = result.Error
	}()

	return snapshotIQTask(task), nil
}

// GetIQCheck returns the state of a task started by StartIQCheck.
func (s *AccountTestService) GetIQCheck(taskID string) (*IQCheckTask, error) {
	iqCheckTasks.Lock()
	defer iqCheckTasks.Unlock()
	task, ok := iqCheckTasks.m[taskID]
	if !ok {
		return nil, errors.New("task not found or expired")
	}
	return snapshotIQTask(task), nil
}

// RunIQCheck runs the check synchronously (used by scheduled tests).
func (s *AccountTestService) RunIQCheck(ctx context.Context, accountID int64, modelID string) (*IQCheckTask, error) {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, errors.New("account not found")
	}
	req, err := s.buildIQCheckRequest(ctx, account, modelID)
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithTimeout(ctx, iqCheckTimeout)
	defer cancel()
	return s.executeIQCheck(runCtx, account, req), nil
}

func snapshotIQTask(t *IQCheckTask) *IQCheckTask {
	cp := *t
	return &cp
}

// buildIQCheckRequest builds the upstream request for the account's protocol.
// The context passed here is replaced at execution time via req.WithContext.
func (s *AccountTestService) buildIQCheckRequest(ctx context.Context, account *Account, modelID string) (*http.Request, error) {
	if account.Type != AccountTypeAPIKey && account.Type != AccountTypeUpstream {
		return nil, errors.New("智商检测仅支持 API Key / 上游类型账号")
	}
	if account.IsCredentialShadow() {
		return nil, errors.New("智商检测不支持影子账号")
	}

	var apiURL, token string
	var payload map[string]any
	model := strings.TrimSpace(modelID)
	headers := http.Header{"Content-Type": {"application/json"}}

	switch {
	case account.Platform == PlatformAnthropic:
		token = strings.TrimSpace(account.GetCredential("api_key"))
		if token == "" {
			return nil, errors.New("账号未配置 api_key")
		}
		if model == "" {
			model = claude.DefaultTestModel
		}
		model = account.GetMappedModel(model)
		base, err := s.validateUpstreamBaseURL(account.GetBaseURL())
		if err != nil {
			return nil, fmt.Errorf("上游地址无效: %w", err)
		}
		apiURL = strings.TrimSuffix(base, "/") + "/v1/messages"
		headers.Set("anthropic-version", "2023-06-01")
		payload = map[string]any{
			"model":      model,
			"max_tokens": 8192,
			"messages":   []map[string]any{{"role": "user", "content": iqCheckQuestion}},
		}
	case account.IsOpenAI() || account.IsMultiProtocolAPIKey():
		token = strings.TrimSpace(account.GetOpenAIProtocolAPIKey())
		if token == "" {
			return nil, errors.New("账号未配置 api_key")
		}
		if model == "" {
			model = openai.DefaultTestModel
		}
		model = account.GetMappedModel(model)
		baseURL := account.GetOpenAIBaseURL()
		if baseURL == "" {
			baseURL = "https://api.openai.com"
		}
		base, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, fmt.Errorf("上游地址无效: %w", err)
		}
		headers.Set("Authorization", "Bearer "+token)
		if openai_compat.ShouldUseResponsesAPI(account.Extra) {
			apiURL = buildOpenAIResponsesURLForPlatform(account.Platform, base)
			payload = map[string]any{"model": model, "input": iqCheckQuestion, "stream": true}
			if iqModelSupportsReasoning(model) {
				payload["reasoning"] = map[string]any{"effort": "low"}
			}
		} else {
			apiURL = buildOpenAIChatCompletionsURL(base)
			payload = map[string]any{
				"model":    model,
				"stream":   false,
				"messages": []map[string]any{{"role": "user", "content": iqCheckQuestion}},
			}
		}
	default:
		return nil, fmt.Errorf("智商检测暂不支持 %s 平台", account.Platform)
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header[k] = v
	}
	if account.Platform == PlatformAnthropic {
		setAnthropicAPIKeyAuthHeader(req.Header, account, token, account.GetBaseURL())
	} else {
		req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	}
	account.ApplyHeaderOverrides(req.Header)
	return req, nil
}

func iqModelSupportsReasoning(model string) bool {
	m := strings.ToLower(model)
	return !strings.HasPrefix(m, "gpt-4") && !strings.HasPrefix(m, "gpt-3") && !strings.Contains(m, "claude")
}

func (s *AccountTestService) executeIQCheck(ctx context.Context, account *Account, req *http.Request) *IQCheckTask {
	started := time.Now()
	task := &IQCheckTask{Status: "failed", Verdict: IQVerdictError}
	finish := func() *IQCheckTask {
		task.DurationMs = time.Since(started).Milliseconds()
		return task
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	resp, err := s.httpUpstream.DoWithTLS(req.WithContext(ctx), proxyURL, account.ID, account.Concurrency, s.tlsFPProfileService.ResolveTLSProfile(account))
	if err != nil {
		task.Error = "请求上游失败: " + err.Error()
		return finish()
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, iqCheckMaxBodyBytes))
	if resp.StatusCode != http.StatusOK {
		task.Error = fmt.Sprintf("上游返回 HTTP %d: %s", resp.StatusCode, truncateForIQ(string(body), 300))
		return finish()
	}

	answer, in, out, upstreamErr := extractIQAnswer(body)
	task.Answer, task.InputTokens, task.OutputTokens = answer, in, out
	if strings.TrimSpace(answer) == "" {
		task.Error = "上游未返回回答内容"
		if upstreamErr != "" {
			task.Error = upstreamErr
		}
		return finish()
	}
	task.Status = "succeeded"
	if iqAnswerPasses(answer) {
		task.Verdict = IQVerdictPassed
	} else {
		task.Verdict = IQVerdictIncorrect
	}
	return finish()
}

func truncateForIQ(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// iqAnswerPasses reports whether text contains a standalone "21" (not part of a longer
// number such as 121, 210 or 2.1/21.5).
func iqAnswerPasses(text string) bool {
	isDigit := func(b byte) bool { return b >= '0' && b <= '9' }
	for i := 0; i+2 <= len(text); i++ {
		if text[i] != '2' || text[i+1] != '1' {
			continue
		}
		if i > 0 && (isDigit(text[i-1]) || (text[i-1] == '.' && i > 1 && isDigit(text[i-2]))) {
			continue
		}
		if j := i + 2; j < len(text) && (isDigit(text[j]) || (text[j] == '.' && j+1 < len(text) && isDigit(text[j+1]))) {
			continue
		}
		return true
	}
	return false
}

// extractIQAnswer pulls the assistant text (and token usage) out of a Responses,
// Chat Completions or Anthropic Messages reply, streamed (SSE) or not.
func extractIQAnswer(body []byte) (text string, inTokens, outTokens int64, errMsg string) {
	trimmed := bytes.TrimSpace(body)
	// 流式回复可能以 SSE 注释行（": keepalive"）等开头，因此凡不是合法 JSON 且含 data: 行的都按 SSE 解析。
	if !gjson.ValidBytes(trimmed) && bytes.Contains(trimmed, []byte("data:")) {
		var deltas, finalText strings.Builder
		for _, line := range strings.Split(string(trimmed), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" || data == "[DONE]" || !gjson.Valid(data) {
				continue
			}
			switch gjson.Get(data, "type").String() {
			case "response.output_text.delta":
				deltas.WriteString(gjson.Get(data, "delta").String())
			case "response.output_text.done":
				if finalText.Len() == 0 {
					finalText.WriteString(gjson.Get(data, "text").String())
				}
			case "response.completed", "response.done":
				t, i, o, _ := extractIQAnswer([]byte(gjson.Get(data, "response").Raw))
				if finalText.Len() == 0 {
					finalText.WriteString(t)
				}
				inTokens, outTokens = i, o
			case "error", "response.failed":
				errMsg = firstNonEmpty(gjson.Get(data, "error.message").String(), gjson.Get(data, "response.error.message").String(), gjson.Get(data, "message").String())
			default:
				deltas.WriteString(gjson.Get(data, "choices.0.delta.content").String())
			}
		}
		text = deltas.String()
		if text == "" {
			text = finalText.String()
		}
		return
	}

	j := string(trimmed)
	if !gjson.Valid(j) {
		return "", 0, 0, truncateForIQ(j, 300)
	}
	errMsg = gjson.Get(j, "error.message").String()
	inTokens = firstNonZero(gjson.Get(j, "usage.input_tokens").Int(), gjson.Get(j, "usage.prompt_tokens").Int())
	outTokens = firstNonZero(gjson.Get(j, "usage.output_tokens").Int(), gjson.Get(j, "usage.completion_tokens").Int())
	var sb strings.Builder
	if v := gjson.Get(j, "output_text").String(); v != "" {
		sb.WriteString(v)
	}
	if sb.Len() == 0 {
		gjson.Get(j, "output").ForEach(func(_, item gjson.Result) bool {
			item.Get("content").ForEach(func(_, c gjson.Result) bool {
				if c.Get("type").String() == "output_text" || c.Get("text").Exists() {
					sb.WriteString(c.Get("text").String())
				}
				return true
			})
			return true
		})
	}
	if sb.Len() == 0 {
		sb.WriteString(gjson.Get(j, "choices.0.message.content").String())
	}
	if sb.Len() == 0 {
		gjson.Get(j, "content").ForEach(func(_, c gjson.Result) bool {
			if c.Get("type").String() == "text" {
				sb.WriteString(c.Get("text").String())
			}
			return true
		})
	}
	return sb.String(), inTokens, outTokens, errMsg
}

func firstNonZero(vals ...int64) int64 {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}
