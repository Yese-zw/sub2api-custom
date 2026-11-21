package service

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// 调度池生效（灰度）：分组开启调度且打开 enforce_enabled 后，网关在该分组只从调度池（含探测通过、正在观察真实请求的
// 恢复中账号）选账号。两层回退保证不会比原逻辑更差：
//  1. 候选列表中没有任何池内账号（池为空 / 池内账号都不可调度）→ 直接使用原候选列表；
//  2. 只用池内账号选号失败（被排除、模型不支持、渠道限制等）→ 去掉过滤，按原逻辑重选一次。
// 网关只读内存快照，快照每 schedulingCenterGateRefreshInterval 从数据库刷新一次；刷新连续失败超过
// schedulingCenterGateStaleAfter 时快照作废，全部分组回到原逻辑。

const (
	schedulingCenterGateRefreshInterval = 10 * time.Second
	schedulingCenterGateStaleAfter      = 2 * time.Minute
	schedulingCenterGateQueryTimeout    = 5 * time.Second
)

// SchedulingCenterGateSource 提供开启调度池生效的分组及其可用账号。
type SchedulingCenterGateSource interface {
	ListEnforcedPools(ctx context.Context) (map[int64][]int64, error)
}

// SchedulingCenterEnforceStats 是本实例启动以来某分组调度池生效的选号统计（多实例部署时每个实例各自统计）。
type SchedulingCenterEnforceStats struct {
	Active         bool       `json:"active"`          // 本实例当前是否按调度池选号
	PoolSize       int        `json:"pool_size"`       // 网关可用的池内账号数（含恢复观察中的账号）
	PoolHits       int64      `json:"pool_hits"`       // 从调度池选中账号的次数
	Fallbacks      int64      `json:"fallbacks"`       // 池内选号失败、回退原逻辑重选的次数
	EmptyFallbacks int64      `json:"empty_fallbacks"` // 候选中没有池内账号、直接走原逻辑的次数
	LastFallbackAt *time.Time `json:"last_fallback_at,omitempty"`
	Since          time.Time  `json:"since"`
}

type schedulingCenterGateCounters struct {
	poolHits       atomic.Int64
	fallbacks      atomic.Int64
	emptyFallbacks atomic.Int64
	lastFallback   atomic.Int64 // unix nano
}

type schedulingCenterGateSnapshot struct {
	pools    map[int64]map[int64]struct{}
	loadedAt time.Time
}

// SchedulingCenterGate 维护调度池生效分组的内存快照，并为网关过滤候选账号。
type SchedulingCenterGate struct {
	src      SchedulingCenterGateSource
	snapshot atomic.Pointer[schedulingCenterGateSnapshot]
	counters sync.Map // groupID -> *schedulingCenterGateCounters
	since    time.Time
	now      func() time.Time

	refreshCh chan struct{}
	stopCh    chan struct{}
	doneCh    chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	started   atomic.Bool
}

func NewSchedulingCenterGate(src SchedulingCenterGateSource) *SchedulingCenterGate {
	return &SchedulingCenterGate{
		src:       src,
		since:     time.Now(),
		now:       time.Now,
		refreshCh: make(chan struct{}, 1),
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
}

// activeSchedulingCenterGate 是网关使用的全局实例（由 ProvideSchedulingCenterService 设置）。
var activeSchedulingCenterGate atomic.Pointer[SchedulingCenterGate]

func setActiveSchedulingCenterGate(g *SchedulingCenterGate) { activeSchedulingCenterGate.Store(g) }

func currentSchedulingCenterGate() *SchedulingCenterGate { return activeSchedulingCenterGate.Load() }

// Start 立即加载一次快照并启动后台刷新。
func (g *SchedulingCenterGate) Start() {
	if g == nil {
		return
	}
	g.startOnce.Do(func() {
		g.Refresh(context.Background())
		g.started.Store(true)
		go g.run()
	})
}

func (g *SchedulingCenterGate) Stop() {
	if g == nil {
		return
	}
	g.stopOnce.Do(func() {
		close(g.stopCh)
		if !g.started.Load() {
			return
		}
		select {
		case <-g.doneCh:
		case <-time.After(3 * time.Second):
		}
	})
}

// Notify 请求尽快刷新快照（调度池或设置变化后调用，不阻塞）。
func (g *SchedulingCenterGate) Notify() {
	if g == nil {
		return
	}
	select {
	case g.refreshCh <- struct{}{}:
	default:
	}
}

func (g *SchedulingCenterGate) run() {
	defer close(g.doneCh)
	ticker := time.NewTicker(schedulingCenterGateRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-g.stopCh:
			return
		case <-ticker.C:
		case <-g.refreshCh:
		}
		g.Refresh(context.Background())
	}
}

// Refresh 从数据库重新加载快照；失败时保留旧快照（超过 staleAfter 后自动作废）。
func (g *SchedulingCenterGate) Refresh(ctx context.Context) {
	if g == nil || g.src == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, schedulingCenterGateQueryTimeout)
	defer cancel()
	raw, err := g.src.ListEnforcedPools(ctx)
	if err != nil {
		slog.Warn("scheduling_center_gate_refresh_failed", "error", err)
		return
	}
	g.store(raw)
}

func (g *SchedulingCenterGate) store(raw map[int64][]int64) {
	pools := make(map[int64]map[int64]struct{}, len(raw))
	for groupID, ids := range raw {
		set := make(map[int64]struct{}, len(ids))
		for _, id := range ids {
			set[id] = struct{}{}
		}
		pools[groupID] = set
	}
	g.snapshot.Store(&schedulingCenterGateSnapshot{pools: pools, loadedAt: g.now()})
}

// current 返回未过期的快照；没有任何生效分组时返回 nil，网关走零开销路径。
func (g *SchedulingCenterGate) current() *schedulingCenterGateSnapshot {
	if g == nil {
		return nil
	}
	snap := g.snapshot.Load()
	if snap == nil || len(snap.pools) == 0 || g.now().Sub(snap.loadedAt) > schedulingCenterGateStaleAfter {
		return nil
	}
	return snap
}

func (g *SchedulingCenterGate) counter(groupID int64) *schedulingCenterGateCounters {
	if v, ok := g.counters.Load(groupID); ok {
		return v.(*schedulingCenterGateCounters)
	}
	v, _ := g.counters.LoadOrStore(groupID, &schedulingCenterGateCounters{})
	return v.(*schedulingCenterGateCounters)
}

// Stats 返回分组在本实例上的生效状态与选号统计。
func (g *SchedulingCenterGate) Stats(groupID int64) SchedulingCenterEnforceStats {
	out := SchedulingCenterEnforceStats{}
	if g == nil {
		return out
	}
	out.Since = g.since
	if snap := g.current(); snap != nil {
		if pool, ok := snap.pools[groupID]; ok {
			out.Active = true
			out.PoolSize = len(pool)
		}
	}
	if v, ok := g.counters.Load(groupID); ok {
		c := v.(*schedulingCenterGateCounters)
		out.PoolHits = c.poolHits.Load()
		out.Fallbacks = c.fallbacks.Load()
		out.EmptyFallbacks = c.emptyFallbacks.Load()
		if ns := c.lastFallback.Load(); ns > 0 {
			t := time.Unix(0, ns)
			out.LastFallbackAt = &t
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// 网关接入
// ---------------------------------------------------------------------------

type schedulingCenterAttemptKey struct{}

// schedulingCenterAttempt 记录一次选号中调度池过滤的执行情况。
type schedulingCenterAttempt struct {
	bypass       bool
	applied      atomic.Bool  // 至少一次把候选收窄到池内账号
	emptyGroup   atomic.Int64 // 候选中没有池内账号而直接回退的分组
	appliedGroup atomic.Int64
}

// filterSchedulingCenterPool 在分组开启调度池生效时把候选账号收窄为池内账号；候选中没有池内账号时原样返回。
// 返回新切片，不修改入参。
func filterSchedulingCenterPool(ctx context.Context, groupID *int64, accounts []Account) []Account {
	if groupID == nil || len(accounts) == 0 {
		return accounts
	}
	att, _ := ctx.Value(schedulingCenterAttemptKey{}).(*schedulingCenterAttempt)
	if att != nil && att.bypass {
		return accounts
	}
	g := currentSchedulingCenterGate()
	snap := g.current()
	if snap == nil {
		return accounts
	}
	pool, ok := snap.pools[*groupID]
	if !ok {
		return accounts
	}
	filtered := make([]Account, 0, min(len(pool), len(accounts)))
	for i := range accounts {
		if _, in := pool[accounts[i].ID]; in {
			filtered = append(filtered, accounts[i])
		}
	}
	if len(filtered) == 0 {
		if att != nil {
			att.emptyGroup.Store(*groupID)
		} else {
			g.counter(*groupID).emptyFallbacks.Add(1)
		}
		return accounts
	}
	if att != nil {
		att.applied.Store(true)
		att.appliedGroup.Store(*groupID)
	}
	return filtered
}

// runWithSchedulingCenterPool 执行一次选号：先按调度池过滤；过滤生效但选号失败时去掉过滤按原逻辑重选。
// 嵌套调用时只有最外层负责回退与统计。没有任何分组开启调度池生效时直接调用 fn。
func runWithSchedulingCenterPool[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	g := currentSchedulingCenterGate()
	if g.current() == nil || ctx.Value(schedulingCenterAttemptKey{}) != nil {
		return fn(ctx)
	}
	att := &schedulingCenterAttempt{}
	res, err := fn(context.WithValue(ctx, schedulingCenterAttemptKey{}, att))
	if !att.applied.Load() {
		if groupID := att.emptyGroup.Load(); groupID != 0 {
			g.counter(groupID).emptyFallbacks.Add(1)
		}
		return res, err
	}
	groupID := att.appliedGroup.Load()
	if err == nil {
		g.counter(groupID).poolHits.Add(1)
		return res, nil
	}
	if ctx.Err() != nil {
		return res, err
	}
	c := g.counter(groupID)
	c.fallbacks.Add(1)
	c.lastFallback.Store(g.now().UnixNano())
	slog.Info("scheduling_center_pool_fallback", "group_id", groupID, "error", err)
	return fn(context.WithValue(ctx, schedulingCenterAttemptKey{}, &schedulingCenterAttempt{bypass: true}))
}

// schedulingCenterBlocksSticky 报告粘性会话绑定的账号是否应被跳过：分组开启调度池生效、池非空且该账号不在池内。
// 跳过只是不复用绑定（继续走候选列表选号），不删除绑定；回退重选（bypass）时不跳过。
// previous_response_id 等必须落在原账号的续接请求不经过这里。
func schedulingCenterBlocksSticky(ctx context.Context, groupID *int64, accountID int64) bool {
	if groupID == nil || accountID <= 0 {
		return false
	}
	if att, _ := ctx.Value(schedulingCenterAttemptKey{}).(*schedulingCenterAttempt); att != nil && att.bypass {
		return false
	}
	snap := currentSchedulingCenterGate().current()
	if snap == nil {
		return false
	}
	pool, ok := snap.pools[*groupID]
	if !ok || len(pool) == 0 {
		return false
	}
	_, in := pool[accountID]
	return !in
}
