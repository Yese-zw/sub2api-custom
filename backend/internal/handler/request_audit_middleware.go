package handler

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// 兜底抓取（请求体未被预读为 PrereadBody 时）的原始上限 = 入库上限 × 该倍数，
// 给 base64 折叠留余量，同时保证单请求内存占用有界。
const (
	requestAuditFallbackCaptureFactor = 4
	requestAuditFallbackCaptureMax    = 64 << 20
)

var requestAuditSensitiveHeaders = map[string]bool{
	"authorization":       true,
	"proxy-authorization": true,
	"x-api-key":           true,
	"x-goog-api-key":      true,
	"api-key":             true,
	"cookie":              true,
}

// RequestAuditMiddleware 抓取网关请求的完整请求体与返回给客户端的响应体，交给审计服务异步落库。
//
// 挂载位置在 rejectConflictingFields 之后：此时 POST/PUT/PATCH 请求体已被读取、
// 解压并回填为不可变的 PrereadBody，这里直接引用其切片（零拷贝），不再复制请求体。
// 请求路径上只做引用与移交；解析/渲染在审计写入协程完成，不推迟响应结束。
// svc 为 nil 或审计关闭时直通。
func RequestAuditMiddleware(svc *service.RequestAuditService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil || c.Request == nil {
			c.Next()
			return
		}
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		// WebSocket 帧不经过 ResponseWriter.Write，无法抓取。
		if strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
			c.Next()
			return
		}
		settings := svc.Settings(c.Request.Context())
		if !settings.Enabled {
			c.Next()
			return
		}

		start := time.Now()
		capture := &service.RequestAuditCapture{MaxBodyBytes: settings.MaxBodyBytes}

		var fallback *requestAuditBuffer
		switch body := c.Request.Body.(type) {
		case nil:
		case *pkghttputil.PrereadBody:
			capture.RequestRaw = body.Bytes()
		default:
			if body != http.NoBody {
				limit := settings.MaxBodyBytes * requestAuditFallbackCaptureFactor
				if limit > requestAuditFallbackCaptureMax {
					limit = requestAuditFallbackCaptureMax
				}
				fallback = &requestAuditBuffer{limit: limit}
				capture.RequestEncoding = c.Request.Header.Get("Content-Encoding")
				c.Request.Body = &requestAuditReadCloser{ReadCloser: body, buf: fallback}
			}
		}
		requestContentType := c.Request.Header.Get("Content-Type")

		originalWriter := c.Writer
		w := &requestAuditWriter{ResponseWriter: originalWriter, buf: &requestAuditBuffer{limit: settings.MaxBodyBytes}}
		c.Writer = w
		defer func() {
			if c.Writer == w {
				c.Writer = originalWriter
			}
		}()

		c.Next()

		// 封存缓冲区：此后（如迟到的 keepalive 写）不再追加，切片可直接移交，无需复制。
		respRaw, respTotal, respOverflow := w.buf.seal()
		var reqTotal int64
		if fallback != nil {
			capture.RequestRaw, reqTotal, capture.RequestOverflow = fallback.seal()
		} else {
			reqTotal = int64(len(capture.RequestRaw))
		}

		apiKey := getOpsAPIKey(c)
		if apiKey == nil {
			return
		}
		ctx := c.Request.Context()
		entry := &service.RequestAuditLog{
			CreatedAt:           start.UTC(),
			Method:              c.Request.Method,
			Path:                c.Request.URL.Path,
			StatusCode:          w.Status(),
			DurationMs:          time.Since(start).Milliseconds(),
			ClientIP:            strings.TrimSpace(ip.GetClientIP(c)),
			UserAgent:           c.Request.UserAgent(),
			RequestHeaders:      requestAuditHeaders(c.Request.Header),
			RequestContentType:  requestContentType,
			RequestBodySize:     reqTotal,
			ResponseContentType: w.Header().Get("Content-Type"),
			ResponseBodySize:    respTotal,
		}
		entry.RequestID, _ = ctx.Value(ctxkey.RequestID).(string)
		entry.ClientRequestID, _ = ctx.Value(ctxkey.ClientRequestID).(string)
		apiKeyID := apiKey.ID
		entry.APIKeyID = &apiKeyID
		if apiKey.UserID > 0 {
			userID := apiKey.UserID
			entry.UserID = &userID
		}
		if v, ok := c.Get(opsModelKey); ok {
			entry.Model, _ = v.(string)
		}
		if v, ok := c.Get(opsStreamKey); ok {
			entry.Stream, _ = v.(bool)
		}
		if v, ok := c.Get(opsAccountIDKey); ok {
			if id, ok := v.(int64); ok && id > 0 {
				entry.AccountID = &id
			}
		}

		capture.Log = entry
		capture.ResponseRaw = respRaw
		capture.ResponseOverflow = respOverflow
		capture.ResponseEncoding = w.Header().Get("Content-Encoding")
		svc.Record(capture)
	}
}

// RequestAuditFromHandlers 从 Handlers 取审计服务，测试里未装配时返回 nil。
func RequestAuditFromHandlers(h *Handlers) *service.RequestAuditService {
	if h == nil || h.Admin == nil || h.Admin.RequestAudit == nil {
		return nil
	}
	return h.Admin.RequestAudit.Service()
}

func requestAuditHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, values := range h {
		lk := strings.ToLower(k)
		v := strings.Join(values, ", ")
		if requestAuditSensitiveHeaders[lk] {
			v = service.MaskAuditCredential(v)
		}
		if len(v) > 1024 {
			v = v[:1024] + "…"
		}
		out[lk] = v
	}
	return out
}

// requestAuditBuffer 有界抓取缓冲区：超过 limit 的数据只计数不保存；seal 后不再追加。
type requestAuditBuffer struct {
	mu       sync.Mutex
	limit    int
	data     []byte
	total    int64
	overflow bool
	sealed   bool
}

func (b *requestAuditBuffer) write(p []byte) {
	if len(p) == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if n := b.reserve(len(p)); n > 0 {
		b.data = append(b.data, p[:n]...)
	}
}

func (b *requestAuditBuffer) writeString(s string) {
	if len(s) == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if n := b.reserve(len(s)); n > 0 {
		b.data = append(b.data, s[:n]...)
	}
}

// reserve 记账并返回本次可保存的字节数（调用方持锁）。
func (b *requestAuditBuffer) reserve(n int) int {
	if b.sealed {
		return 0
	}
	b.total += int64(n)
	room := b.limit - len(b.data)
	if room <= 0 {
		b.overflow = true
		return 0
	}
	if n > room {
		b.overflow = true
		return room
	}
	return n
}

// seal 停止追加并移交内部切片（不复制）。
func (b *requestAuditBuffer) seal() ([]byte, int64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sealed = true
	data := b.data
	b.data = nil
	return data, b.total, b.overflow
}

type requestAuditReadCloser struct {
	io.ReadCloser
	buf *requestAuditBuffer
}

func (r *requestAuditReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 {
		r.buf.write(p[:n])
	}
	return n, err
}

type requestAuditWriter struct {
	gin.ResponseWriter
	buf *requestAuditBuffer
}

func (w *requestAuditWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	if n > 0 {
		w.buf.write(p[:n])
	}
	return n, err
}

func (w *requestAuditWriter) WriteString(s string) (int, error) {
	n, err := w.ResponseWriter.WriteString(s)
	if n > 0 {
		w.buf.writeString(s[:n])
	}
	return n, err
}

func (w *requestAuditWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
