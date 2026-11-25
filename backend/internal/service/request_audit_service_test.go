package service

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type requestAuditSlowSettingRepo struct {
	SettingRepository
	calls atomic.Int32
	value atomic.Value
	delay time.Duration
}

func (r *requestAuditSlowSettingRepo) GetValue(context.Context, string) (string, error) {
	r.calls.Add(1)
	time.Sleep(r.delay)
	return r.value.Load().(string), nil
}

func TestRequestAuditSettingsStaleWhileRefresh(t *testing.T) {
	repo := &requestAuditSlowSettingRepo{delay: 200 * time.Millisecond}
	repo.value.Store(`{"enabled":true,"retention_days":7,"max_body_bytes":4096}`)
	svc := NewRequestAuditService(nil, repo, nil)

	require.True(t, svc.Settings(context.Background()).Enabled) // 首次同步加载
	require.Equal(t, int32(1), repo.calls.Load())

	// 人为让缓存过期，并改库中值。
	cached := svc.cache.Load()
	svc.cache.Store(&requestAuditSettingsCache{settings: cached.settings, loadedAt: time.Now().Add(-time.Hour)})
	repo.value.Store(`{"enabled":false,"retention_days":7,"max_body_bytes":4096}`)

	start := time.Now()
	for i := 0; i < 50; i++ {
		require.True(t, svc.Settings(context.Background()).Enabled, "stale value returned while refreshing")
	}
	require.Less(t, time.Since(start), 100*time.Millisecond, "hot path must not wait for the database")

	require.Eventually(t, func() bool { return !svc.Settings(context.Background()).Enabled }, 2*time.Second, 20*time.Millisecond)
	require.Equal(t, int32(2), repo.calls.Load(), "only one background refresh")
}

func TestRenderRequestAuditCaptureFoldsAndDoesNotMutateInput(t *testing.T) {
	b64 := strings.Repeat("Q", 5000)
	req := []byte(`{"messages":[{"content":"data:image/png;base64,` + b64 + `"}]}`)
	orig := string(req)
	resp := []byte(`{"data":[{"b64_json":"` + b64 + `"}]}`)
	log := renderRequestAuditCapture(&RequestAuditCapture{
		Log:          &RequestAuditLog{RequestContentType: "application/json"},
		RequestRaw:   req,
		ResponseRaw:  resp,
		MaxBodyBytes: 4096,
	})
	require.Equal(t, orig, string(req), "PrereadBody bytes are shared and must never be mutated")
	require.Contains(t, log.RequestBody, "data:image/png;base64,[base64 omitted, 5000 bytes]")
	require.Contains(t, log.ResponseBody, `"b64_json":"[base64 omitted, 5000 bytes]"`)
	require.False(t, log.RequestTruncated)
}
