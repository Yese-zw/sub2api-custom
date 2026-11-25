package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type requestAuditMemRepo struct {
	mu   sync.Mutex
	logs []*service.RequestAuditLog
}

func (r *requestAuditMemRepo) BatchInsert(_ context.Context, logs []*service.RequestAuditLog) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, logs...)
	return int64(len(logs)), nil
}

func (r *requestAuditMemRepo) FindForUsage(context.Context, service.RequestAuditLookup) (*service.RequestAuditLog, error) {
	return nil, nil
}

func (r *requestAuditMemRepo) DeleteBefore(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}

type requestAuditSettingStub struct {
	service.SettingRepository
	value string
}

func (s *requestAuditSettingStub) GetValue(context.Context, string) (string, error) {
	if s.value == "" {
		return "", service.ErrSettingNotFound
	}
	return s.value, nil
}

func runRequestAudit(t *testing.T, settingsJSON string, body string, handler gin.HandlerFunc) []*service.RequestAuditLog {
	t.Helper()
	return runRequestAuditWith(t, settingsJSON, body, nil, nil, handler)
}

// runRequestAuditWith 可在审计中间件之前插入 pre（模拟 rejectConflictingFields 预读），
// 并自定义请求头。
func runRequestAuditWith(t *testing.T, settingsJSON string, body string, headers map[string]string, pre gin.HandlerFunc, handler gin.HandlerFunc) []*service.RequestAuditLog {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := &requestAuditMemRepo{}
	svc := service.NewRequestAuditService(repo, &requestAuditSettingStub{value: settingsJSON}, nil)
	svc.Start()

	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), ctxkey.ClientRequestID, "cid-1")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	if pre != nil {
		r.Use(pre)
	}
	r.Use(RequestAuditMiddleware(svc))
	r.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{ID: 7, UserID: 9})
		c.Next()
	})
	r.POST("/v1/messages", handler)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-secret-token-value-123456")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	svc.Stop()

	repo.mu.Lock()
	defer repo.mu.Unlock()
	return repo.logs
}

func TestRequestAuditCapturesRequestAndStreamedResponse(t *testing.T) {
	logs := runRequestAudit(t, "", `{"model":"claude","messages":[{"role":"user","content":"hi"}]}`, func(c *gin.Context) {
		raw, _ := io.ReadAll(c.Request.Body)
		require.Contains(t, string(raw), "hi")
		c.Set(opsModelKey, "claude")
		c.Set(opsStreamKey, true)
		c.Header("Content-Type", "text/event-stream")
		c.Status(http.StatusOK)
		_, _ = c.Writer.WriteString("data: {\"delta\":\"hel\"}\n\n")
		c.Writer.Flush()
		_, _ = c.Writer.Write([]byte("data: {\"delta\":\"lo\"}\n\n"))
	})
	require.Len(t, logs, 1)
	l := logs[0]
	require.Equal(t, "cid-1", l.ClientRequestID)
	require.Equal(t, int64(7), *l.APIKeyID)
	require.Equal(t, int64(9), *l.UserID)
	require.Equal(t, "claude", l.Model)
	require.True(t, l.Stream)
	require.Equal(t, http.StatusOK, l.StatusCode)
	require.Contains(t, l.RequestBody, `"content":"hi"`)
	require.Contains(t, l.ResponseBody, `"delta":"hel"`)
	require.Contains(t, l.ResponseBody, `"delta":"lo"`)
	require.False(t, l.RequestTruncated)
	require.NotContains(t, l.RequestHeaders["authorization"], "secret-token-value")
}

func TestRequestAuditTruncatesAndFoldsBase64(t *testing.T) {
	img := strings.Repeat("A", 10000)
	text := strings.Repeat("x", 6000)
	body := `{"image":"data:image/png;base64,` + img + `","text":"` + text + `"}`
	logs := runRequestAudit(t, `{"enabled":true,"retention_days":7,"max_body_bytes":4096}`, body, func(c *gin.Context) {
		_, _ = io.ReadAll(c.Request.Body)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	require.Len(t, logs, 1)
	l := logs[0]
	require.Contains(t, l.RequestBody, "[base64 omitted, 10000 bytes]")
	require.NotContains(t, l.RequestBody, img)
	require.True(t, l.RequestTruncated)
	require.LessOrEqual(t, len(l.RequestBody), 4096)
	require.Equal(t, int64(len(body)), l.RequestBodySize)
}

func TestRequestAuditDisabledSkipsRecording(t *testing.T) {
	logs := runRequestAudit(t, `{"enabled":false,"retention_days":7,"max_body_bytes":4096}`, `{}`, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	require.Empty(t, logs)
}

func TestRequestAuditReferencesPrereadBodyWithoutCopy(t *testing.T) {
	body := `{"model":"gpt","input":"from preread"}`
	var installed *pkghttputil.PrereadBody
	pre := func(c *gin.Context) {
		raw, _ := io.ReadAll(c.Request.Body)
		installed = pkghttputil.NewPrereadBody(raw)
		c.Request.Body = installed
		c.Next()
	}
	logs := runRequestAuditWith(t, "", body, nil, pre, func(c *gin.Context) {
		// 审计中间件不得替换已预读的请求体（下游零拷贝快速路径依赖其类型）。
		require.Same(t, installed, c.Request.Body)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	require.Len(t, logs, 1)
	require.Equal(t, body, logs[0].RequestBody)
	require.Equal(t, int64(len(body)), logs[0].RequestBodySize)
	require.Contains(t, logs[0].ResponseBody, `"ok":true`)
}

func TestRequestAuditCompressedFallbackStoresPlaceholder(t *testing.T) {
	logs := runRequestAuditWith(t, "", "\x28\xb5\x2f\xfd-not-really-zstd", map[string]string{"Content-Encoding": "zstd"}, nil, func(c *gin.Context) {
		_, _ = io.ReadAll(c.Request.Body)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	require.Len(t, logs, 1)
	require.Contains(t, logs[0].RequestBody, "[zstd-encoded body")
}

func TestRequestAuditSealIgnoresLateWrites(t *testing.T) {
	b := &requestAuditBuffer{limit: 8}
	b.writeString("abcd")
	data, total, overflow := b.seal()
	b.write([]byte("late"))
	require.Equal(t, "abcd", string(data))
	require.Equal(t, int64(4), total)
	require.False(t, overflow)
	b.writeString("123456789")
	_, total2, _ := b.seal()
	require.Equal(t, int64(4), total2)
}
