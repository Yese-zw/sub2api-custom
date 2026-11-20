package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/adminaccess"
)

const accountCacheRateWindow = 24 * time.Hour

// AccountCacheRateTotals 是账号在窗口内的请求数与各类 token 汇总。
type AccountCacheRateTotals struct {
	Requests            int64
	InputTokens         int64
	CacheCreationTokens int64
	CacheReadTokens     int64
}

// AccountCacheRate 是返回给前端的账号近 24 小时缓存率。
// Rate = 缓存读取 ÷ (输入 + 缓存写入 + 缓存读取) × 100，保留两位小数；没有请求或分母为 0 时为 nil。
type AccountCacheRate struct {
	Rate                *float64 `json:"rate"`
	Requests            int64    `json:"requests"`
	InputTokens         int64    `json:"input_tokens"`
	CacheCreationTokens int64    `json:"cache_creation_tokens"`
	CacheReadTokens     int64    `json:"cache_read_tokens"`
}

type accountCacheRateBatchReader interface {
	GetAccountCacheRateBatch(ctx context.Context, accountIDs []int64, startTime time.Time) (map[int64]AccountCacheRateTotals, error)
}

// cacheRatePercent 计算缓存率百分比（0-100），保留两位小数；分母不大于 0 时返回 nil。
func cacheRatePercent(totals AccountCacheRateTotals) *float64 {
	denom := totals.InputTokens + totals.CacheCreationTokens + totals.CacheReadTokens
	if totals.Requests <= 0 || denom <= 0 {
		return nil
	}
	rate := math.Round(float64(totals.CacheReadTokens)/float64(denom)*100*100) / 100
	return &rate
}

func newAccountCacheRate(totals AccountCacheRateTotals) *AccountCacheRate {
	return &AccountCacheRate{
		Rate:                cacheRatePercent(totals),
		Requests:            totals.Requests,
		InputTokens:         totals.InputTokens,
		CacheCreationTokens: totals.CacheCreationTokens,
		CacheReadTokens:     totals.CacheReadTokens,
	}
}

// GetCacheRate24hBatch 返回各账号近 24 小时的缓存率与 token 明细。
// 每个有效账号 ID 都会出现在结果里；没有请求的账号 Rate 为 nil、计数为 0。
func (s *AccountUsageService) GetCacheRate24hBatch(ctx context.Context, accountIDs []int64) (map[int64]*AccountCacheRate, error) {
	if adminaccess.RestrictedFromContext(ctx) != nil {
		if len(accountIDs) > 100 {
			return nil, fmt.Errorf("账户数量超过限制")
		}
		for _, id := range accountIDs {
			if err := scopedAccountIDAccess(ctx, s.accountRepo, id, "accounts.stats.read", false); err != nil {
				return nil, err
			}
		}
	}

	uniqueIDs := make([]int64, 0, len(accountIDs))
	seen := make(map[int64]struct{}, len(accountIDs))
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			continue
		}
		if _, exists := seen[accountID]; exists {
			continue
		}
		seen[accountID] = struct{}{}
		uniqueIDs = append(uniqueIDs, accountID)
	}

	result := make(map[int64]*AccountCacheRate, len(uniqueIDs))
	for _, accountID := range uniqueIDs {
		result[accountID] = newAccountCacheRate(AccountCacheRateTotals{})
	}
	if len(uniqueIDs) == 0 || s.usageLogRepo == nil {
		return result, nil
	}
	reader, ok := s.usageLogRepo.(accountCacheRateBatchReader)
	if !ok {
		return result, nil
	}

	totals, err := reader.GetAccountCacheRateBatch(ctx, uniqueIDs, time.Now().Add(-accountCacheRateWindow))
	if err != nil {
		return nil, err
	}
	for accountID, total := range totals {
		if _, wanted := result[accountID]; !wanted {
			continue
		}
		result[accountID] = newAccountCacheRate(total)
	}
	return result, nil
}
