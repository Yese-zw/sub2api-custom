package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// 调度中心账号生命周期（每个分组独立，与账号管理中的调度状态无关）：
//
//	账号池(空闲) ──入池规则/手动──▶ 调度池
//	调度池 ──出池规则触发（一次「坏」）──▶ cooling（统计窗口内第 n 次坏按 cooldown_minutes[n-1] 冷却，默认 3 小时内 5/15/30/60 分钟封顶）
//	cooling ──到期──▶ probing：先主动探测（智商检测；账号不支持时跳过探测）
//	probing ──探测通过 + 真实请求连续正常 N 次──▶ 恢复：
//	         原为手动成员或入池规则未开启 → 直接回到调度池；否则 → ready，等待入池规则
//	probing ──探测失败 / 真实请求出现错误或过慢──▶ 再记一次「坏」，重新 cooling
//	调度池 ──手动移出──▶ manual_out（入池规则不会自动加回，手动移入可恢复）
//
// 出池条件（满足任意一条）：
//   - 24h 缓存率 < 入池缓存率门槛 − 下降点数（24h 请求数需达到最少样本）
//   - 入池后最近 W 次智商检测中降智次数 > M
//   - 入池后（且最近 exit_traffic_window_hours 内）最近连续 N 次流式真实请求首 Token 都 > 阈值
//   - 入池后（且最近 exit_traffic_window_hours 内）最近连续 N 次真实请求都是上游原因的错误（需开启运维监控）

const (
	SchedulingCenterStateCooling   = "cooling"
	SchedulingCenterStateProbing   = "probing"
	SchedulingCenterStateReady     = "ready"
	SchedulingCenterStateManualOut = "manual_out"

	SchedulingCenterProbeRunning = "running"
	SchedulingCenterProbePassed  = "passed"

	SchedulingCenterActionEnter       = "enter"
	SchedulingCenterActionExit        = "exit"
	SchedulingCenterActionCooldownEnd = "cooldown_end"
	SchedulingCenterActionProbePass   = "probe_pass"
	SchedulingCenterActionProbeFail   = "probe_fail"
	SchedulingCenterActionRecoverPass = "recover_pass"
	SchedulingCenterActionRecoverFail = "recover_fail"

	SchedulingCenterReasonManualIn       = "manual_in"
	SchedulingCenterReasonManualOut      = "manual_out"
	SchedulingCenterReasonRuleMatch      = "rule_match"
	SchedulingCenterReasonRecovered      = "recovered"
	SchedulingCenterReasonExitCache      = "exit_cache"
	SchedulingCenterReasonExitIQ         = "exit_iq"
	SchedulingCenterReasonExitLatency    = "exit_latency"
	SchedulingCenterReasonExitError      = "exit_error"
	SchedulingCenterReasonCooldownEnd    = "cooldown_end"
	SchedulingCenterReasonProbePassed    = "probe_passed"
	SchedulingCenterReasonProbeFailed    = "probe_failed"
	SchedulingCenterReasonRecoverOK      = "recover_ok"
	SchedulingCenterReasonRecoverLatency = "recover_latency"
	SchedulingCenterReasonRecoverError   = "recover_error"
	SchedulingCenterReasonWaitEntryRule  = "wait_entry_rule"

	SchedulingCenterSourceSystem = "system"

	schedulingCenterProbeConcurrency   = 5
	schedulingCenterProbeTimeout       = 6 * time.Minute
	schedulingCenterProbeStaleAfter    = 30 * time.Minute
	schedulingCenterRequestSampleLimit = 50
	schedulingCenterMaxRuleCount       = 50
)

// SchedulingCenterAccountState 是不在调度池中的账号在调度中心的状态。
type SchedulingCenterAccountState struct {
	GroupID        int64      `json:"group_id"`
	AccountID      int64      `json:"account_id"`
	State          string     `json:"state"`
	PrevSource     string     `json:"prev_source"`
	ReasonCode     string     `json:"reason_code"`
	CooldownUntil  *time.Time `json:"cooldown_until"`
	ProbeStatus    string     `json:"probe_status"`
	ProbeStartedAt *time.Time `json:"probe_started_at"`
	ProbePassedAt  *time.Time `json:"probe_passed_at"`
	RecoverOKCount int        `json:"recover_ok_count"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// SchedulingCenterEvent 是一条调度日志。
type SchedulingCenterEvent struct {
	ID          int64          `json:"id"`
	GroupID     int64          `json:"group_id"`
	AccountID   int64          `json:"account_id"`
	AccountName string         `json:"account_name"`
	Action      string         `json:"action"`
	Source      string         `json:"source"`
	Reason      string         `json:"reason"`
	Detail      map[string]any `json:"detail"`
	IsBad       bool           `json:"is_bad"`
	CreatedAt   time.Time      `json:"created_at"`
}

// SchedulingCenterRequestSample 是一次真实请求的结果。
type SchedulingCenterRequestSample struct {
	At           time.Time
	OK           bool
	FirstTokenMs *int64
}

// SchedulingCenterPoolView 是分组调度池成员与非成员账号状态。
type SchedulingCenterPoolView struct {
	Members []SchedulingCenterPoolMember   `json:"members"`
	States  []SchedulingCenterAccountState `json:"states"`
}

// SchedulingCenterExitDecision 描述一次出池判定。
type SchedulingCenterExitDecision struct {
	Reason string
	Detail map[string]any
}

type schedulingCenterProbeFunc func(ctx context.Context, accountID int64, gs SchedulingCenterGroupSettings) (passed bool, detail map[string]any)

// SchedulingCenterBackoff 返回窗口内第 badCount 次「坏」之后的冷却时长：
// steps[i] 对应第 i+1 次，超出后一直使用最后一个值（封顶）。steps 为空时用默认 5/15/30/60 分钟。
func SchedulingCenterBackoff(steps []int, badCount int) time.Duration {
	if len(steps) == 0 {
		steps = defaultSchedulingCenterGroupSettings(0).CooldownMinutes
	}
	idx := min(max(badCount, 1), len(steps)) - 1
	return time.Duration(steps[idx]) * time.Minute
}

func schedulingCenterBadWindow(gs *SchedulingCenterGroupSettings) time.Duration {
	if gs == nil || gs.BadWindowMinutes <= 0 {
		return 180 * time.Minute
	}
	return time.Duration(gs.BadWindowMinutes) * time.Minute
}

// SchedulingCenterCheckExit 判断调度池成员是否触发出池规则；未触发返回 nil。
// reqs / iq 均为新 → 旧，且已限定在入池之后。缺少数据（样本不足）一律视为未触发。
func SchedulingCenterCheckExit(gs *SchedulingCenterGroupSettings, m *SchedulingCenterAccountMetrics, reqs []SchedulingCenterRequestSample, iq []string) *SchedulingCenterExitDecision {
	if gs == nil {
		return nil
	}

	if gs.ExitCacheDropPoints != nil && gs.MinCacheRate != nil && m != nil && m.CacheRate != nil &&
		m.Requests >= int64(gs.ExitCacheMinRequests) {
		threshold := *gs.MinCacheRate - *gs.ExitCacheDropPoints
		if *m.CacheRate < threshold {
			return &SchedulingCenterExitDecision{Reason: SchedulingCenterReasonExitCache, Detail: map[string]any{
				"cache_rate": *m.CacheRate,
				"threshold":  threshold,
				"entry":      *gs.MinCacheRate,
				"drop":       *gs.ExitCacheDropPoints,
				"requests":   m.Requests,
			}}
		}
	}

	if gs.ExitIQMaxDegraded != nil {
		degraded := 0
		for _, status := range iq {
			if status == IQVerdictIncorrect {
				degraded++
			}
		}
		if degraded > *gs.ExitIQMaxDegraded {
			return &SchedulingCenterExitDecision{Reason: SchedulingCenterReasonExitIQ, Detail: map[string]any{
				"degraded": degraded,
				"checks":   len(iq),
				"max":      *gs.ExitIQMaxDegraded,
			}}
		}
	}

	if gs.ExitLatencyMs != nil && gs.ExitLatencyCount > 0 {
		values := make([]int64, 0, gs.ExitLatencyCount)
		for _, r := range reqs {
			if !r.OK || r.FirstTokenMs == nil {
				continue
			}
			values = append(values, *r.FirstTokenMs)
			if len(values) == gs.ExitLatencyCount {
				break
			}
		}
		if len(values) == gs.ExitLatencyCount {
			allSlow := true
			for _, v := range values {
				if v <= *gs.ExitLatencyMs {
					allSlow = false
					break
				}
			}
			if allSlow {
				return &SchedulingCenterExitDecision{Reason: SchedulingCenterReasonExitLatency, Detail: map[string]any{
					"values": values,
					"limit":  *gs.ExitLatencyMs,
					"count":  gs.ExitLatencyCount,
				}}
			}
		}
	}

	if gs.ExitErrorCount != nil && *gs.ExitErrorCount > 0 && len(reqs) >= *gs.ExitErrorCount {
		allErr := true
		for _, r := range reqs[:*gs.ExitErrorCount] {
			if r.OK {
				allErr = false
				break
			}
		}
		if allErr {
			return &SchedulingCenterExitDecision{Reason: SchedulingCenterReasonExitError, Detail: map[string]any{
				"count": *gs.ExitErrorCount,
			}}
		}
	}
	return nil
}

// schedulingCenterRecoverVerdict 按时间顺序（旧 → 新）检查探测通过后的真实请求：
// 出现上游错误或首 Token 超过阈值即失败；否则返回连续正常次数。
func schedulingCenterRecoverVerdict(gs *SchedulingCenterGroupSettings, newestFirst []SchedulingCenterRequestSample) (okCount int, failReason string, failDetail map[string]any) {
	for i := len(newestFirst) - 1; i >= 0; i-- {
		r := newestFirst[i]
		if !r.OK {
			return okCount, SchedulingCenterReasonRecoverError, map[string]any{"ok_count": okCount}
		}
		if r.FirstTokenMs != nil && gs.RecoverLatencyMs != nil && *r.FirstTokenMs > *gs.RecoverLatencyMs {
			return okCount, SchedulingCenterReasonRecoverLatency, map[string]any{
				"ok_count":       okCount,
				"first_token_ms": *r.FirstTokenMs,
				"limit":          *gs.RecoverLatencyMs,
			}
		}
		okCount++
	}
	return okCount, "", nil
}

func validateSchedulingCenterExitSettings(in *SchedulingCenterGroupSettings) error {
	def := defaultSchedulingCenterGroupSettings(0)
	if in.FirstTokenTestSamples <= 0 {
		in.FirstTokenTestSamples = def.FirstTokenTestSamples
	}
	if in.EntryCacheMinRequests <= 0 {
		in.EntryCacheMinRequests = def.EntryCacheMinRequests
	}
	if in.EntryIQPassCount <= 0 {
		in.EntryIQPassCount = def.EntryIQPassCount
	}
	if in.ExitTrafficWindowHours <= 0 {
		in.ExitTrafficWindowHours = def.ExitTrafficWindowHours
	}
	if in.BadWindowMinutes <= 0 {
		in.BadWindowMinutes = def.BadWindowMinutes
	}
	if len(in.CooldownMinutes) == 0 {
		in.CooldownMinutes = def.CooldownMinutes
	}
	if in.ExitCacheMinRequests <= 0 {
		in.ExitCacheMinRequests = 20
	}
	if in.ExitIQWindow <= 0 {
		in.ExitIQWindow = 10
	}
	if in.ExitLatencyCount <= 0 {
		in.ExitLatencyCount = 3
	}
	if in.RecoverSuccessCount <= 0 {
		in.RecoverSuccessCount = 3
	}
	invalid := func(msg string) error { return fmt.Errorf("%w: %s", ErrSchedulingCenterInvalidRule, msg) }
	if in.ExitCacheDropPoints != nil && (*in.ExitCacheDropPoints < 0 || *in.ExitCacheDropPoints > 100) {
		return invalid("exit_cache_drop_points must be between 0 and 100")
	}
	if in.ExitIQWindow > SchedulingCenterPlanMaxResults {
		return invalid("exit_iq_window must be at most 100")
	}
	if in.ExitIQMaxDegraded != nil && (*in.ExitIQMaxDegraded < 0 || *in.ExitIQMaxDegraded >= in.ExitIQWindow) {
		return invalid("exit_iq_max_degraded must be between 0 and exit_iq_window - 1")
	}
	if in.ExitLatencyMs != nil && *in.ExitLatencyMs <= 0 {
		return invalid("exit_latency_ms must be positive")
	}
	if in.ExitLatencyCount > schedulingCenterMaxRuleCount || in.RecoverSuccessCount > schedulingCenterMaxRuleCount {
		return invalid("counts must be at most 50")
	}
	if in.ExitErrorCount != nil && (*in.ExitErrorCount <= 0 || *in.ExitErrorCount > schedulingCenterMaxRuleCount) {
		return invalid("exit_error_count must be between 1 and 50")
	}
	if in.FirstTokenTestSamples > SchedulingCenterPlanMaxResults || in.EntryIQPassCount > SchedulingCenterPlanMaxResults {
		return invalid("first_token_test_samples / entry_iq_pass_count must be at most 100")
	}
	if in.EntryCacheMinRequests > 1_000_000 || in.ExitCacheMinRequests > 1_000_000 {
		return invalid("min requests too large")
	}
	if in.ExitTrafficWindowHours > 168 {
		return invalid("exit_traffic_window_hours must be between 1 and 168")
	}
	if in.BadWindowMinutes > 10080 {
		return invalid("bad_window_minutes must be between 1 and 10080")
	}
	if len(in.CooldownMinutes) > 10 {
		return invalid("cooldown_minutes supports at most 10 steps")
	}
	for _, m := range in.CooldownMinutes {
		if m <= 0 || m > 1440 {
			return invalid("each cooldown step must be between 1 and 1440 minutes")
		}
	}
	if in.RecoverLatencyMs != nil && *in.RecoverLatencyMs <= 0 {
		return invalid("recover_latency_ms must be positive")
	}
	return nil
}

// ---------------------------------------------------------------------------
// 查询
// ---------------------------------------------------------------------------

func (s *SchedulingCenterService) GetPoolView(ctx context.Context, groupID int64) (*SchedulingCenterPoolView, error) {
	members, err := s.repo.ListPoolMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}
	states, err := s.repo.ListAccountStates(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if members == nil {
		members = []SchedulingCenterPoolMember{}
	}
	if states == nil {
		states = []SchedulingCenterAccountState{}
	}
	return &SchedulingCenterPoolView{Members: members, States: states}, nil
}

func (s *SchedulingCenterService) ListEvents(ctx context.Context, groupID int64, keyword string, page, pageSize int) ([]SchedulingCenterEvent, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	events, total, err := s.repo.ListEvents(ctx, groupID, strings.TrimSpace(keyword), pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	if events == nil {
		events = []SchedulingCenterEvent{}
	}
	return events, total, nil
}

// ---------------------------------------------------------------------------
// 手动移动
// ---------------------------------------------------------------------------

// MoveAccounts 手动移动账号：移入调度池（来源 manual，清除冷却 / 恢复中状态）；
// 移回账号池（标记 manual_out，入池规则不会自动加回）。两者都写调度日志，不计入「坏」。
func (s *SchedulingCenterService) MoveAccounts(ctx context.Context, groupID int64, accountIDs []int64, pool string) error {
	ids := normalizeSchedulingCenterIDs(accountIDs)
	if len(ids) == 0 {
		return nil
	}
	now := s.now()
	switch pool {
	case SchedulingCenterPoolScheduling:
		added, err := s.repo.AddToSchedulingPool(ctx, groupID, ids, SchedulingCenterSourceManual)
		if err != nil {
			return err
		}
		if err := s.repo.DeleteAccountStates(ctx, groupID, added); err != nil {
			return err
		}
		events := make([]SchedulingCenterEvent, 0, len(added))
		for _, id := range added {
			events = append(events, newSchedulingCenterEvent(groupID, id, SchedulingCenterActionEnter,
				SchedulingCenterSourceManual, SchedulingCenterReasonManualIn, nil, false, now))
		}
		return s.repo.InsertEvents(ctx, events)
	case SchedulingCenterPoolAccount:
		removed, err := s.repo.RemoveFromSchedulingPool(ctx, groupID, ids)
		if err != nil {
			return err
		}
		events := make([]SchedulingCenterEvent, 0, len(removed))
		for _, id := range removed {
			if err := s.repo.UpsertAccountState(ctx, &SchedulingCenterAccountState{
				GroupID: groupID, AccountID: id, State: SchedulingCenterStateManualOut,
				ReasonCode: SchedulingCenterReasonManualOut,
			}); err != nil {
				return err
			}
			events = append(events, newSchedulingCenterEvent(groupID, id, SchedulingCenterActionExit,
				SchedulingCenterSourceManual, SchedulingCenterReasonManualOut, nil, false, now))
		}
		return s.repo.InsertEvents(ctx, events)
	default:
		return ErrSchedulingCenterInvalidPool
	}
}

// ---------------------------------------------------------------------------
// 周期评估
// ---------------------------------------------------------------------------

// EvaluateRules 对开启调度的分组依次执行：出池 → 冷却到期 → 恢复（探测 + 实测）→ 入池。
// 未开启调度的分组保持现状。
func (s *SchedulingCenterService) EvaluateRules(ctx context.Context) error {
	return s.withLock(ctx, "scheduling_center:evaluate_rules", func() error {
		settings, err := s.repo.ListGroupSettings(ctx)
		if err != nil {
			return err
		}
		var errs []error
		for i := range settings {
			gs := settings[i]
			if !gs.Enabled {
				continue
			}
			if err := s.evaluateGroup(ctx, &gs); err != nil {
				errs = append(errs, fmt.Errorf("group %d: %w", gs.GroupID, err))
			}
		}
		return errors.Join(errs...)
	})
}

func (s *SchedulingCenterService) evaluateGroup(ctx context.Context, gs *SchedulingCenterGroupSettings) error {
	now := s.now()
	groupID := gs.GroupID

	accountIDs, err := s.repo.ListGroupAccountIDs(ctx, groupID)
	if err != nil {
		return err
	}
	inGroup := make(map[int64]struct{}, len(accountIDs))
	for _, id := range accountIDs {
		inGroup[id] = struct{}{}
	}
	members, err := s.repo.ListPoolMembers(ctx, groupID)
	if err != nil {
		return err
	}
	memberByID := make(map[int64]SchedulingCenterPoolMember, len(members))
	for _, m := range members {
		memberByID[m.AccountID] = m
	}
	stateList, err := s.repo.ListAccountStates(ctx, groupID)
	if err != nil {
		return err
	}
	states := make(map[int64]*SchedulingCenterAccountState, len(stateList))
	for i := range stateList {
		states[stateList[i].AccountID] = &stateList[i]
	}

	type pendingProbe struct {
		accountID int64
		startedAt time.Time
	}
	var (
		events []SchedulingCenterEvent
		probes []pendingProbe
	)
	// 先写本轮日志，再启动探测，保证「冷却结束」排在探测结果之前。
	defer func() {
		if len(events) > 0 {
			if err := s.repo.InsertEvents(ctx, events); err != nil {
				logger.LegacyPrintf("service.scheduling_center", "[SchedulingCenter] group=%d insert events failed: %v", groupID, err)
			}
		}
		for _, p := range probes {
			s.launchProbe(*gs, p.accountID, p.startedAt)
		}
	}()

	// 1. 出池
	if gs.ExitEnabled && gs.HasExitCondition() && len(members) > 0 {
		decisions, err := s.evaluateExits(ctx, gs, members, now)
		if err != nil {
			return err
		}
		if len(decisions) > 0 {
			ids := make([]int64, 0, len(decisions))
			for id := range decisions {
				ids = append(ids, id)
			}
			removed, err := s.repo.RemoveFromSchedulingPool(ctx, groupID, ids)
			if err != nil {
				return err
			}
			for _, id := range removed {
				dec := decisions[id]
				st := &SchedulingCenterAccountState{
					GroupID: groupID, AccountID: id, PrevSource: memberByID[id].Source,
				}
				event, err := s.markBad(ctx, gs, st, SchedulingCenterActionExit, dec.Reason, dec.Detail, now)
				if err != nil {
					return err
				}
				events = append(events, event)
				states[id] = st
				delete(memberByID, id)
			}
		}
	}

	// 2. 冷却到期 → 恢复检测；3. 恢复检测（探测 + 实测）
	var recovering []*SchedulingCenterAccountState
	for id, st := range states {
		if _, ok := inGroup[id]; !ok {
			continue
		}
		if st.State == SchedulingCenterStateCooling && (st.CooldownUntil == nil || !now.Before(*st.CooldownUntil)) {
			st.State = SchedulingCenterStateProbing
			st.ReasonCode = SchedulingCenterReasonCooldownEnd
			st.CooldownUntil = nil
			st.ProbeStatus, st.ProbeStartedAt, st.ProbePassedAt, st.RecoverOKCount = "", nil, nil, 0
			if err := s.repo.UpsertAccountState(ctx, st); err != nil {
				return err
			}
			events = append(events, newSchedulingCenterEvent(groupID, id, SchedulingCenterActionCooldownEnd,
				SchedulingCenterSourceSystem, SchedulingCenterReasonCooldownEnd, nil, false, now))
		}
		if st.State != SchedulingCenterStateProbing {
			continue
		}
		switch {
		case st.ProbeStatus == "" ||
			(st.ProbeStatus == SchedulingCenterProbeRunning && st.ProbeStartedAt != nil && now.Sub(*st.ProbeStartedAt) > schedulingCenterProbeStaleAfter):
			startedAt := now.Truncate(time.Microsecond)
			st.ProbeStatus = SchedulingCenterProbeRunning
			st.ProbeStartedAt = &startedAt
			if err := s.repo.UpsertAccountState(ctx, st); err != nil {
				return err
			}
			probes = append(probes, pendingProbe{accountID: id, startedAt: startedAt})
		case st.ProbeStatus == SchedulingCenterProbePassed && st.ProbePassedAt != nil:
			recovering = append(recovering, st)
		}
	}
	if len(recovering) > 0 {
		recoverEvents, err := s.evaluateRecovering(ctx, gs, recovering, memberByID, now)
		events = append(events, recoverEvents...)
		if err != nil {
			return err
		}
	}

	// 4. 入池规则：只考虑空闲或 ready 的账号（冷却、恢复检测、手动移出的账号跳过）
	if gs.EntryRuleActive() {
		candidates := make([]int64, 0)
		for _, id := range accountIDs {
			if _, isMember := memberByID[id]; isMember {
				continue
			}
			if st, ok := states[id]; ok && st.State != SchedulingCenterStateReady {
				continue
			}
			candidates = append(candidates, id)
		}
		entryEvents, err := s.evaluateEntry(ctx, gs, candidates, now)
		events = append(events, entryEvents...)
		if err != nil {
			return err
		}
	}
	return nil
}

// markBad 记一次「坏」：按统计窗口内的坏次数计算冷却时长，把 st 改为 cooling 并写入，返回对应日志。
func (s *SchedulingCenterService) markBad(ctx context.Context, gs *SchedulingCenterGroupSettings, st *SchedulingCenterAccountState, action, reason string, detail map[string]any, now time.Time) (SchedulingCenterEvent, error) {
	counts, err := s.repo.CountBadEvents(ctx, st.GroupID, []int64{st.AccountID}, now.Add(-schedulingCenterBadWindow(gs)))
	if err != nil {
		return SchedulingCenterEvent{}, err
	}
	badCount := counts[st.AccountID] + 1
	cooldown := SchedulingCenterBackoff(gs.CooldownMinutes, badCount)
	until := now.Add(cooldown)

	st.State = SchedulingCenterStateCooling
	st.ReasonCode = reason
	st.CooldownUntil = &until
	st.ProbeStatus, st.ProbeStartedAt, st.ProbePassedAt, st.RecoverOKCount = "", nil, nil, 0
	if err := s.repo.UpsertAccountState(ctx, st); err != nil {
		return SchedulingCenterEvent{}, err
	}

	d := cloneSchedulingCenterDetail(detail)
	d["bad_count"] = badCount
	d["cooldown_minutes"] = int(cooldown / time.Minute)
	d["cooldown_until"] = until
	return newSchedulingCenterEvent(st.GroupID, st.AccountID, action, SchedulingCenterSourceSystem, reason, d, true, now), nil
}

func (s *SchedulingCenterService) evaluateExits(ctx context.Context, gs *SchedulingCenterGroupSettings, members []SchedulingCenterPoolMember, now time.Time) (map[int64]*SchedulingCenterExitDecision, error) {
	ids := make([]int64, 0, len(members))
	trafficSince := make(map[int64]time.Time, len(members))
	iqSince := make(map[int64]time.Time, len(members))
	windowHours := gs.ExitTrafficWindowHours
	if windowHours <= 0 {
		windowHours = 24
	}
	floor := now.Add(-time.Duration(windowHours) * time.Hour)
	for _, m := range members {
		ids = append(ids, m.AccountID)
		since := m.EnteredAt
		if since.Before(floor) {
			since = floor
		}
		trafficSince[m.AccountID] = since
		iqSince[m.AccountID] = m.EnteredAt
	}

	metrics := map[int64]*SchedulingCenterAccountMetrics{}
	if gs.ExitCacheDropPoints != nil && gs.MinCacheRate != nil {
		for start := 0; start < len(ids); start += schedulingCenterMaxBatchAccounts {
			end := min(start+schedulingCenterMaxBatchAccounts, len(ids))
			batch, err := s.GetAccountMetrics(ctx, ids[start:end], gs.FirstTokenTestSamples)
			if err != nil {
				return nil, err
			}
			for id, m := range batch {
				metrics[id] = m
			}
		}
	}
	var reqs map[int64][]SchedulingCenterRequestSample
	if gs.ExitLatencyMs != nil || gs.ExitErrorCount != nil {
		var err error
		if reqs, err = s.repo.GetRecentRequests(ctx, trafficSince, schedulingCenterRequestSampleLimit); err != nil {
			return nil, err
		}
	}
	var iq map[int64][]string
	if gs.ExitIQMaxDegraded != nil {
		var err error
		if iq, err = s.repo.GetRecentIQStatuses(ctx, iqSince, gs.ExitIQWindow); err != nil {
			return nil, err
		}
	}

	out := make(map[int64]*SchedulingCenterExitDecision)
	for _, id := range ids {
		if dec := SchedulingCenterCheckExit(gs, metrics[id], reqs[id], iq[id]); dec != nil {
			out[id] = dec
		}
	}
	return out, nil
}

func (s *SchedulingCenterService) evaluateRecovering(ctx context.Context, gs *SchedulingCenterGroupSettings, recovering []*SchedulingCenterAccountState, memberByID map[int64]SchedulingCenterPoolMember, now time.Time) ([]SchedulingCenterEvent, error) {
	since := make(map[int64]time.Time, len(recovering))
	for _, st := range recovering {
		since[st.AccountID] = *st.ProbePassedAt
	}
	reqs, err := s.repo.GetRecentRequests(ctx, since, schedulingCenterRequestSampleLimit)
	if err != nil {
		return nil, err
	}
	need := gs.RecoverSuccessCount
	if need <= 0 {
		need = 3
	}

	var events []SchedulingCenterEvent
	for _, st := range recovering {
		okCount, failReason, failDetail := schedulingCenterRecoverVerdict(gs, reqs[st.AccountID])
		switch {
		case failReason != "":
			event, err := s.markBad(ctx, gs, st, SchedulingCenterActionRecoverFail, failReason, failDetail, now)
			if err != nil {
				return events, err
			}
			events = append(events, event)
		case okCount >= need:
			events = append(events, newSchedulingCenterEvent(st.GroupID, st.AccountID, SchedulingCenterActionRecoverPass,
				SchedulingCenterSourceSystem, SchedulingCenterReasonRecoverOK, map[string]any{"ok_count": okCount}, false, now))
			if st.PrevSource == SchedulingCenterSourceManual || !gs.EntryRuleActive() {
				source := st.PrevSource
				if source == "" {
					source = SchedulingCenterSourceRule
				}
				added, err := s.repo.AddToSchedulingPool(ctx, st.GroupID, []int64{st.AccountID}, source)
				if err != nil {
					return events, err
				}
				if err := s.repo.DeleteAccountStates(ctx, st.GroupID, []int64{st.AccountID}); err != nil {
					return events, err
				}
				for _, id := range added {
					memberByID[id] = SchedulingCenterPoolMember{AccountID: id, Source: source, EnteredAt: now}
					events = append(events, newSchedulingCenterEvent(st.GroupID, id, SchedulingCenterActionEnter,
						SchedulingCenterSourceSystem, SchedulingCenterReasonRecovered, map[string]any{"source": source}, false, now))
				}
			} else {
				st.State = SchedulingCenterStateReady
				st.ReasonCode = SchedulingCenterReasonWaitEntryRule
				st.RecoverOKCount = okCount
				if err := s.repo.UpsertAccountState(ctx, st); err != nil {
					return events, err
				}
			}
		case okCount != st.RecoverOKCount:
			st.RecoverOKCount = okCount
			if err := s.repo.UpsertAccountState(ctx, st); err != nil {
				return events, err
			}
		}
	}
	return events, nil
}

func (s *SchedulingCenterService) evaluateEntry(ctx context.Context, gs *SchedulingCenterGroupSettings, candidates []int64, now time.Time) ([]SchedulingCenterEvent, error) {
	satisfied := make([]int64, 0)
	metricsByID := make(map[int64]*SchedulingCenterAccountMetrics)
	for start := 0; start < len(candidates); start += schedulingCenterMaxBatchAccounts {
		end := min(start+schedulingCenterMaxBatchAccounts, len(candidates))
		batch := candidates[start:end]
		metrics, err := s.GetAccountMetrics(ctx, batch, gs.FirstTokenTestSamples)
		if err != nil {
			return nil, err
		}
		var recentIQ map[int64][]string
		if gs.RequireIQPass {
			since := make(map[int64]time.Time, len(batch))
			for _, id := range batch {
				since[id] = time.Time{}
			}
			if recentIQ, err = s.repo.GetRecentIQStatuses(ctx, since, max(gs.EntryIQPassCount, 1)); err != nil {
				return nil, err
			}
		}
		for _, id := range batch {
			if SchedulingCenterRuleSatisfied(gs, metrics[id], recentIQ[id]) {
				satisfied = append(satisfied, id)
				metricsByID[id] = metrics[id]
			}
		}
	}
	if len(satisfied) == 0 {
		return nil, nil
	}
	added, err := s.repo.AddToSchedulingPool(ctx, gs.GroupID, satisfied, SchedulingCenterSourceRule)
	if err != nil {
		return nil, err
	}
	if err := s.repo.DeleteAccountStates(ctx, gs.GroupID, added); err != nil {
		return nil, err
	}
	events := make([]SchedulingCenterEvent, 0, len(added))
	for _, id := range added {
		detail := map[string]any{}
		if m := metricsByID[id]; m != nil {
			if m.CacheRate != nil {
				detail["cache_rate"] = *m.CacheRate
			}
			if m.FirstTokenMs != nil {
				detail["first_token_ms"] = *m.FirstTokenMs
			}
			if m.IQStatus != "" {
				detail["iq_status"] = m.IQStatus
			}
		}
		events = append(events, newSchedulingCenterEvent(gs.GroupID, id, SchedulingCenterActionEnter,
			SchedulingCenterSourceRule, SchedulingCenterReasonRuleMatch, detail, false, now))
	}
	return events, nil
}

// ---------------------------------------------------------------------------
// 主动探测
// ---------------------------------------------------------------------------

func (s *SchedulingCenterService) launchProbe(gs SchedulingCenterGroupSettings, accountID int64, startedAt time.Time) {
	run := func() {
		s.probeSem <- struct{}{}
		defer func() { <-s.probeSem }()
		ctx, cancel := context.WithTimeout(context.Background(), schedulingCenterProbeTimeout)
		defer cancel()
		passed, detail := s.probe(ctx, accountID, gs)
		if err := s.finishProbe(ctx, gs, accountID, startedAt, passed, detail); err != nil {
			logger.LegacyPrintf("service.scheduling_center", "[SchedulingCenter] group=%d account=%d finish probe failed: %v", gs.GroupID, accountID, err)
		}
	}
	if s.probeSync {
		run()
		return
	}
	go run()
}

// finishProbe 写回探测结果；若账号状态在探测期间已变化（手动移动、重新探测），丢弃本次结果。
func (s *SchedulingCenterService) finishProbe(ctx context.Context, gs SchedulingCenterGroupSettings, accountID int64, startedAt time.Time, passed bool, detail map[string]any) error {
	st, err := s.repo.GetAccountState(ctx, gs.GroupID, accountID)
	if err != nil || st == nil {
		return err
	}
	if st.State != SchedulingCenterStateProbing || st.ProbeStartedAt == nil || !st.ProbeStartedAt.Equal(startedAt) {
		return nil
	}
	now := s.now()
	var event SchedulingCenterEvent
	if passed {
		passedAt := now
		st.ProbeStatus = SchedulingCenterProbePassed
		st.ProbePassedAt = &passedAt
		st.ReasonCode = SchedulingCenterReasonProbePassed
		st.RecoverOKCount = 0
		event = newSchedulingCenterEvent(gs.GroupID, accountID, SchedulingCenterActionProbePass,
			SchedulingCenterSourceSystem, SchedulingCenterReasonProbePassed, detail, false, now)
	} else {
		counts, err := s.repo.CountBadEvents(ctx, gs.GroupID, []int64{accountID}, now.Add(-schedulingCenterBadWindow(&gs)))
		if err != nil {
			return err
		}
		badCount := counts[accountID] + 1
		cooldown := SchedulingCenterBackoff(gs.CooldownMinutes, badCount)
		until := now.Add(cooldown)
		st.State = SchedulingCenterStateCooling
		st.ReasonCode = SchedulingCenterReasonProbeFailed
		st.CooldownUntil = &until
		// CAS 的匹配条件使用参数 startedAt，这里可以清空。
		st.ProbeStatus, st.ProbeStartedAt, st.ProbePassedAt, st.RecoverOKCount = "", nil, nil, 0
		d := cloneSchedulingCenterDetail(detail)
		d["bad_count"] = badCount
		d["cooldown_minutes"] = int(cooldown / time.Minute)
		d["cooldown_until"] = until
		event = newSchedulingCenterEvent(gs.GroupID, accountID, SchedulingCenterActionProbeFail,
			SchedulingCenterSourceSystem, SchedulingCenterReasonProbeFailed, d, true, now)
	}
	ok, err := s.repo.CompareAndSetProbingState(ctx, st, startedAt)
	if err != nil || !ok {
		return err
	}
	return s.repo.InsertEvents(ctx, []SchedulingCenterEvent{event})
}

// defaultProbe 执行一次智商检测作为探测（必须判定通过，且首 Token 不超过恢复阈值）。
// 智商检测只读结果、不改账号状态。账号不支持智商检测（如 OAuth 账号）时跳过主动探测，
// 直接进入真实请求观察；不会退回连通性测试，因为连通性测试遇到 401/403/429 会改写账号状态，
// 从而影响真实调度。智商检测结果写入该账号的检测记录。
func (s *SchedulingCenterService) defaultProbe(ctx context.Context, accountID int64, gs SchedulingCenterGroupSettings) (bool, map[string]any) {
	if s.accountTestSvc == nil {
		return false, map[string]any{"error": "probe unavailable"}
	}
	startedAt := time.Now()
	task, err := s.accountTestSvc.RunIQCheck(ctx, accountID, gs.ModelID, true)
	if err != nil || task == nil {
		detail := map[string]any{"kind": "skipped"}
		if err != nil {
			detail["error"] = truncateForIQ(err.Error(), 200)
		}
		return true, detail
	}

	s.saveProbeResult(ctx, accountID, schedulingCenterIQTaskResult(task, startedAt))
	detail := map[string]any{"kind": "iq", "verdict": task.Verdict}
	if task.FirstTokenMs > 0 {
		detail["first_token_ms"] = task.FirstTokenMs
	}
	if task.Error != "" {
		detail["error"] = truncateForIQ(task.Error, 200)
	}
	passed := task.Verdict == IQVerdictPassed
	if passed && gs.RecoverLatencyMs != nil && task.FirstTokenMs > *gs.RecoverLatencyMs {
		passed = false
		detail["slow"] = true
		detail["limit"] = *gs.RecoverLatencyMs
	}
	return passed, detail
}

func (s *SchedulingCenterService) saveProbeResult(ctx context.Context, accountID int64, result *ScheduledTestResult) {
	if s.scheduledSvc == nil {
		return
	}
	planID, ok, err := s.repo.GetManagedPlanID(ctx, accountID)
	if err != nil || !ok {
		return
	}
	if err := s.scheduledSvc.SaveResult(ctx, planID, SchedulingCenterPlanMaxResults, result); err != nil {
		logger.LegacyPrintf("service.scheduling_center", "[SchedulingCenter] account=%d save probe result failed: %v", accountID, err)
	}
}

func schedulingCenterIQTaskResult(task *IQCheckTask, startedAt time.Time) *ScheduledTestResult {
	result := &ScheduledTestResult{
		Status:       "success",
		ResponseText: task.Answer,
		LatencyMs:    task.DurationMs,
		IQStatus:     task.Verdict,
		IQMessage:    "恢复探测 · " + IQCheckVerdictLabel(task),
		StartedAt:    startedAt,
		FinishedAt:   time.Now(),
	}
	if task.Status != "succeeded" {
		result.Status = "failed"
		result.ErrorMessage = task.Error
	}
	if task.FirstTokenMs > 0 {
		ft := task.FirstTokenMs
		result.FirstTokenMs = &ft
	}
	return result
}

func newSchedulingCenterEvent(groupID, accountID int64, action, source, reason string, detail map[string]any, isBad bool, at time.Time) SchedulingCenterEvent {
	if detail == nil {
		detail = map[string]any{}
	}
	return SchedulingCenterEvent{
		GroupID: groupID, AccountID: accountID, Action: action, Source: source,
		Reason: reason, Detail: detail, IsBad: isBad, CreatedAt: at,
	}
}

func cloneSchedulingCenterDetail(detail map[string]any) map[string]any {
	out := make(map[string]any, len(detail)+3)
	for k, v := range detail {
		out[k] = v
	}
	return out
}
