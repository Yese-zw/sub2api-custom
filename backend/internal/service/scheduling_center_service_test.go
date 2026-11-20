package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeSchedulingCenterRepo 是单分组场景下的内存实现。
type fakeSchedulingCenterRepo struct {
	members       map[int64]SchedulingCenterPoolMember
	states        map[int64]*SchedulingCenterAccountState
	events        []SchedulingCenterEvent
	stats         map[int64]SchedulingCenterAccountStats
	iq            map[int64]SchedulingCenterIQResult
	iqHistory     map[int64][]string
	testTTFT      map[int64]float64
	requests      map[int64][]SchedulingCenterRequestSample
	settings      []SchedulingCenterGroupSettings
	groupAccounts map[int64][]int64
	managed       []SchedulingCenterManagedPlan
	inserted      []SchedulingCenterManagedPlan
	now           func() time.Time
}

func newFakeSchedulingCenterRepo() *fakeSchedulingCenterRepo {
	return &fakeSchedulingCenterRepo{
		members:       map[int64]SchedulingCenterPoolMember{},
		states:        map[int64]*SchedulingCenterAccountState{},
		groupAccounts: map[int64][]int64{},
		now:           time.Now,
	}
}

func (f *fakeSchedulingCenterRepo) inGroup(groupID, accountID int64) bool {
	for _, id := range f.groupAccounts[groupID] {
		if id == accountID {
			return true
		}
	}
	return false
}

func (f *fakeSchedulingCenterRepo) ListPoolMembers(_ context.Context, groupID int64) ([]SchedulingCenterPoolMember, error) {
	out := []SchedulingCenterPoolMember{}
	for _, m := range f.members {
		if f.inGroup(groupID, m.AccountID) {
			out = append(out, m)
		}
	}
	return out, nil
}
func (f *fakeSchedulingCenterRepo) AddToSchedulingPool(_ context.Context, groupID int64, ids []int64, source string) ([]int64, error) {
	out := []int64{}
	for _, id := range ids {
		if !f.inGroup(groupID, id) {
			continue
		}
		if m, ok := f.members[id]; ok {
			if source == SchedulingCenterSourceManual {
				m.Source = source
				f.members[id] = m
				out = append(out, id)
			}
			continue
		}
		f.members[id] = SchedulingCenterPoolMember{AccountID: id, Source: source, EnteredAt: f.now()}
		out = append(out, id)
	}
	return out, nil
}
func (f *fakeSchedulingCenterRepo) RemoveFromSchedulingPool(_ context.Context, _ int64, ids []int64) ([]int64, error) {
	out := []int64{}
	for _, id := range ids {
		if _, ok := f.members[id]; ok {
			delete(f.members, id)
			out = append(out, id)
		}
	}
	return out, nil
}
func (f *fakeSchedulingCenterRepo) ListAccountStates(context.Context, int64) ([]SchedulingCenterAccountState, error) {
	out := []SchedulingCenterAccountState{}
	for _, st := range f.states {
		out = append(out, *st)
	}
	return out, nil
}
func (f *fakeSchedulingCenterRepo) GetAccountState(_ context.Context, _ int64, accountID int64) (*SchedulingCenterAccountState, error) {
	st, ok := f.states[accountID]
	if !ok {
		return nil, nil
	}
	cp := *st
	return &cp, nil
}
func (f *fakeSchedulingCenterRepo) UpsertAccountState(_ context.Context, st *SchedulingCenterAccountState) error {
	cp := *st
	f.states[st.AccountID] = &cp
	return nil
}
func (f *fakeSchedulingCenterRepo) CompareAndSetProbingState(_ context.Context, st *SchedulingCenterAccountState, expect time.Time) (bool, error) {
	cur, ok := f.states[st.AccountID]
	if !ok || cur.State != SchedulingCenterStateProbing || cur.ProbeStartedAt == nil || !cur.ProbeStartedAt.Equal(expect) {
		return false, nil
	}
	cp := *st
	f.states[st.AccountID] = &cp
	return true, nil
}
func (f *fakeSchedulingCenterRepo) DeleteAccountStates(_ context.Context, _ int64, ids []int64) error {
	for _, id := range ids {
		delete(f.states, id)
	}
	return nil
}
func (f *fakeSchedulingCenterRepo) InsertEvents(_ context.Context, events []SchedulingCenterEvent) error {
	f.events = append(f.events, events...)
	return nil
}
func (f *fakeSchedulingCenterRepo) ListEvents(context.Context, int64, string, int, int) ([]SchedulingCenterEvent, int64, error) {
	return f.events, int64(len(f.events)), nil
}
func (f *fakeSchedulingCenterRepo) CountBadEvents(_ context.Context, _ int64, ids []int64, since time.Time) (map[int64]int, error) {
	out := map[int64]int{}
	for _, e := range f.events {
		if e.IsBad && !e.CreatedAt.Before(since) {
			for _, id := range ids {
				if id == e.AccountID {
					out[id]++
				}
			}
		}
	}
	return out, nil
}
func (f *fakeSchedulingCenterRepo) GetRecentRequests(_ context.Context, since map[int64]time.Time, limit int) (map[int64][]SchedulingCenterRequestSample, error) {
	out := map[int64][]SchedulingCenterRequestSample{}
	for id, t := range since {
		for _, r := range f.requests[id] { // 新 → 旧
			if r.At.Before(t) || len(out[id]) >= limit {
				continue
			}
			out[id] = append(out[id], r)
		}
	}
	return out, nil
}
func (f *fakeSchedulingCenterRepo) GetRecentIQStatuses(_ context.Context, since map[int64]time.Time, limit int) (map[int64][]string, error) {
	out := map[int64][]string{}
	for id := range since {
		h := f.iqHistory[id]
		if len(h) > limit {
			h = h[:limit]
		}
		out[id] = h
	}
	return out, nil
}
func (f *fakeSchedulingCenterRepo) GetManagedPlanID(context.Context, int64) (int64, bool, error) {
	return 0, false, nil
}
func (f *fakeSchedulingCenterRepo) GetAccountStats(context.Context, []int64, time.Time) (map[int64]SchedulingCenterAccountStats, error) {
	return f.stats, nil
}
func (f *fakeSchedulingCenterRepo) GetLatestIQResults(context.Context, []int64) (map[int64]SchedulingCenterIQResult, error) {
	return f.iq, nil
}
func (f *fakeSchedulingCenterRepo) GetRecentTestFirstTokenAvg(context.Context, []int64, int) (map[int64]float64, error) {
	return f.testTTFT, nil
}
func (f *fakeSchedulingCenterRepo) ListAccountTestRecords(context.Context, int64, int) ([]SchedulingCenterTestRecord, error) {
	return nil, nil
}
func (f *fakeSchedulingCenterRepo) ListGroupSettings(context.Context) ([]SchedulingCenterGroupSettings, error) {
	return f.settings, nil
}
func (f *fakeSchedulingCenterRepo) GetGroupSettings(context.Context, int64) (*SchedulingCenterGroupSettings, error) {
	return nil, nil
}
func (f *fakeSchedulingCenterRepo) UpsertGroupSettings(_ context.Context, in *SchedulingCenterGroupSettings) (*SchedulingCenterGroupSettings, error) {
	return in, nil
}
func (f *fakeSchedulingCenterRepo) ListGroupAccountIDs(_ context.Context, groupID int64) ([]int64, error) {
	return f.groupAccounts[groupID], nil
}
func (f *fakeSchedulingCenterRepo) ListManagedPlans(context.Context) ([]SchedulingCenterManagedPlan, error) {
	return f.managed, nil
}
func (f *fakeSchedulingCenterRepo) InsertManagedPlan(_ context.Context, accountID, planID, groupID int64) error {
	f.inserted = append(f.inserted, SchedulingCenterManagedPlan{AccountID: accountID, PlanID: planID, GroupID: groupID})
	return nil
}
func (f *fakeSchedulingCenterRepo) UpdateManagedPlanGroup(context.Context, int64, int64) error {
	return nil
}

func ptrF(v float64) *float64 { return &v }
func ptrI(v int64) *int64     { return &v }
func ptrInt(v int) *int       { return &v }

// testClock 是可手动推进的时钟。
type testClock struct{ t time.Time }

func (c *testClock) now() time.Time          { return c.t }
func (c *testClock) advance(d time.Duration) { c.t = c.t.Add(d) }
func newTestClock() *testClock               { return &testClock{t: time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)} }
func okReq(at time.Time, ttft int64) SchedulingCenterRequestSample {
	return SchedulingCenterRequestSample{At: at, OK: true, FirstTokenMs: &ttft}
}
func errReq(at time.Time) SchedulingCenterRequestSample { return SchedulingCenterRequestSample{At: at} }

func newLifecycleService(repo *fakeSchedulingCenterRepo, clock *testClock, probePass *bool) *SchedulingCenterService {
	svc := NewSchedulingCenterService(repo, nil, nil, nil)
	svc.now = clock.now
	repo.now = clock.now
	svc.probeSync = true
	svc.probe = func(context.Context, int64, SchedulingCenterGroupSettings) (bool, map[string]any) {
		return *probePass, map[string]any{"kind": "iq"}
	}
	return svc
}

func lastEvent(repo *fakeSchedulingCenterRepo) SchedulingCenterEvent {
	return repo.events[len(repo.events)-1]
}

// ---------------------------------------------------------------------------

func TestSchedulingCenterBackoff(t *testing.T) {
	def := []int{5, 15, 30, 60}
	require.Equal(t, 5*time.Minute, SchedulingCenterBackoff(def, 1))
	require.Equal(t, 15*time.Minute, SchedulingCenterBackoff(def, 2))
	require.Equal(t, 30*time.Minute, SchedulingCenterBackoff(def, 3))
	require.Equal(t, time.Hour, SchedulingCenterBackoff(def, 4))
	require.Equal(t, time.Hour, SchedulingCenterBackoff(def, 9))
	require.Equal(t, 5*time.Minute, SchedulingCenterBackoff(nil, 1), "empty steps fall back to defaults")

	custom := []int{2, 120}
	require.Equal(t, 2*time.Minute, SchedulingCenterBackoff(custom, 1))
	require.Equal(t, 120*time.Minute, SchedulingCenterBackoff(custom, 2))
	require.Equal(t, 120*time.Minute, SchedulingCenterBackoff(custom, 5), "last step is the cap")
}

func TestSchedulingCenterCheckExit(t *testing.T) {
	gs := defaultSchedulingCenterGroupSettings(1)
	gs.MinCacheRate = ptrF(60)
	now := time.Now()

	require.Nil(t, SchedulingCenterCheckExit(gs, nil, nil, nil), "no data never exits")

	// 缓存率：60 − 20 = 40
	low := &SchedulingCenterAccountMetrics{CacheRate: ptrF(35), Requests: 25}
	dec := SchedulingCenterCheckExit(gs, low, nil, nil)
	require.Equal(t, SchedulingCenterReasonExitCache, dec.Reason)
	require.Equal(t, 40.0, dec.Detail["threshold"])
	fewReq := &SchedulingCenterAccountMetrics{CacheRate: ptrF(35), Requests: 5}
	require.Nil(t, SchedulingCenterCheckExit(gs, fewReq, nil, nil), "below min requests")
	noEntry := defaultSchedulingCenterGroupSettings(1)
	require.Nil(t, SchedulingCenterCheckExit(noEntry, low, nil, nil), "cache exit needs entry threshold")

	// 降智：最近 10 次中 > 3 次
	require.Nil(t, SchedulingCenterCheckExit(gs, nil, nil, []string{"incorrect", "incorrect", "incorrect", "passed"}))
	dec = SchedulingCenterCheckExit(gs, nil, nil, []string{"incorrect", "incorrect", "passed", "incorrect", "incorrect"})
	require.Equal(t, SchedulingCenterReasonExitIQ, dec.Reason)
	require.Equal(t, 4, dec.Detail["degraded"])

	// 延迟：最近连续 3 次流式首 Token > 10s（非流式请求跳过，错误请求跳过）
	nonStream := SchedulingCenterRequestSample{At: now, OK: true}
	slow := []SchedulingCenterRequestSample{okReq(now, 12000), nonStream, okReq(now, 15000), errReq(now), okReq(now, 11000)}
	require.Equal(t, SchedulingCenterReasonExitLatency, SchedulingCenterCheckExit(gs, nil, slow, nil).Reason)
	mixed := []SchedulingCenterRequestSample{okReq(now, 12000), okReq(now, 900), okReq(now, 15000)}
	require.Nil(t, SchedulingCenterCheckExit(gs, nil, mixed, nil))
	require.Nil(t, SchedulingCenterCheckExit(gs, nil, []SchedulingCenterRequestSample{okReq(now, 12000), okReq(now, 12000)}, nil), "not enough samples")

	// 错误：最近连续 3 次都是上游错误
	errs := []SchedulingCenterRequestSample{errReq(now), errReq(now), errReq(now), okReq(now, 100)}
	require.Equal(t, SchedulingCenterReasonExitError, SchedulingCenterCheckExit(gs, nil, errs, nil).Reason)
	require.Nil(t, SchedulingCenterCheckExit(gs, nil, []SchedulingCenterRequestSample{errReq(now), okReq(now, 100), errReq(now), errReq(now)}, nil))
}

func TestSchedulingCenterRecoverVerdict(t *testing.T) {
	gs := defaultSchedulingCenterGroupSettings(1)
	now := time.Now()
	ok, reason, _ := schedulingCenterRecoverVerdict(gs, []SchedulingCenterRequestSample{okReq(now, 900), okReq(now, 800)})
	require.Equal(t, 2, ok)
	require.Empty(t, reason)

	// 新 → 旧：最旧一次正常，之后出错
	_, reason, _ = schedulingCenterRecoverVerdict(gs, []SchedulingCenterRequestSample{okReq(now, 900), errReq(now), okReq(now, 800)})
	require.Equal(t, SchedulingCenterReasonRecoverError, reason)
	_, reason, _ = schedulingCenterRecoverVerdict(gs, []SchedulingCenterRequestSample{okReq(now, 20000)})
	require.Equal(t, SchedulingCenterReasonRecoverLatency, reason)
}

func TestSchedulingCenterLifecycle_ExitBackoffProbeRecover(t *testing.T) {
	clock := newTestClock()
	repo := newFakeSchedulingCenterRepo()
	probePass := true
	svc := newLifecycleService(repo, clock, &probePass)
	ctx := context.Background()

	gs := defaultSchedulingCenterGroupSettings(1)
	gs.Enabled = true
	gs.ExitEnabled = true
	repo.settings = []SchedulingCenterGroupSettings{*gs}
	repo.groupAccounts[1] = []int64{7}

	// 手动移入，然后真实请求连续 3 次上游错误 → 第 1 次坏，冷却 5 分钟
	require.NoError(t, svc.MoveAccounts(ctx, 1, []int64{7}, SchedulingCenterPoolScheduling))
	clock.advance(time.Minute)
	repo.requests = map[int64][]SchedulingCenterRequestSample{7: {errReq(clock.t), errReq(clock.t), errReq(clock.t)}}
	require.NoError(t, svc.EvaluateRules(ctx))

	require.NotContains(t, repo.members, int64(7))
	st := repo.states[7]
	require.Equal(t, SchedulingCenterStateCooling, st.State)
	require.Equal(t, SchedulingCenterSourceManual, st.PrevSource)
	require.Equal(t, clock.t.Add(5*time.Minute), *st.CooldownUntil)
	ev := lastEvent(repo)
	require.Equal(t, SchedulingCenterActionExit, ev.Action)
	require.Equal(t, SchedulingCenterReasonExitError, ev.Reason)
	require.True(t, ev.IsBad)
	require.Equal(t, 1, ev.Detail["bad_count"])

	// 冷却未到：保持 cooling
	clock.advance(4 * time.Minute)
	require.NoError(t, svc.EvaluateRules(ctx))
	require.Equal(t, SchedulingCenterStateCooling, repo.states[7].State)

	// 冷却到期 → probing → 探测失败 → 第 2 次坏，冷却 15 分钟
	clock.advance(2 * time.Minute)
	probePass = false
	require.NoError(t, svc.EvaluateRules(ctx))
	st = repo.states[7]
	require.Equal(t, SchedulingCenterStateCooling, st.State)
	require.Equal(t, SchedulingCenterReasonProbeFailed, st.ReasonCode)
	require.Equal(t, clock.t.Add(15*time.Minute), *st.CooldownUntil)
	require.Equal(t, SchedulingCenterActionProbeFail, lastEvent(repo).Action)

	// 再次到期 → 探测通过 → 等待真实请求
	clock.advance(16 * time.Minute)
	probePass = true
	repo.requests = nil
	require.NoError(t, svc.EvaluateRules(ctx))
	st = repo.states[7]
	require.Equal(t, SchedulingCenterStateProbing, st.State)
	require.Equal(t, SchedulingCenterProbePassed, st.ProbeStatus)
	require.Equal(t, SchedulingCenterActionProbePass, lastEvent(repo).Action)

	// 探测通过之后只有 2 次正常请求：继续等待
	clock.advance(time.Minute)
	repo.requests = map[int64][]SchedulingCenterRequestSample{7: {okReq(clock.t, 800), okReq(clock.t, 900)}}
	require.NoError(t, svc.EvaluateRules(ctx))
	require.Equal(t, 2, repo.states[7].RecoverOKCount)

	// 第 3 次正常 → 恢复；原为手动成员，直接回到调度池
	repo.requests[7] = append([]SchedulingCenterRequestSample{okReq(clock.t, 700)}, repo.requests[7]...)
	clock.advance(time.Minute)
	require.NoError(t, svc.EvaluateRules(ctx))
	require.Contains(t, repo.members, int64(7))
	require.Equal(t, SchedulingCenterSourceManual, repo.members[7].Source)
	require.NotContains(t, repo.states, int64(7))
	require.Equal(t, SchedulingCenterActionEnter, lastEvent(repo).Action)
	require.Equal(t, SchedulingCenterReasonRecovered, lastEvent(repo).Reason)
}

func TestSchedulingCenterLifecycle_BackoffCapsAtOneHour(t *testing.T) {
	clock := newTestClock()
	repo := newFakeSchedulingCenterRepo()
	probePass := false
	svc := newLifecycleService(repo, clock, &probePass)
	ctx := context.Background()
	repo.groupAccounts[1] = []int64{7}
	for i := 0; i < 4; i++ {
		repo.events = append(repo.events, SchedulingCenterEvent{AccountID: 7, IsBad: true, CreatedAt: clock.t.Add(-time.Hour)})
	}
	st := &SchedulingCenterAccountState{GroupID: 1, AccountID: 7}
	gs := defaultSchedulingCenterGroupSettings(1)
	ev, err := svc.markBad(ctx, gs, st, SchedulingCenterActionExit, SchedulingCenterReasonExitIQ, nil, clock.t)
	require.NoError(t, err)
	require.Equal(t, 5, ev.Detail["bad_count"])
	require.Equal(t, 60, ev.Detail["cooldown_minutes"])

	// 3 小时之前的坏不计入
	repo.events = []SchedulingCenterEvent{{AccountID: 7, IsBad: true, CreatedAt: clock.t.Add(-4 * time.Hour)}}
	ev, err = svc.markBad(ctx, gs, st, SchedulingCenterActionExit, SchedulingCenterReasonExitIQ, nil, clock.t)
	require.NoError(t, err)
	require.Equal(t, 5, ev.Detail["cooldown_minutes"])

	// 自定义窗口与冷却步长：窗口 5 小时内已有 1 次坏 → 第 2 次用 7 分钟
	gs.BadWindowMinutes = 300
	gs.CooldownMinutes = []int{1, 7, 90}
	ev, err = svc.markBad(ctx, gs, st, SchedulingCenterActionExit, SchedulingCenterReasonExitIQ, nil, clock.t)
	require.NoError(t, err)
	require.Equal(t, 2, ev.Detail["bad_count"])
	require.Equal(t, 7, ev.Detail["cooldown_minutes"])
}

func TestSchedulingCenterLifecycle_RecoverWaitsForEntryRule(t *testing.T) {
	clock := newTestClock()
	repo := newFakeSchedulingCenterRepo()
	probePass := true
	svc := newLifecycleService(repo, clock, &probePass)
	ctx := context.Background()

	gs := defaultSchedulingCenterGroupSettings(1)
	gs.Enabled, gs.RuleEnabled, gs.RequireIQPass = true, true, true
	repo.settings = []SchedulingCenterGroupSettings{*gs}
	repo.groupAccounts[1] = []int64{7}
	passedAt := clock.t.Add(-time.Minute)
	started := passedAt
	repo.states[7] = &SchedulingCenterAccountState{GroupID: 1, AccountID: 7, State: SchedulingCenterStateProbing,
		PrevSource: SchedulingCenterSourceRule, ProbeStatus: SchedulingCenterProbePassed, ProbeStartedAt: &started, ProbePassedAt: &passedAt}
	repo.requests = map[int64][]SchedulingCenterRequestSample{7: {okReq(clock.t, 500), okReq(clock.t, 500), okReq(clock.t, 500)}}

	// 智商检测未通过 → 恢复后停在 ready
	repo.iqHistory = map[int64][]string{7: {IQVerdictIncorrect}}
	require.NoError(t, svc.EvaluateRules(ctx))
	require.Equal(t, SchedulingCenterStateReady, repo.states[7].State)
	require.NotContains(t, repo.members, int64(7))

	// 满足入池规则 → 规则入池
	repo.iqHistory[7] = []string{IQVerdictPassed}
	require.NoError(t, svc.EvaluateRules(ctx))
	require.Equal(t, SchedulingCenterSourceRule, repo.members[7].Source)
	require.NotContains(t, repo.states, int64(7))
	require.Equal(t, SchedulingCenterReasonRuleMatch, lastEvent(repo).Reason)
}

func TestSchedulingCenterManualOutBlocksEntryRule(t *testing.T) {
	clock := newTestClock()
	repo := newFakeSchedulingCenterRepo()
	probePass := true
	svc := newLifecycleService(repo, clock, &probePass)
	ctx := context.Background()

	gs := defaultSchedulingCenterGroupSettings(1)
	gs.Enabled, gs.RuleEnabled, gs.RequireIQPass = true, true, true
	repo.settings = []SchedulingCenterGroupSettings{*gs}
	repo.groupAccounts[1] = []int64{7, 8}
	repo.iqHistory = map[int64][]string{7: {IQVerdictPassed}, 8: {IQVerdictPassed}}

	require.NoError(t, svc.EvaluateRules(ctx))
	require.Contains(t, repo.members, int64(7))
	require.Contains(t, repo.members, int64(8))

	require.NoError(t, svc.MoveAccounts(ctx, 1, []int64{7}, SchedulingCenterPoolAccount))
	require.Equal(t, SchedulingCenterStateManualOut, repo.states[7].State)
	require.False(t, lastEvent(repo).IsBad, "manual move is not a bad event")

	require.NoError(t, svc.EvaluateRules(ctx))
	require.NotContains(t, repo.members, int64(7), "rule must not re-add manually removed account")

	require.NoError(t, svc.MoveAccounts(ctx, 1, []int64{7}, SchedulingCenterPoolScheduling))
	require.Equal(t, SchedulingCenterSourceManual, repo.members[7].Source)
	require.NotContains(t, repo.states, int64(7))
}

func TestSchedulingCenterGetAccountMetricsFirstTokenSource(t *testing.T) {
	checked := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	repo := newFakeSchedulingCenterRepo()
	repo.stats = map[int64]SchedulingCenterAccountStats{
		7: {Requests: 4, InputTokens: 20, CacheCreationTokens: 10, CacheReadTokens: 70, AvgFirstTokenMs: ptrF(900)},
		8: {Requests: 1, InputTokens: 10, AvgFirstTokenMs: ptrF(3000)},
	}
	repo.testTTFT = map[int64]float64{7: 650}
	repo.iq = map[int64]SchedulingCenterIQResult{8: {Status: IQVerdictIncorrect, CheckedAt: checked}}
	got, err := NewSchedulingCenterService(repo, nil, nil, nil).GetAccountMetrics(context.Background(), []int64{7, 8, 9}, 10)
	require.NoError(t, err)

	require.InDelta(t, 70.0, *got[7].CacheRate, 0.001)
	require.Equal(t, 650.0, *got[7].FirstTokenMs)
	require.Equal(t, SchedulingCenterFirstTokenSourceTest, got[7].FirstTokenSource)
	require.Equal(t, 3000.0, *got[8].FirstTokenMs)
	require.Equal(t, SchedulingCenterFirstTokenSourceTraffic, got[8].FirstTokenSource)
	require.Equal(t, checked, *got[8].IQCheckedAt)
	require.Nil(t, got[9].FirstTokenMs)
}

func TestSchedulingCenterRuleSatisfied(t *testing.T) {
	good := &SchedulingCenterAccountMetrics{CacheRate: ptrF(80), Requests: 100, FirstTokenMs: ptrF(1200), IQStatus: IQVerdictPassed}
	rule := &SchedulingCenterGroupSettings{MinCacheRate: ptrF(50), RequireIQPass: true, MaxFirstTokenMs: ptrI(2000)}
	passed := []string{IQVerdictPassed}
	require.True(t, SchedulingCenterRuleSatisfied(rule, good, passed))
	require.False(t, SchedulingCenterRuleSatisfied(&SchedulingCenterGroupSettings{}, good, passed))
	require.False(t, SchedulingCenterRuleSatisfied(rule, good, []string{IQVerdictIncorrect}))
	require.False(t, SchedulingCenterRuleSatisfied(rule, good, nil), "no IQ data")
	slow := *good
	slow.FirstTokenMs = ptrF(2500)
	require.False(t, SchedulingCenterRuleSatisfied(rule, &slow, passed))

	// 最近 3 次都通过才入池
	rule.EntryIQPassCount = 3
	require.False(t, SchedulingCenterRuleSatisfied(rule, good, []string{"passed", "passed"}))
	require.False(t, SchedulingCenterRuleSatisfied(rule, good, []string{"passed", "incorrect", "passed"}))
	require.True(t, SchedulingCenterRuleSatisfied(rule, good, []string{"passed", "passed", "passed", "incorrect"}))

	// 缓存率最少请求数
	rule.EntryCacheMinRequests = 50
	few := *good
	few.Requests = 10
	require.False(t, SchedulingCenterRuleSatisfied(rule, &few, []string{"passed", "passed", "passed"}))
}

func TestSchedulingCenterUpdateSettingsValidation(t *testing.T) {
	svc := NewSchedulingCenterService(newFakeSchedulingCenterRepo(), nil, nil, nil)
	ctx := context.Background()

	_, err := svc.UpdateGroupSettings(ctx, SchedulingCenterGroupSettings{GroupID: 1, CronExpression: "not a cron"})
	require.ErrorIs(t, err, ErrSchedulingCenterInvalidCron)
	_, err = svc.UpdateGroupSettings(ctx, SchedulingCenterGroupSettings{GroupID: 1, MinCacheRate: ptrF(120)})
	require.ErrorIs(t, err, ErrSchedulingCenterInvalidRule)
	_, err = svc.UpdateGroupSettings(ctx, SchedulingCenterGroupSettings{GroupID: 1, ExitIQWindow: 10, ExitIQMaxDegraded: ptrInt(10)})
	require.ErrorIs(t, err, ErrSchedulingCenterInvalidRule, "max degraded must be below window")
	_, err = svc.UpdateGroupSettings(ctx, SchedulingCenterGroupSettings{GroupID: 1, ExitErrorCount: ptrInt(0)})
	require.ErrorIs(t, err, ErrSchedulingCenterInvalidRule)
	_, err = svc.UpdateGroupSettings(ctx, SchedulingCenterGroupSettings{GroupID: 1, CooldownMinutes: []int{5, 0}})
	require.ErrorIs(t, err, ErrSchedulingCenterInvalidRule, "cooldown step must be positive")
	_, err = svc.UpdateGroupSettings(ctx, SchedulingCenterGroupSettings{GroupID: 1, ExitTrafficWindowHours: 500})
	require.ErrorIs(t, err, ErrSchedulingCenterInvalidRule)
	_, err = svc.UpdateGroupSettings(ctx, SchedulingCenterGroupSettings{GroupID: 1, RecoverLatencyMs: ptrI(-1)})
	require.ErrorIs(t, err, ErrSchedulingCenterInvalidRule)

	saved, err := svc.UpdateGroupSettings(ctx, SchedulingCenterGroupSettings{GroupID: 1, Enabled: true})
	require.NoError(t, err)
	require.Equal(t, SchedulingCenterDefaultCron, saved.CronExpression)
	require.Equal(t, 3, saved.RecoverSuccessCount, "zero counts fall back to defaults")
	require.Equal(t, 10, saved.ExitIQWindow)
	require.Equal(t, []int{5, 15, 30, 60}, saved.CooldownMinutes)
	require.Equal(t, 180, saved.BadWindowMinutes)
	require.Equal(t, 24, saved.ExitTrafficWindowHours)
}

func TestIQSSEDataStartsTTFT(t *testing.T) {
	// 与网关口径一致：首个有效事件即计时，不必等到正文
	require.True(t, iqSSEDataStartsTTFT(`{"type":"message_start","message":{}}`))
	require.True(t, iqSSEDataStartsTTFT(`{"choices":[{"delta":{"role":"assistant"}}]}`))
	require.True(t, iqSSEDataStartsTTFT(`{"type":"response.output_item.added","item":{"type":"reasoning"}}`))
	// 元数据、usage 分片、结束标记不计
	require.False(t, iqSSEDataStartsTTFT(`{"type":"response.created","response":{}}`))
	require.False(t, iqSSEDataStartsTTFT(`{"type":"response.in_progress","response":{}}`))
	require.False(t, iqSSEDataStartsTTFT(`{"choices":[],"usage":{"prompt_tokens":3}}`))
	require.False(t, iqSSEDataStartsTTFT("[DONE]"))
	require.False(t, iqSSEDataStartsTTFT(""))
}

func TestReadIQBodyStreamedVsJSON(t *testing.T) {
	sse := "event: response.created\n" +
		`data: {"type":"response.created","response":{}}` + "\n\n" +
		`data: {"type":"response.output_text.delta","delta":"21"}` + "\n\n"
	body, first, streamed := readIQBody(strings.NewReader(sse))
	require.True(t, streamed)
	require.False(t, first.IsZero())
	require.Equal(t, sse, string(body))

	// 上游忽略 stream 参数直接返回 JSON：不算流式，首 Token 不可用
	_, first, streamed = readIQBody(strings.NewReader(`{"choices":[{"message":{"content":"21"}}]}`))
	require.False(t, streamed)
	require.True(t, first.IsZero())
}

func TestExtractIQAnswerAnthropicStream(t *testing.T) {
	body := []byte("event: message_start\n" +
		`data: {"type":"message_start","message":{"usage":{"input_tokens":12}}}` + "\n\n" +
		`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"2"}}` + "\n\n" +
		`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"1"}}` + "\n\n" +
		`data: {"type":"message_delta","usage":{"output_tokens":3}}` + "\n\n")
	text, in, out, errMsg := extractIQAnswer(body)
	require.Equal(t, "21", text)
	require.Equal(t, int64(12), in)
	require.Equal(t, int64(3), out)
	require.Empty(t, errMsg)
}

// ---------------------------------------------------------------------------
// SyncPlans
// ---------------------------------------------------------------------------

type fakeScheduledPlanRepo struct {
	plans   map[int64]*ScheduledTestPlan
	nextID  int64
	deleted []int64
}

func (f *fakeScheduledPlanRepo) Create(_ context.Context, p *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	f.nextID++
	cp := *p
	cp.ID = f.nextID
	f.plans[cp.ID] = &cp
	return &cp, nil
}
func (f *fakeScheduledPlanRepo) GetByID(_ context.Context, id int64) (*ScheduledTestPlan, error) {
	cp := *f.plans[id]
	return &cp, nil
}
func (f *fakeScheduledPlanRepo) ListByAccountID(context.Context, int64) ([]*ScheduledTestPlan, error) {
	return nil, nil
}
func (f *fakeScheduledPlanRepo) ListDue(context.Context, time.Time) ([]*ScheduledTestPlan, error) {
	return nil, nil
}
func (f *fakeScheduledPlanRepo) Update(_ context.Context, p *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	cp := *p
	f.plans[p.ID] = &cp
	return &cp, nil
}
func (f *fakeScheduledPlanRepo) Delete(_ context.Context, id int64) error {
	delete(f.plans, id)
	f.deleted = append(f.deleted, id)
	return nil
}
func (f *fakeScheduledPlanRepo) UpdateAfterRun(context.Context, int64, time.Time, time.Time) error {
	return nil
}

func TestSchedulingCenterSyncPlansAppliesModelAndCron(t *testing.T) {
	planRepo := &fakeScheduledPlanRepo{plans: map[int64]*ScheduledTestPlan{
		1: {ID: 1, AccountID: 10, CronExpression: "*/30 * * * *", ModelID: "old-model", IQCheck: true},
		2: {ID: 2, AccountID: 99, CronExpression: "*/30 * * * *", IQCheck: true},
	}, nextID: 2}
	repo := newFakeSchedulingCenterRepo()
	repo.settings = []SchedulingCenterGroupSettings{
		{GroupID: 1, Enabled: true, CronExpression: "*/5 * * * *", ModelID: "claude-sonnet-4-5"},
	}
	repo.groupAccounts[1] = []int64{10, 11}
	repo.managed = []SchedulingCenterManagedPlan{
		{AccountID: 10, PlanID: 1, GroupID: 1, CronExpression: "*/30 * * * *", ModelID: "old-model"},
		{AccountID: 99, PlanID: 2, GroupID: 1, CronExpression: "*/30 * * * *"},
	}
	svc := NewSchedulingCenterService(repo, NewScheduledTestService(planRepo, nil), nil, nil)
	require.NoError(t, svc.SyncPlans(context.Background()))

	require.Equal(t, "*/5 * * * *", planRepo.plans[1].CronExpression)
	require.Equal(t, "claude-sonnet-4-5", planRepo.plans[1].ModelID)
	require.Equal(t, []int64{2}, planRepo.deleted)
	require.Len(t, repo.inserted, 1)
	created := planRepo.plans[repo.inserted[0].PlanID]
	require.Equal(t, int64(11), created.AccountID)
	require.Equal(t, "claude-sonnet-4-5", created.ModelID)
	require.True(t, created.IQCheck)
	require.False(t, created.IQStopOnFail)
	require.Equal(t, SchedulingCenterPlanMaxResults, created.MaxResults)
}

// ---------------------------------------------------------------------------
// 对比与导出
// ---------------------------------------------------------------------------

type fakeReportRepo struct {
	*fakeSchedulingCenterRepo
	prior   map[int64]string
	membEv  []SchedulingCenterMembershipEvent
	usage   []SchedulingCenterUsageBucket
	errs    []SchedulingCenterErrorBucket
	iqb     []SchedulingCenterIQBucket
	briefs  []SchedulingCenterAccountBrief
	evRange []SchedulingCenterEvent
	iqRecs  []SchedulingCenterExportIQRecord
}

func (f *fakeReportRepo) GetUsageBuckets(context.Context, []int64, time.Time, time.Time, time.Duration, int64) ([]SchedulingCenterUsageBucket, error) {
	return f.usage, nil
}
func (f *fakeReportRepo) GetErrorBuckets(context.Context, []int64, time.Time, time.Time, time.Duration) ([]SchedulingCenterErrorBucket, error) {
	return f.errs, nil
}
func (f *fakeReportRepo) GetIQBuckets(context.Context, []int64, time.Time, time.Time, time.Duration) ([]SchedulingCenterIQBucket, error) {
	return f.iqb, nil
}
func (f *fakeReportRepo) ListMembershipEvents(context.Context, int64, time.Time, time.Time) (map[int64]string, []SchedulingCenterMembershipEvent, error) {
	return f.prior, f.membEv, nil
}
func (f *fakeReportRepo) ListEventsInRange(context.Context, int64, time.Time, time.Time, int) ([]SchedulingCenterEvent, error) {
	return f.evRange, nil
}
func (f *fakeReportRepo) ListIQRecordsInRange(context.Context, []int64, time.Time, time.Time, int) ([]SchedulingCenterExportIQRecord, error) {
	return f.iqRecs, nil
}
func (f *fakeReportRepo) GetGroupBrief(_ context.Context, id int64) (*SchedulingCenterGroupBrief, error) {
	return &SchedulingCenterGroupBrief{ID: id, Name: "g"}, nil
}
func (f *fakeReportRepo) ListGroupAccountsBrief(context.Context, int64) ([]SchedulingCenterAccountBrief, error) {
	return f.briefs, nil
}

func TestBuildSchedulingCenterTimeline(t *testing.T) {
	from := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	to := from.Add(10 * time.Hour)
	at := func(h float64) time.Time { return from.Add(time.Duration(h * float64(time.Hour))) }

	tl := buildSchedulingCenterTimeline(from, to,
		map[int64]string{1: SchedulingCenterActionEnter, 2: SchedulingCenterActionExit},
		[]SchedulingCenterMembershipEvent{
			{AccountID: 1, Action: SchedulingCenterActionExit, At: at(2)},
			{AccountID: 1, Action: SchedulingCenterActionEnter, At: at(5)},
			{AccountID: 2, Action: SchedulingCenterActionEnter, At: at(3)},
			{AccountID: 2, Action: SchedulingCenterActionExit, At: at(4)},
			{AccountID: 3, Action: SchedulingCenterActionExit, At: at(6)}, // 无窗口前日志：第一条出池 → 此前在池
		},
		[]SchedulingCenterPoolMember{
			{AccountID: 1, EnteredAt: at(5)},
			{AccountID: 4, EnteredAt: from.Add(-time.Hour)}, // 日志上线前加入的成员
			{AccountID: 5, EnteredAt: at(8)},
		})

	require.True(t, tl.inPool(1, at(1)))
	require.False(t, tl.inPool(1, at(3)))
	require.True(t, tl.inPool(1, at(9)))
	require.False(t, tl.inPool(2, at(1)))
	require.True(t, tl.inPool(2, at(3.5)))
	require.False(t, tl.inPool(2, at(5)))
	require.True(t, tl.inPool(3, at(1)))
	require.False(t, tl.inPool(3, at(7)))
	require.True(t, tl.inPool(4, at(0.5)))
	require.False(t, tl.inPool(5, at(7)))
	require.True(t, tl.inPool(5, at(9)))
	require.False(t, tl.inPool(99, at(1)))
	require.Equal(t, 7*60.0, tl.minutesInPool(1)) // 0-2h + 5-10h
}

func TestSchedulingCenterComparisonAttributesByTime(t *testing.T) {
	clock := newTestClock()
	base := newFakeSchedulingCenterRepo()
	base.now = clock.now
	base.groupAccounts[1] = []int64{1, 2}
	from := clock.t.Truncate(time.Minute).Add(-24 * time.Hour)
	// 账号 1 在窗口开始 12 小时后入池，账号 2 一直在账号池
	base.members[1] = SchedulingCenterPoolMember{AccountID: 1, Source: SchedulingCenterSourceRule, EnteredAt: from.Add(12 * time.Hour)}
	repo := &fakeReportRepo{
		fakeSchedulingCenterRepo: base,
		membEv:                   []SchedulingCenterMembershipEvent{{AccountID: 1, Action: SchedulingCenterActionEnter, At: from.Add(12 * time.Hour)}},
		usage: []SchedulingCenterUsageBucket{
			{AccountID: 1, Bucket: from.Add(2 * time.Hour), Requests: 10, InputTokens: 100, CacheReadTokens: 0, FirstTokenSamples: 10, FirstTokenSumMs: 50000, SlowRequests: 2},
			{AccountID: 1, Bucket: from.Add(20 * time.Hour), Requests: 30, InputTokens: 20, CacheReadTokens: 80, FirstTokenSamples: 30, FirstTokenSumMs: 30000},
			{AccountID: 2, Bucket: from.Add(20 * time.Hour), Requests: 20, InputTokens: 100, FirstTokenSamples: 20, FirstTokenSumMs: 60000, SlowRequests: 5},
		},
		errs: []SchedulingCenterErrorBucket{
			{AccountID: 2, Bucket: from.Add(20 * time.Hour), Errors: 5},
			{AccountID: 1, Bucket: from.Add(3 * time.Hour), Errors: 10},
		},
		iqb: []SchedulingCenterIQBucket{
			{AccountID: 1, Bucket: from.Add(20 * time.Hour), Checks: 4, Passed: 4},
			{AccountID: 2, Bucket: from.Add(20 * time.Hour), Checks: 4, Passed: 1, Incorrect: 3},
		},
	}
	svc := NewSchedulingCenterService(repo, nil, nil, nil)
	svc.now = clock.now

	c, err := svc.GetComparison(context.Background(), 1, 24)
	require.NoError(t, err)

	// 调度池：只有账号 1 入池之后的数据
	require.Equal(t, int64(30), c.Pool.Requests)
	require.Equal(t, int64(0), c.Pool.Errors)
	require.Equal(t, 0.0, *c.Pool.ErrorRate)
	require.Equal(t, 80.0, *c.Pool.CacheRate)
	require.Equal(t, 1000.0, *c.Pool.AvgFirstTokenMs)
	require.Equal(t, 100.0, *c.Pool.IQPassRate)
	require.Equal(t, 1, c.Pool.Accounts)

	// 账号池：账号 1 入池前 + 账号 2
	require.Equal(t, int64(30), c.AccountPool.Requests)
	require.Equal(t, int64(15), c.AccountPool.Errors)
	require.Equal(t, 33.33, *c.AccountPool.ErrorRate)
	require.Equal(t, 2, c.AccountPool.Accounts)
	require.Equal(t, 25.0, *c.AccountPool.IQPassRate)
	require.InDelta(t, 23.33, *c.AccountPool.SlowRate, 0.01)

	require.Len(t, c.Series, 24, "hourly series covers the whole window")
	require.Equal(t, 1, c.Snapshot.PoolAccounts)
	require.Equal(t, 1, c.Snapshot.AccountAccounts)

	_, err = svc.GetComparison(context.Background(), 1, 500)
	require.ErrorIs(t, err, ErrSchedulingCenterInvalidRange)

	// 导出：账号级拆分
	repo.briefs = []SchedulingCenterAccountBrief{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}
	repo.evRange = []SchedulingCenterEvent{{AccountID: 2, Action: SchedulingCenterActionExit, Reason: SchedulingCenterReasonExitError, IsBad: true}}
	exp, err := svc.Export(context.Background(), 1, 24)
	require.NoError(t, err)
	require.Equal(t, SchedulingCenterExportSchema, exp.Schema)
	require.Len(t, exp.Accounts, 2)
	a1 := exp.Accounts[0]
	require.Equal(t, "in_pool", a1.CenterStatus)
	require.Equal(t, int64(30), a1.InPool.Requests)
	require.Equal(t, int64(10), a1.OutOfPool.Requests)
	require.Equal(t, int64(10), a1.OutOfPool.Errors)
	require.Equal(t, 12*60.0, a1.MinutesInPool)
	require.Equal(t, 1, exp.Accounts[1].BadEvents)
	require.Equal(t, 1, exp.Accounts[1].ExitReasons[SchedulingCenterReasonExitError])
}
