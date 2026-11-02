package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// 智商检测的长输出请求（尤其是鹈鹕测试要模型一次写完整个 HTML 动画）如果走非流式，
// 上游要生成完才会返回响应头，经 Cloudflare 等代理时会在 100s 左右被掐断（HTTP 504 / 524）。
// 这里改成流式读取：响应头立刻返回，数据持续到达，不会被代理判定为超时。

const (
	iqStreamMaxTextBytes = 4 << 20 // 累计回答文本上限
	iqStreamLineBytes    = 4 << 20 // 单行 SSE 上限
)

// streamProviderText 以流式方式请求上游并返回完整回答文本。
// 与 callProvider 的返回约定一致：上游返回非 2xx 时 status 为该状态码、rawBody 为响应体，err 为 nil。
func streamProviderText(ctx context.Context, provider, endpoint, apiKey, model, prompt string, opts *CheckOptions) (text, rawBody string, status int, err error) {
	requestedAPIMode := checkAPIMode(opts)
	if err := validateAPIMode(provider, requestedAPIMode); err != nil {
		return "", "", 0, err
	}
	adapter, apiMode, ok := providerAdapterFor(provider, requestedAPIMode)
	if !ok {
		return "", "", 0, fmt.Errorf("unsupported provider %q", provider)
	}
	body, err := buildRequestBody(adapter, provider, apiMode, model, prompt, opts)
	if err != nil {
		return "", "", 0, err
	}
	path := monitorRequestPath(provider, endpoint, adapter, model)
	if provider == MonitorProviderGemini {
		// Gemini 的流式是另一个方法名，且需要 alt=sse 才返回 SSE。
		path = strings.Replace(path, ":generateContent", ":streamGenerateContent", 1) + "?alt=sse"
	} else if body, err = sjson.SetBytes(body, "stream", true); err != nil {
		return "", "", 0, fmt.Errorf("set stream: %w", err)
	}
	headers := mergeHeaders(adapter.buildHeaders(apiKey), opts)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(endpoint, path), strings.NewReader(string(body)))
	if err != nil {
		return "", "", 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := monitorHTTPClient
	if opts != nil && opts.HTTPClient != nil {
		client = opts.HTTPClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, monitorResponseMaxBytes))
		return "", string(raw), resp.StatusCode, nil
	}

	// 上游忽略了 stream 参数、直接返回完整 JSON：按非流式解析。
	if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, iqStreamMaxTextBytes))
		if readErr != nil {
			return "", "", resp.StatusCode, fmt.Errorf("read body: %w", readErr)
		}
		if provider == MonitorProviderOpenAI && apiMode == MonitorAPIModeResponses {
			return extractOpenAIResponsesText(raw), string(raw), resp.StatusCode, nil
		}
		return extractMonitorResponseText(adapter, raw), string(raw), resp.StatusCode, nil
	}

	text, err = readSSEText(resp.Body, provider, apiMode)
	return text, "", resp.StatusCode, err
}

// readSSEText 逐行读取 SSE，累计各家格式里的文本增量（忽略思考内容）。
func readSSEText(r io.Reader, provider, apiMode string) (string, error) {
	var sb strings.Builder
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), iqStreamLineBytes)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		if !gjson.Valid(data) {
			continue
		}
		if msg := gjson.Get(data, "error.message"); msg.Exists() && msg.String() != "" {
			return sb.String(), fmt.Errorf("upstream stream error: %s", truncateForIQ(msg.String(), 200))
		}
		sb.WriteString(sseDeltaText(data, provider, apiMode))
		if sb.Len() > iqStreamMaxTextBytes {
			return sb.String(), errors.New("answer too large")
		}
	}
	if err := sc.Err(); err != nil {
		return sb.String(), fmt.Errorf("read stream: %w", err)
	}
	return sb.String(), nil
}

// sseDeltaText 取出单个 SSE 事件里的文本增量。
func sseDeltaText(data, provider, apiMode string) string {
	switch {
	case provider == MonitorProviderAnthropic:
		if gjson.Get(data, "type").String() == "content_block_delta" && gjson.Get(data, "delta.type").String() == "text_delta" {
			return gjson.Get(data, "delta.text").String()
		}
	case provider == MonitorProviderGemini:
		var sb strings.Builder
		gjson.Get(data, "candidates.0.content.parts").ForEach(func(_, part gjson.Result) bool {
			if !part.Get("thought").Bool() {
				sb.WriteString(part.Get("text").String())
			}
			return true
		})
		return sb.String()
	case provider == MonitorProviderOpenAI && apiMode == MonitorAPIModeResponses:
		if gjson.Get(data, "type").String() == "response.output_text.delta" {
			return gjson.Get(data, "delta").String()
		}
	default: // OpenAI 兼容 Chat Completions（含 Grok / Kimi / 智谱 / DeepSeek / MiniMax）
		return gjson.Get(data, "choices.0.delta.content").String()
	}
	return ""
}
