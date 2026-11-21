package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeGateSource struct {
	pools map[int64][]int64
	err   error
}

func (f *fakeGateSource) ListEnforcedPools(context.Context) (map[int64][]int64, error) {
	return f.pools, f.err
}

func installTestGate(t *testing.T, pools map[int64][]int64) *SchedulingCenterGate {
	t.Helper()
	g := NewSchedulingCenterGate(&fakeGateSource{pools: pools})
	g.Refresh(context.Background())
	prev := currentSchedulingCenterGate()
	setActiveSchedulingCenterGate(g)
	t.Cleanup(func() { setActiveSchedulingCenterGate(prev) })
	return g
}

func gateAccounts(ids ...int64) []Account {
	out := make([]Account, 0, len(ids))
	for _, id := range ids {
		out = append(out, Account{ID: id})
	}
	return out
}

func accountIDsOf(accounts []Account) []int64 {
	out := make([]int64, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, a.ID)
	}
	return out
}

func TestSchedulingCenterGateFilter(t *testing.T) {
	gid := int64(7)
	other := int64(8)
	installTestGate(t, map[int64][]int64{gid: {2, 3}, 9: {}})
	ctx := context.Background()

	if got := accountIDsOf(filterSchedulingCenterPool(ctx, &gid, gateAccounts(1, 2, 3, 4))); len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Fatalf("pool filter = %v, want [2 3]", got)
	}
	// 未开启生效的分组 / 无分组：原样返回
	if got := filterSchedulingCenterPool(ctx, &other, gateAccounts(1, 2)); len(got) != 2 {
		t.Fatalf("non-enforced group filtered: %v", accountIDsOf(got))
	}
	if got := filterSchedulingCenterPool(ctx, nil, gateAccounts(1, 2)); len(got) != 2 {
		t.Fatalf("nil group filtered")
	}
	// 候选中没有池内账号：回退原列表
	if got := filterSchedulingCenterPool(ctx, &gid, gateAccounts(1, 4)); len(got) != 2 {
		t.Fatalf("empty intersection should fall back, got %v", accountIDsOf(got))
	}
	// 池为空的生效分组：回退原列表
	empty := int64(9)
	if got := filterSchedulingCenterPool(ctx, &empty, gateAccounts(1, 4)); len(got) != 2 {
		t.Fatalf("empty pool should fall back")
	}
}

func TestSchedulingCenterGateRunFallback(t *testing.T) {
	gid := int64(7)
	g := installTestGate(t, map[int64][]int64{gid: {2}})

	// 池内选号成功
	calls := 0
	got, err := runWithSchedulingCenterPool(context.Background(), func(ctx context.Context) ([]int64, error) {
		calls++
		return accountIDsOf(filterSchedulingCenterPool(ctx, &gid, gateAccounts(1, 2, 3))), nil
	})
	if err != nil || calls != 1 || len(got) != 1 || got[0] != 2 {
		t.Fatalf("pool hit: got=%v err=%v calls=%d", got, err, calls)
	}

	// 池内选号失败 → 不过滤重选
	calls = 0
	got, err = runWithSchedulingCenterPool(context.Background(), func(ctx context.Context) ([]int64, error) {
		calls++
		list := filterSchedulingCenterPool(ctx, &gid, gateAccounts(1, 2, 3))
		if len(list) == 1 {
			return nil, ErrNoAvailableAccounts // 池内账号被排除 / 不支持模型
		}
		return accountIDsOf(list), nil
	})
	if err != nil || calls != 2 || len(got) != 3 {
		t.Fatalf("fallback: got=%v err=%v calls=%d", got, err, calls)
	}

	// 未收窄时的失败不重试
	calls = 0
	_, err = runWithSchedulingCenterPool(context.Background(), func(ctx context.Context) ([]int64, error) {
		calls++
		_ = filterSchedulingCenterPool(ctx, &gid, gateAccounts(1, 3))
		return nil, ErrNoAvailableAccounts
	})
	if !errors.Is(err, ErrNoAvailableAccounts) || calls != 1 {
		t.Fatalf("no-narrow failure: err=%v calls=%d", err, calls)
	}

	st := g.Stats(gid)
	if !st.Active || st.PoolSize != 1 || st.PoolHits != 1 || st.Fallbacks != 1 || st.EmptyFallbacks != 1 || st.LastFallbackAt == nil {
		t.Fatalf("stats = %+v", st)
	}
}

func TestSchedulingCenterGateNestedAndSticky(t *testing.T) {
	gid := int64(7)
	installTestGate(t, map[int64][]int64{gid: {2}})

	// 嵌套：只有最外层回退一次
	outer := 0
	_, _ = runWithSchedulingCenterPool(context.Background(), func(ctx context.Context) (int, error) {
		outer++
		return runWithSchedulingCenterPool(ctx, func(ctx context.Context) (int, error) {
			if len(filterSchedulingCenterPool(ctx, &gid, gateAccounts(1, 2))) == 1 {
				return 0, ErrNoAvailableAccounts
			}
			return 1, nil
		})
	})
	if outer != 2 {
		t.Fatalf("outer calls = %d, want 2", outer)
	}

	ctx := context.Background()
	if !schedulingCenterBlocksSticky(ctx, &gid, 1) {
		t.Fatal("non-pool sticky account should be blocked")
	}
	if schedulingCenterBlocksSticky(ctx, &gid, 2) {
		t.Fatal("pool sticky account should not be blocked")
	}
	bypass := context.WithValue(ctx, schedulingCenterAttemptKey{}, &schedulingCenterAttempt{bypass: true})
	if schedulingCenterBlocksSticky(bypass, &gid, 1) {
		t.Fatal("bypass should not block sticky")
	}
	other := int64(8)
	if schedulingCenterBlocksSticky(ctx, &other, 1) {
		t.Fatal("non-enforced group should not block sticky")
	}
}

func TestSchedulingCenterGateStaleAndRefreshError(t *testing.T) {
	gid := int64(7)
	src := &fakeGateSource{pools: map[int64][]int64{gid: {2}}}
	g := NewSchedulingCenterGate(src)
	now := time.Now()
	g.now = func() time.Time { return now }
	g.Refresh(context.Background())
	prev := currentSchedulingCenterGate()
	setActiveSchedulingCenterGate(g)
	defer setActiveSchedulingCenterGate(prev)

	// 刷新失败保留旧快照
	src.err = errors.New("db down")
	g.Refresh(context.Background())
	if got := filterSchedulingCenterPool(context.Background(), &gid, gateAccounts(1, 2)); len(got) != 1 {
		t.Fatalf("should keep last snapshot on error")
	}
	// 超过 staleAfter 作废，回到原逻辑
	now = now.Add(schedulingCenterGateStaleAfter + time.Second)
	if got := filterSchedulingCenterPool(context.Background(), &gid, gateAccounts(1, 2)); len(got) != 2 {
		t.Fatalf("stale snapshot should be ignored")
	}
	if g.Stats(gid).Active {
		t.Fatal("stale snapshot should report inactive")
	}
}

func TestSchedulingCenterRuleSatisfiedEnforceSkipsMissingCache(t *testing.T) {
	minCache := 50.0
	gs := &SchedulingCenterGroupSettings{Enabled: true, RuleEnabled: true, MinCacheRate: &minCache, EntryCacheMinRequests: 1}
	noTraffic := &SchedulingCenterAccountMetrics{Requests: 0}
	if SchedulingCenterRuleSatisfied(gs, noTraffic, nil) {
		t.Fatal("shadow mode: missing cache data should not satisfy")
	}
	gs.EnforceEnabled = true
	if !SchedulingCenterRuleSatisfied(gs, noTraffic, nil) {
		t.Fatal("enforce mode: missing cache data should not block entry")
	}
	low := 10.0
	if SchedulingCenterRuleSatisfied(gs, &SchedulingCenterAccountMetrics{Requests: 50, CacheRate: &low}, nil) {
		t.Fatal("enforce mode: known low cache must still block")
	}
}
