package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type cacheRateBatchRepoStub struct {
	usageBatchLogRepoStub
	totals map[int64]AccountCacheRateTotals
	err    error
	since  time.Time
	ids    []int64
}

func (r *cacheRateBatchRepoStub) GetAccountCacheRateBatch(_ context.Context, accountIDs []int64, startTime time.Time) (map[int64]AccountCacheRateTotals, error) {
	r.ids = append([]int64(nil), accountIDs...)
	r.since = startTime
	return r.totals, r.err
}

func TestAccountCacheRate24h(t *testing.T) {
	t.Run("公式只用缓存读取做分子", func(t *testing.T) {
		require.Nil(t, cacheRatePercent(AccountCacheRateTotals{}))
		require.Nil(t, cacheRatePercent(AccountCacheRateTotals{Requests: 2}))
		require.InDelta(t, 70, *cacheRatePercent(AccountCacheRateTotals{Requests: 1, InputTokens: 20, CacheCreationTokens: 10, CacheReadTokens: 70}), 0.0001)
		require.InDelta(t, 0, *cacheRatePercent(AccountCacheRateTotals{Requests: 1, InputTokens: 5, CacheCreationTokens: 5}), 0.0001)
		require.InDelta(t, 33.33, *cacheRatePercent(AccountCacheRateTotals{Requests: 1, InputTokens: 1, CacheCreationTokens: 1, CacheReadTokens: 1}), 0.0001)
	})

	t.Run("批量查询去重并补齐无请求账号", func(t *testing.T) {
		repo := &cacheRateBatchRepoStub{totals: map[int64]AccountCacheRateTotals{
			7:  {Requests: 4, InputTokens: 20, CacheCreationTokens: 10, CacheReadTokens: 70},
			99: {Requests: 1, InputTokens: 1},
		}}
		svc := &AccountUsageService{usageLogRepo: repo}

		rates, err := svc.GetCacheRate24hBatch(context.Background(), []int64{7, 7, 0, 9})
		require.NoError(t, err)
		require.Equal(t, []int64{7, 9}, repo.ids)
		require.WithinDuration(t, time.Now().Add(-accountCacheRateWindow), repo.since, 2*time.Second)

		require.Len(t, rates, 2)
		require.InDelta(t, 70, *rates[7].Rate, 0.0001)
		require.Equal(t, int64(4), rates[7].Requests)
		require.Equal(t, int64(20), rates[7].InputTokens)
		require.Equal(t, int64(10), rates[7].CacheCreationTokens)
		require.Equal(t, int64(70), rates[7].CacheReadTokens)
		require.Nil(t, rates[9].Rate)
		require.Equal(t, int64(0), rates[9].Requests)
	})

	t.Run("仓库不支持批量查询时返回空缓存率", func(t *testing.T) {
		svc := &AccountUsageService{usageLogRepo: &usageBatchLogRepoStub{}}
		rates, err := svc.GetCacheRate24hBatch(context.Background(), []int64{3})
		require.NoError(t, err)
		require.Nil(t, rates[3].Rate)
	})

	t.Run("查询失败时返回错误", func(t *testing.T) {
		svc := &AccountUsageService{usageLogRepo: &cacheRateBatchRepoStub{err: errors.New("boom")}}
		_, err := svc.GetCacheRate24hBatch(context.Background(), []int64{3})
		require.Error(t, err)
	})
}
