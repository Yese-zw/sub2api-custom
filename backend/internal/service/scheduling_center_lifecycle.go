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
//	cooling ──到期──▶ probing：先主动探测（常规请求 hi 测速 + 智商检测，均只读；不支持智商检测时只测速）
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

	SchedulingCenterActionEnter        = "enter"
	SchedulingCenterActionExit         = "exit"
	SchedulingCenterActionCooldownEnd  = "cooldown_end"
	SchedulingCenterActionProbePass    = "probe_pass"
	SchedulingCenterActionProbeFail    = "probe_fail"
	SchedulingCenterActionRecoverPass  = "recover_pass"
	SchedulingCenterActionRecoverFail  = "recover_fail"
	SchedulingCenterActionManualOutEnd = "manual_out_end"

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
	SchedulingCenterReasonManualOutEnd   = "manual_out_expired"

	SchedulingCenterSourceSystem = "system"

	schedulingCenterProbeConcurrency = 5
	schedulingCenterProbeTimeout     = 6 * time.Minute
	schedulingCenterProbeStaleAfter  = 30 * time.Minute
	// schedulingCenterProbeShareTTL 内同一账号同一检测模型的测量结果在分组之间共用
	schedulingCenterProbeShareTTL      = 5 * time.Minute
	schedulingCenterRequestSampleLimit = 50
	schedulingCenterMaxExitSampleLimit = 300
	schedulingCenterMaxExitWindow      = 100
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
	Stream       bool
	FirstTokenMs *int64
}

// SchedulingCenterPoolView 是分组调度池成员与非成员账号状态。
type SchedulingCenterPoolView struct {
	Members []SchedulingCenterPoolMember   `json:"members"`
	States  []SchedulingCenterAccountState `json:"states"`
	// NextCheckAt 是分组下一次智商检测的时间（分组未开启调度时为空）。
	NextCheckAt *time.Time `json:"next_check_at"`
	// PlanOwners 列出本分组中检测计划归属于其他分组的账号（账号同属多个开启调度的分组时，
	// 检测频率 / 模型 / 保留条数按分组 ID 最小的分组执行）。
	PlanOwners []SchedulingCenterPlanOwner `json:"plan_owners"`
	// OpsMonitoringEnabled 为 false 时没有上游错误日志：「连续上游错误」出池与恢复观察中的错误判断都不会生效。
	OpsMonitoringEnabled bool `json:"ops_monitoring_enabled"`
	// Enforce 是调度池生效在本实例上的状态与选号统计（多实例部署时只反映处理本次请求的实例）。
	Enforce SchedulingCenterEnforceStats `json:"enforce"`
}

// SchedulingCenterPlanOwner 描述账号检测计划实际归属的分组及其生效设置。
type SchedulingCenterPlanOwner struct {
	AccountID      int64  `json:"account_id"`
	GroupID        int64  `json:"group_id"`
	GroupName      string `json:"group_name"`
	CronExpression string `json:"cron_expression"`
	ModelID        string `json:"model_id"`
	MaxResults     int    `json:"max_results"`
}

// SchedulingCenterExitDecision 描述一次出池判定。
type SchedulingCenterExitDecision struct {
	Reason string
	Detail map[string]any
}

// schedulingCenterProbeOutcome 是一次主动探测的原始测量结果（与分组无关）；
// 是否通过由各分组按自己的阈值判定（judgeSchedulingCenterProbe）。
type schedulingCenterProbeOutcome struct {
	IQSkipped      bool
	IQError        string
	IQVerdict      string
	IQTaskError    string
	LatencySuccess bool
	FirstTokenMs   *int64
	TotalMs        int64
	LatencyError   string
}

// schedulingCenterMeasureFunc 执行一次测量（智商检测 + 常规请求测速）并写入检测记录；单测可替换。
type schedulingCenterMeasureFunc func(ctx context.Context, accountID int64, modelID string, keep int) *schedulingCenterProbeOutcome

// schedulingCenterProbeShare 是同一账号同一检测模型的一次测量，供多个分组共用。
type schedulingCenterProbeShare struct {
	done    chan struct{}
	outcome *schedulingCenterProbeOutcome
	at      time.Time
}

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

	// 首 Token：最近 W 次成功的流式请求（有首 Token）中，超过阈值的达到 K 次；样本不足 W 次不判断。
	if latencyWindow, latencyCount := schedulingCenterExitLatencyWindow(gs), gs.ExitLatencyCount; gs.ExitLatencyMs != nil && latencyCount > 0 {
		values := make([]int64, 0, latencyWindow)
		for _, r := range reqs {
			if !r.OK || !r.Stream || r.FirstTokenMs == nil {
				continue
			}
			values = append(values, *r.FirstTokenMs)
			if len(values) == latencyWindow {
				break
			}
		}
		if len(values) == latencyWindow {
			slow := 0
			for _, v := range values {
				if v > *gs.ExitLatencyMs {
					slow++
				}
			}
			if slow >= latencyCount {
				return &SchedulingCenterExitDecision{Reason: SchedulingCenterReasonExitLatency, Detail: map[string]any{
					"values": values,
					"limit":  *gs.ExitLatencyMs,
					"window": latencyWindow,
					"slow":   slow,
					"count":  latencyCount,
				}}
			}
		}
	}

	// 上游错误：最近 W 次真实请求（流式与非流式）中上游错误达到 K 次；样本不足 W 次不判断。
	if errorWindow := schedulingCenterExitErrorWindow(gs); gs.ExitErrorCount != nil && *gs.ExitErrorCount > 0 && len(reqs) >= errorWindow {
		errs := 0
		for _, r := range reqs[:errorWindow] {
			if !r.OK {
				errs++
			}
		}
		if errs >= *gs.ExitErrorCount {
			return &SchedulingCenterExitDecision{Reason: SchedulingCenterReasonExitError, Detail: map[string]any{
				"window": errorWindow,
				"errors": errs,
				"count":  *gs.ExitErrorCount,
			}}
		}
	}
	return nil
}

// schedulingCenterExitLatencyWindow 返回首 Token 出池规则的观察次数（未设置时等于触发次数，即「连续 N 次」）。
func schedulingCenterExitLatencyWindow(gs *SchedulingCenterGroupSettings) int {
	if gs.ExitLatencyWindow > 0 {
		return gs.ExitLatencyWindow
	}
	return max(gs.ExitLatencyCount, 1)
}

// schedulingCenterExitErrorWindow 返回错误出池规则的观察次数（未设置时等于触发次数）。
func schedulingCenterExitErrorWindow(gs *SchedulingCenterGroupSettings) int {
	if gs.ExitErrorWindow > 0 {
		return gs.ExitErrorWindow
	}
	if gs.ExitErrorCount != nil && *gs.ExitErrorCount > 0 {
		return *gs.ExitErrorCount
	}
	return 3
}

// schedulingCenterExitSampleLimit 返回出池判断需要读取的真实请求条数：首 Token 规则只看成功的流式请求，
// 多读一些以免被非流式请求挤掉。
func schedulingCenterExitSampleLimit(gs *SchedulingCenterGroupSettings) int {
	need := max(schedulingCenterExitLatencyWindow(gs)*3, schedulingCenterExitErrorWindow(gs))
	return min(max(need, schedulingCenterRequestSampleLimit), schedulingCenterMaxExitSampleLimit)
}

// schedulingCenterRecoverVerdict 按时间顺序（旧 → 新）检查探测通过后的真实请求：
// 出现上游错误或首 Token 超过阈值即失败；否则返回连续正常次数。
func schedulingCenterRecoverVerdict(gs *SchedulingCenterGroupSettings, newestFirst []SchedulingCenterRequestSample) (okCount int, failReason string, failDetail map[string]any) {
	for i := len(newestFirst) - 1; i >= 0; i-- {
		r := newestFirst[i]
		if !r.OK {
			return okCount, SchedulingCenterReasonRecoverError, map[string]any{"ok_count": okCount}
		}
		if r.Stream && r.FirstTokenMs != nil && gs.RecoverLatencyMs != nil && *r.FirstTokenMs > *gs.RecoverLatencyMs {
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

// SchedulingCenterManualOutUntil 返回手动移出的到期时间：从移出时刻（状态写入时间）起算 ManualOutHours 小时；
// 0 表示永久，返回 nil。按当前设置计算，修改设置对已移出的账号立即生效。
func SchedulingCenterManualOutUntil(gs *SchedulingCenterGroupSettings, st *SchedulingCenterAccountState) *time.Time {
	if gs == nil || st == nil || st.State != SchedulingCenterStateManualOut || gs.ManualOutHours <= 0 || st.UpdatedAt.IsZero() {
		return nil
	}
	until := st.UpdatedAt.Add(time.Duration(gs.ManualOutHours) * time.Hour)
	return &until
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
	if in.ExitLatencyWindow <= 0 {
		in.ExitLatencyWindow = in.ExitLatencyCount
	}
	if in.ExitErrorWindow <= 0 {
		in.ExitErrorWindow = 3
		if in.ExitErrorCount != nil && *in.ExitErrorCount > 0 {
			in.ExitErrorWindow = *in.ExitErrorCount
		}
	}
	if in.RecoverSuccessCount <= 0 {
		in.RecoverSuccessCount = 3
	}
	invalid := func(msg string) error { return fmt.Errorf("%w: %s", ErrSchedulingCenterInvalidRule, msg) }
	if in.ExitCacheDropPoints != nil && (*in.ExitCacheDropPoints < 0 || *in.ExitCacheDropPoints > 100) {
		return invalid("exit_cache_drop_points must be between 0 and 100")
	}
	if in.RecordRetention <= 0 {
		in.RecordRetention = SchedulingCenterPlanMaxResults
	}
	if in.RecordRetention < SchedulingCenterMinRecordRetention || in.RecordRetention > SchedulingCenterMaxRecordRetention {
		return invalid(fmt.Sprintf("record_retention must be between %d and %d", SchedulingCenterMinRecordRetention, SchedulingCenterMaxRecordRetention))
	}
	// 依赖历史检测记录的条数不能超过保留条数
	if in.ExitIQWindow > in.RecordRetention {
		return invalid("exit_iq_window must not exceed record_retention")
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
	if in.ExitLatencyWindow > schedulingCenterMaxExitWindow || in.ExitErrorWindow > schedulingCenterMaxExitWindow {
		return invalid(fmt.Sprintf("exit_latency_window / exit_error_window must be between 1 and %d", schedulingCenterMaxExitWindow))
	}
	if in.ExitLatencyCount > in.ExitLatencyWindow {
		return invalid("exit_latency_count must not exceed exit_latency_window")
	}
	if in.ExitErrorCount != nil && (*in.ExitErrorCount <= 0 || *in.ExitErrorCount > in.ExitErrorWindow) {
		return invalid("exit_error_count must be between 1 and exit_error_window")
	}
	if in.FirstTokenTestSamples > in.RecordRetention || in.EntryIQPassCount > in.RecordRetention {
		return invalid("first_token_test_samples / entry_iq_pass_count must not exceed record_retention")
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
	if in.ManualOutHours < 0 || in.ManualOutHours > 720 {
		return invalid("manual_out_hours must be between 0 and 720")
	}
	if in.EventRetentionDays <= 0 {
		in.EventRetentionDays = SchedulingCenterDefaultEventRetentionDays
	}
	if in.EventRetentionDays < SchedulingCenterMinEventRetentionDays || in.EventRetentionDays > SchedulingCenterMaxEventRetentionDays {
		return invalid(fmt.Sprintf("event_retention_days must be between %d and %d", SchedulingCenterMinEventRetentionDays, SchedulingCenterMaxEventRetentionDays))
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
	nextCheckAt, err := s.nextCheckAt(ctx, groupID)
	if err != nil {
		return nil, err
	}
	owners, err := s.repo.ListForeignPlanOwners(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if owners == nil {
		owners = []SchedulingCenterPlanOwner{}
	}
	return &SchedulingCenterPoolView{Members: members, States: states, NextCheckAt: nextCheckAt, PlanOwners: owners,
		OpsMonitoringEnabled: s.OpsMonitoringEnabled(ctx), Enforce: s.gate.Stats(groupID)}, nil
}

// nextCheckAt 取该分组托管检测计划中最早的下次执行时间；计划尚未同步出来时按分组的 cron 推算。
func (s *SchedulingCenterService) nextCheckAt(ctx context.Context, groupID int64) (*time.Time, error) {
	settings, err := s.GetGroupSettings(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled {
		return nil, nil
	}
	next, err := s.repo.GetNextPlanRunAt(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if next != nil {
		return next, nil
	}
	computed, err := computeNextRun(settings.CronExpression, s.now())
	if err != nil {
		return nil, nil
	}
	return &computed, nil
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
	defer s.gate.Notify()
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
	defer s.gate.Notify()
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

	// 2. 冷却到期 → 恢复检测；3. 恢复检测（探测 + 实测）；手动移出到期 → 回到普通账号池
	var recovering []*SchedulingCenterAccountState
	var manualExpired []int64
	for id, st := range states {
		if _, ok := inGroup[id]; !ok {
			continue
		}
		if st.State == SchedulingCenterStateManualOut {
			if until := SchedulingCenterManualOutUntil(gs, st); until != nil && !now.Before(*until) {
				manualExpired = append(manualExpired, id)
			}
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
	if len(manualExpired) > 0 {
		if err := s.repo.DeleteAccountStates(ctx, groupID, manualExpired); err != nil {
			return err
		}
		for _, id := range manualExpired {
			delete(states, id)
			events = append(events, newSchedulingCenterEvent(groupID, id, SchedulingCenterActionManualOutEnd,
				SchedulingCenterSourceSystem, SchedulingCenterReasonManualOutEnd,
				map[string]any{"hours": gs.ManualOutHours}, false, now))
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
		for _, e := range entryEvents {
			memberByID[e.AccountID] = SchedulingCenterPoolMember{AccountID: e.AccountID, Source: SchedulingCenterSourceRule, EnteredAt: now}
		}
	}

	// 5. 兜底：保证调度池不少于下限；非兜底成员够数后释放兜底账号
	fallbackEvents, err := s.ensureFallback(ctx, gs, accountIDs, memberByID, states, now)
	events = append(events, fallbackEvents...)
	return err
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
		if m.Source == SchedulingCenterSourceFixed {
			continue // OAuth / Setup Token 账号固定在调度池，不参与出池
		}
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
		if reqs, err = s.repo.GetRecentRequests(ctx, trafficSince, schedulingCenterExitSampleLimit(gs)); err != nil {
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
		outcome, shared := s.sharedMeasure(ctx, accountID, gs)
		passed, detail := judgeSchedulingCenterProbe(&gs, outcome)
		if shared {
			detail["shared"] = true
		}
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
	defer s.gate.Notify()
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

// sharedMeasure 返回账号在该检测模型下的测量结果：schedulingCenterProbeShareTTL 内已有结果（或正在测量）时
// 直接复用，避免账号同属多个分组、同时冷却结束时被重复探测。返回值 shared 表示复用了已有结果。
func (s *SchedulingCenterService) sharedMeasure(ctx context.Context, accountID int64, gs SchedulingCenterGroupSettings) (*schedulingCenterProbeOutcome, bool) {
	key := fmt.Sprintf("%d|%s", accountID, gs.ModelID)
	now := s.now()

	s.probeShareMu.Lock()
	for k, e := range s.probeShares {
		select {
		case <-e.done:
			if now.Sub(e.at) >= schedulingCenterProbeShareTTL {
				delete(s.probeShares, k)
			}
		default:
		}
	}
	if e, ok := s.probeShares[key]; ok {
		s.probeShareMu.Unlock()
		select {
		case <-e.done:
			return e.outcome, true
		case <-ctx.Done():
			return &schedulingCenterProbeOutcome{LatencyError: ctx.Err().Error()}, false
		}
	}
	e := &schedulingCenterProbeShare{done: make(chan struct{})}
	s.probeShares[key] = e
	s.probeShareMu.Unlock()

	outcome := s.measure(ctx, accountID, gs.ModelID, schedulingCenterRecordRetention(&gs))
	if outcome == nil {
		outcome = &schedulingCenterProbeOutcome{LatencyError: "probe unavailable"}
	}
	e.outcome, e.at = outcome, s.now()
	close(e.done)
	return outcome, false
}

// judgeSchedulingCenterProbe 按分组设置判定测量结果：
//  1. 测速必须成功，且首 Token 不超过恢复阈值（recover_latency_ms）；
//  2. 账号支持智商检测时必须判定通过；不支持（如 OAuth 账号）时跳过这一项。
func judgeSchedulingCenterProbe(gs *SchedulingCenterGroupSettings, o *schedulingCenterProbeOutcome) (bool, map[string]any) {
	detail := map[string]any{"kind": "probe", "total_ms": o.TotalMs}
	passed := true
	if o.IQSkipped {
		detail["iq_skipped"] = true
		if o.IQError != "" {
			detail["iq_error"] = o.IQError
		}
	} else {
		detail["verdict"] = o.IQVerdict
		if o.IQTaskError != "" {
			detail["error"] = o.IQTaskError
		}
		if o.IQVerdict != IQVerdictPassed {
			passed = false
		}
	}
	if o.FirstTokenMs != nil {
		detail["first_token_ms"] = *o.FirstTokenMs
	}
	switch {
	case !o.LatencySuccess:
		passed = false
		detail["latency_failed"] = true
		if o.LatencyError != "" {
			detail["latency_error"] = o.LatencyError
		}
	case gs.RecoverLatencyMs != nil && (o.FirstTokenMs == nil || *o.FirstTokenMs > *gs.RecoverLatencyMs):
		passed = false
		detail["slow"] = true
		detail["limit"] = *gs.RecoverLatencyMs
	}
	return passed, detail
}

// defaultMeasure 执行一次主动探测测量，均不会改写账号状态：
//   - 智商检测：账号不支持时标记 IQSkipped；
//   - 测速：按「测试账号连接 → 常规请求（hi）」在只读账号仓储上请求一次。
//
// 结果写入该账号的检测记录（按 keep 条保留）。
func (s *SchedulingCenterService) defaultMeasure(ctx context.Context, accountID int64, modelID string, keep int) *schedulingCenterProbeOutcome {
	if s.accountTestSvc == nil || s.latencyProbe == nil {
		return &schedulingCenterProbeOutcome{LatencyError: "probe unavailable"}
	}
	o := &schedulingCenterProbeOutcome{}
	startedAt := time.Now()
	var record *ScheduledTestResult
	task, iqErr := s.accountTestSvc.RunIQCheck(ctx, accountID, modelID, true)
	if iqErr != nil || task == nil {
		o.IQSkipped = true
		if iqErr != nil {
			o.IQError = truncateForIQ(iqErr.Error(), 200)
		}
		record = &ScheduledTestResult{
			Status: "success", IQMessage: "恢复探测 · 不支持智商检测，仅测速",
			StartedAt: startedAt, FinishedAt: time.Now(),
		}
	} else {
		record = schedulingCenterIQTaskResult(task, startedAt)
		o.IQVerdict = task.Verdict
		if task.Error != "" {
			o.IQTaskError = truncateForIQ(task.Error, 200)
		}
	}

	probe := s.latencyProbe.RunLatencyProbe(ctx, accountID, modelID)
	applyLatencyProbe(record, probe)
	o.LatencySuccess = probe.Success
	o.FirstTokenMs = probe.FirstTokenMs
	o.TotalMs = probe.TotalMs
	if probe.Error != "" {
		o.LatencyError = truncateForIQ(probe.Error, 200)
	}

	s.saveProbeResult(ctx, accountID, record, keep)
	return o
}

func (s *SchedulingCenterService) saveProbeResult(ctx context.Context, accountID int64, result *ScheduledTestResult, keep int) {
	if s.scheduledSvc == nil {
		return
	}
	planID, ok, err := s.repo.GetManagedPlanID(ctx, accountID)
	if err != nil || !ok {
		return
	}
	if err := s.scheduledSvc.SaveResult(ctx, planID, keep, result); err != nil {
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
