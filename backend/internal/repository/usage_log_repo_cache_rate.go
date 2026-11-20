package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// GetAccountCacheRateBatch 批量汇总账号在 startTime 之后的请求数与各类 token，用于计算缓存率。
// 窗口内没有日志的账号不会出现在结果里。
func (r *usageLogRepository) GetAccountCacheRateBatch(ctx context.Context, accountIDs []int64, startTime time.Time) (map[int64]service.AccountCacheRateTotals, error) {
	result := make(map[int64]service.AccountCacheRateTotals, len(accountIDs))
	if len(accountIDs) == 0 {
		return result, nil
	}

	query := `
		SELECT
			account_id,
			COUNT(*) AS requests,
			COALESCE(SUM(input_tokens), 0) AS input_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) AS cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) AS cache_read_tokens
		FROM usage_logs
		WHERE account_id = ANY($1) AND created_at >= $2
		GROUP BY account_id
	`
	rows, err := r.sql.QueryContext(ctx, query, pq.Array(accountIDs), startTime)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var accountID int64
		var totals service.AccountCacheRateTotals
		if err := rows.Scan(
			&accountID,
			&totals.Requests,
			&totals.InputTokens,
			&totals.CacheCreationTokens,
			&totals.CacheReadTokens,
		); err != nil {
			return nil, err
		}
		result[accountID] = totals
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
