package service

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// 调度中心兜底账号：
//
//   - 分组开启调度且开启兜底时，若调度池账号数低于 FallbackMinAccounts，从账号池挑选账号补入，来源记为 fallback；
//   - 候选：仍属于该分组、账号状态为启用且在现有系统中可调度、未被手动移出；开启「智商正常」时要求最近一次智商检测通过；
//   - 排序：先按调度中心状态分层（空闲 / 可入池 → 恢复检测中 → 冷却中），层内按 FallbackOrder 依次比较
//     （倍率低优先、首 Token 低优先、缓存率高优先，缺数据排后），最后按账号 ID；
//   - 非兜底成员数达到下限后，兜底账号自动退回账号池（不算一次「坏」）；关闭兜底时同样退回；
//   - 兜底账号与其他成员一样受出池规则约束。

const (
	SchedulingCenterSourceFallback = "fallback"

	SchedulingCenterReasonFallbackFill    = "fallback_fill"
	SchedulingCenterReasonFallbackRelease = "fallback_release"

	SchedulingCenterFallbackKeyRateMultiplier = "rate_multiplier"
	SchedulingCenterFallbackKeyFirstToken     = "first_token"
	SchedulingCenterFallbackKeyCacheRate      = "cache_rate"

	schedulingCenterMaxFallbackAccounts = 50
)

// DefaultSchedulingCenterFallbackOrder 返回默认的兜底排序：倍率低 → 首 Token 低 → 缓存率高。
func DefaultSchedulingCenterFallbackOrder() []string {
	return []string{
		SchedulingCenterFallbackKeyRateMultiplier,
		SchedulingCenterFallbackKeyFirstToken,
		SchedulingCenterFallbackKeyCacheRate,
	}
}

func normalizeSchedulingCenterFallbackOrder(order []string) ([]string, error) {
	if len(order) == 0 {
		return DefaultSchedulingCenterFallbackOrder(), nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(order))
	for _, key := range order {
		switch key {
		case SchedulingCenterFallbackKeyRateMultiplier, SchedulingCenterFallbackKeyFirstToken, SchedulingCenterFallbackKeyCacheRate:
		default:
			return nil, fmt.Errorf("%w: unknown fallback_order key %q", ErrSchedulingCenterInvalidRule, key)
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out, nil
}

// SchedulingCenterFallbackCandidate 是一个兜底候选账号。
type SchedulingCenterFallbackCandidate struct {
	AccountID      int64
	Tier           int // 0 空闲 / 可入池，1 恢复检测中，2 冷却中
	RateMultiplier float64
	FirstTokenMs   *float64
	CacheRate      *float64
	IQStatus       string
}

// compareOptional 比较两个可能缺失的值：缺失排后；lowerFirst 为 true 时小值优先。
func compareOptional(a, b *float64, lowerFirst bool) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	case *a == *b:
		return 0
	case (*a < *b) == lowerFirst:
		return -1
	default:
		return 1
	}
}

// RankSchedulingCenterFallback 按分层与 order 对候选排序（返回新切片）。
func RankSchedulingCenterFallback(candidates []SchedulingCenterFallbackCandidate, order []string) []SchedulingCenterFallbackCandidate {
	out := append([]SchedulingCenterFallbackCandidate(nil), candidates...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Tier != b.Tier {
			return a.Tier < b.Tier
		}
		for _, key := range order {
			var c int
			switch key {
			case SchedulingCenterFallbackKeyRateMultiplier:
				ra, rb := a.RateMultiplier, b.RateMultiplier
				c = compareOptional(&ra, &rb, true)
			case SchedulingCenterFallbackKeyFirstToken:
				c = compareOptional(a.FirstTokenMs, b.FirstTokenMs, true)
			case SchedulingCenterFallbackKeyCacheRate:
				c = compareOptional(a.CacheRate, b.CacheRate, false)
			}
			if c != 0 {
				return c < 0
			}
		}
		return a.AccountID < b.AccountID
	})
	return out
}

func validateSchedulingCenterFallbackSettings(in *SchedulingCenterGroupSettings) error {
	if in.FallbackMinAccounts <= 0 {
		in.FallbackMinAccounts = 1
	}
	if in.FallbackMinAccounts > schedulingCenterMaxFallbackAccounts {
		return fmt.Errorf("%w: fallback_min_accounts must be between 1 and %d", ErrSchedulingCenterInvalidRule, schedulingCenterMaxFallbackAccounts)
	}
	order, err := normalizeSchedulingCenterFallbackOrder(in.FallbackOrder)
	if err != nil {
		return err
	}
	in.FallbackOrder = order
	return nil
}

// ensureFallback 在出入池评估之后执行：补足或释放兜底账号，返回对应日志。
func (s *SchedulingCenterService) ensureFallback(
	ctx context.Context,
	gs *SchedulingCenterGroupSettings,
	accountIDs []int64,
	memberByID map[int64]SchedulingCenterPoolMember,
	states map[int64]*SchedulingCenterAccountState,
	now time.Time,
) ([]SchedulingCenterEvent, error) {
	minAccounts := max(gs.FallbackMinAccounts, 1)
	var fallbackIDs []int64
	nonFallback := 0
	for id, m := range memberByID {
		if m.Source == SchedulingCenterSourceFallback {
			fallbackIDs = append(fallbackIDs, id)
		} else {
			nonFallback++
		}
	}
	sort.Slice(fallbackIDs, func(i, j int) bool { return fallbackIDs[i] < fallbackIDs[j] })

	// 关闭兜底，或非兜底成员已够数：释放全部兜底账号
	if !gs.FallbackEnabled || nonFallback >= minAccounts {
		return s.releaseFallback(ctx, gs.GroupID, fallbackIDs, memberByID, now)
	}

	need := minAccounts - nonFallback - len(fallbackIDs)
	if need <= 0 {
		return nil, nil
	}
	candidates, err := s.fallbackCandidates(ctx, gs, accountIDs, memberByID, states)
	if err != nil || len(candidates) == 0 {
		return nil, err
	}
	ranked := RankSchedulingCenterFallback(candidates, gs.FallbackOrder)
	if len(ranked) > need {
		ranked = ranked[:need]
	}
	ids := make([]int64, 0, len(ranked))
	byID := make(map[int64]SchedulingCenterFallbackCandidate, len(ranked))
	for _, c := range ranked {
		ids = append(ids, c.AccountID)
		byID[c.AccountID] = c
	}
	added, err := s.repo.AddToSchedulingPool(ctx, gs.GroupID, ids, SchedulingCenterSourceFallback)
	if err != nil {
		return nil, err
	}
	// 兜底优先保证调度池不为空：被选中的冷却 / 恢复中账号清除其状态
	if err := s.repo.DeleteAccountStates(ctx, gs.GroupID, added); err != nil {
		return nil, err
	}
	events := make([]SchedulingCenterEvent, 0, len(added))
	for rank, id := range added {
		c := byID[id]
		detail := map[string]any{
			"rank":            rank + 1,
			"order":           gs.FallbackOrder,
			"rate_multiplier": c.RateMultiplier,
			"min_accounts":    minAccounts,
			"tier":            c.Tier,
		}
		if c.FirstTokenMs != nil {
			detail["first_token_ms"] = *c.FirstTokenMs
		}
		if c.CacheRate != nil {
			detail["cache_rate"] = *c.CacheRate
		}
		if c.IQStatus != "" {
			detail["iq_status"] = c.IQStatus
		}
		memberByID[id] = SchedulingCenterPoolMember{AccountID: id, Source: SchedulingCenterSourceFallback, EnteredAt: now}
		delete(states, id)
		events = append(events, newSchedulingCenterEvent(gs.GroupID, id, SchedulingCenterActionEnter,
			SchedulingCenterSourceFallback, SchedulingCenterReasonFallbackFill, detail, false, now))
	}
	return events, nil
}

func (s *SchedulingCenterService) releaseFallback(ctx context.Context, groupID int64, ids []int64, memberByID map[int64]SchedulingCenterPoolMember, now time.Time) ([]SchedulingCenterEvent, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	removed, err := s.repo.RemoveFromSchedulingPool(ctx, groupID, ids)
	if err != nil {
		return nil, err
	}
	events := make([]SchedulingCenterEvent, 0, len(removed))
	for _, id := range removed {
		delete(memberByID, id)
		events = append(events, newSchedulingCenterEvent(groupID, id, SchedulingCenterActionExit,
			SchedulingCenterSourceSystem, SchedulingCenterReasonFallbackRelease, nil, false, now))
	}
	return events, nil
}

func (s *SchedulingCenterService) fallbackCandidates(
	ctx context.Context,
	gs *SchedulingCenterGroupSettings,
	accountIDs []int64,
	memberByID map[int64]SchedulingCenterPoolMember,
	states map[int64]*SchedulingCenterAccountState,
) ([]SchedulingCenterFallbackCandidate, error) {
	// 账号启用状态、现有系统中的可调度开关与倍率；仓储不支持时不做过滤、倍率按 1
	briefs := map[int64]SchedulingCenterAccountBrief{}
	if rr, ok := s.repo.(SchedulingCenterReportRepository); ok {
		list, err := rr.ListGroupAccountsBrief(ctx, gs.GroupID)
		if err != nil {
			return nil, err
		}
		for _, b := range list {
			briefs[b.ID] = b
		}
	}

	pool := make([]int64, 0, len(accountIDs))
	tiers := map[int64]int{}
	for _, id := range accountIDs {
		if _, isMember := memberByID[id]; isMember {
			continue
		}
		tier := 0
		if st, ok := states[id]; ok {
			switch st.State {
			case SchedulingCenterStateManualOut:
				continue
			case SchedulingCenterStateProbing:
				tier = 1
			case SchedulingCenterStateCooling:
				tier = 2
			}
		}
		if b, ok := briefs[id]; ok && (b.Status != StatusActive || !b.Schedulable) {
			continue
		}
		pool = append(pool, id)
		tiers[id] = tier
	}
	if len(pool) == 0 {
		return nil, nil
	}

	out := make([]SchedulingCenterFallbackCandidate, 0, len(pool))
	for start := 0; start < len(pool); start += schedulingCenterMaxBatchAccounts {
		end := min(start+schedulingCenterMaxBatchAccounts, len(pool))
		metrics, err := s.GetAccountMetrics(ctx, pool[start:end], gs.FirstTokenTestSamples)
		if err != nil {
			return nil, err
		}
		for _, id := range pool[start:end] {
			m := metrics[id]
			c := SchedulingCenterFallbackCandidate{AccountID: id, Tier: tiers[id], RateMultiplier: 1}
			if b, ok := briefs[id]; ok && b.RateMultiplier != nil {
				c.RateMultiplier = *b.RateMultiplier
			}
			if m != nil {
				c.FirstTokenMs, c.CacheRate, c.IQStatus = m.FirstTokenMs, m.CacheRate, m.IQStatus
			}
			if gs.FallbackRequireIQPass && c.IQStatus != IQVerdictPassed {
				continue
			}
			out = append(out, c)
		}
	}
	return out, nil
}
