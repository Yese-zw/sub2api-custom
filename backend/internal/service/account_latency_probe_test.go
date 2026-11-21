package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// recordingAccountRepo 记录写调用；只读包装后不应收到任何写操作。
type recordingAccountRepo struct {
	AccountRepository
	writes []string
}

func (r *recordingAccountRepo) GetByID(_ context.Context, id int64) (*Account, error) {
	return &Account{ID: id}, nil
}
func (r *recordingAccountRepo) SetError(context.Context, int64, string) error {
	r.writes = append(r.writes, "SetError")
	return nil
}
func (r *recordingAccountRepo) SetRateLimited(context.Context, int64, time.Time) error {
	r.writes = append(r.writes, "SetRateLimited")
	return nil
}
func (r *recordingAccountRepo) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	r.writes = append(r.writes, "SetTempUnschedulable")
	return nil
}
func (r *recordingAccountRepo) ClearError(context.Context, int64) error {
	r.writes = append(r.writes, "ClearError")
	return nil
}
func (r *recordingAccountRepo) UpdateExtra(context.Context, int64, map[string]any) error {
	r.writes = append(r.writes, "UpdateExtra")
	return nil
}

func TestReadOnlyAccountRepositoryIgnoresWrites(t *testing.T) {
	inner := &recordingAccountRepo{}
	ro := NewReadOnlyAccountRepository(inner)
	ctx := context.Background()

	acc, err := ro.GetByID(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, int64(7), acc.ID, "reads pass through")

	require.NoError(t, ro.SetError(ctx, 7, "x"))
	require.NoError(t, ro.SetRateLimited(ctx, 7, time.Now()))
	require.NoError(t, ro.SetTempUnschedulable(ctx, 7, time.Now(), "x"))
	require.NoError(t, ro.ClearError(ctx, 7))
	require.NoError(t, ro.UpdateExtra(ctx, 7, map[string]any{"a": 1}))
	require.Empty(t, inner.writes)

	require.Same(t, ro, NewReadOnlyAccountRepository(ro), "no double wrapping")
	require.NotNil(t, (&AccountTestService{accountRepo: inner}).ReadOnlyClone())
	require.Nil(t, (*AccountTestService)(nil).ReadOnlyClone())
}

func TestParseLatencyProbeOutput(t *testing.T) {
	ok := parseLatencyProbeOutput("data: {\"type\":\"test_start\",\"model\":\"m\"}\n\n" +
		"data: {\"type\":\"content\",\"text\":\"Hello\"}\n\n" +
		"data: {\"type\":\"test_complete\",\"success\":true,\"token_latency_ms\":412,\"total_latency_ms\":980}\n\n")
	require.True(t, ok.Success)
	require.Equal(t, int64(412), *ok.FirstTokenMs)
	require.Equal(t, int64(980), ok.TotalMs)

	failed := parseLatencyProbeOutput("data: {\"type\":\"error\",\"error\":\"API returned 429\",\"total_latency_ms\":300}\n\n")
	require.False(t, failed.Success)
	require.Equal(t, "API returned 429", failed.Error)
	require.Equal(t, int64(300), failed.TotalMs)
}

func TestApplyLatencyProbe(t *testing.T) {
	ft := int64(350)
	r := &ScheduledTestResult{IQMessage: "通过", FirstTokenMs: ptrI(9999)}
	applyLatencyProbe(r, &AccountLatencyProbeResult{Success: true, FirstTokenMs: &ft, TotalMs: 900})
	require.Equal(t, int64(350), *r.FirstTokenMs, "first token comes from the probe, not the IQ question")
	require.Equal(t, int64(900), *r.ProbeTotalMs)
	require.Equal(t, "通过", r.IQMessage)

	r = &ScheduledTestResult{IQMessage: "通过", FirstTokenMs: ptrI(9999)}
	applyLatencyProbe(r, &AccountLatencyProbeResult{Success: false, TotalMs: 120, Error: "API returned 401"})
	require.Nil(t, r.FirstTokenMs)
	require.Equal(t, int64(120), *r.ProbeTotalMs)
	require.Contains(t, r.IQMessage, "测速失败：API returned 401")
}
