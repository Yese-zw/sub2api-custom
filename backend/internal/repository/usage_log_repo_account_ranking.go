package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// GetAccountRequestRanking 统计时间范围内每个账号的请求数据，按请求数从高到低排序。
func (r *usageLogRepository) GetAccountRequestRanking(ctx context.Context, startTime, endTime time.Time) (result *service.AccountRequestRankingResponse, err error) {
	query := `
		WITH account_usage AS (
			SELECT
				u.account_id,
				COUNT(*) AS requests,
				COALESCE(SUM(u.input_tokens), 0) AS input_tokens,
				COALESCE(SUM(u.output_tokens), 0) AS output_tokens,
				COALESCE(SUM(u.cache_creation_tokens + u.cache_read_tokens), 0) AS cache_tokens,
				COALESCE(SUM(u.input_tokens + u.output_tokens + u.cache_creation_tokens + u.cache_read_tokens), 0) AS tokens,
				COALESCE(SUM(COALESCE(u.account_stats_cost, u.total_cost) * COALESCE(u.account_rate_multiplier, 1)), 0) AS account_cost,
				COALESCE(SUM(u.actual_cost), 0) AS user_cost,
				COALESCE(AVG(u.duration_ms), 0) AS avg_duration_ms,
				COALESCE(AVG(u.first_token_ms) FILTER (WHERE u.first_token_ms > 0), 0) AS avg_first_token_ms,
				COUNT(DISTINCT u.user_id) AS users,
				COUNT(DISTINCT u.model) AS models,
				MAX(u.created_at) AS last_used_at
			FROM usage_logs u
			WHERE u.created_at >= $1 AND u.created_at < $2
			GROUP BY u.account_id
		),
		account_errors AS (
			-- 归属到账号的失败请求：排除用户侧业务限制（余额/配额不足等）与 count_tokens 请求
			SELECT
				e.account_id,
				COUNT(*) AS errors,
				COUNT(*) FILTER (WHERE COALESCE(e.upstream_status_code, e.status_code, 0) = 429) AS rate_limited
			FROM ops_error_logs e
			WHERE e.created_at >= $1 AND e.created_at < $2
				AND e.account_id IS NOT NULL
				AND COALESCE(e.status_code, 0) >= 400
				AND NOT e.is_business_limited
				AND NOT e.is_count_tokens
			GROUP BY e.account_id
		),
		merged AS (
			SELECT
				COALESCE(au.account_id, ae.account_id) AS account_id,
				COALESCE(au.requests, 0) AS requests,
				COALESCE(au.input_tokens, 0) AS input_tokens,
				COALESCE(au.output_tokens, 0) AS output_tokens,
				COALESCE(au.cache_tokens, 0) AS cache_tokens,
				COALESCE(au.tokens, 0) AS tokens,
				COALESCE(au.account_cost, 0) AS account_cost,
				COALESCE(au.user_cost, 0) AS user_cost,
				COALESCE(au.avg_duration_ms, 0) AS avg_duration_ms,
				COALESCE(au.avg_first_token_ms, 0) AS avg_first_token_ms,
				COALESCE(au.users, 0) AS users,
				COALESCE(au.models, 0) AS models,
				au.last_used_at,
				COALESCE(ae.errors, 0) AS errors,
				COALESCE(ae.rate_limited, 0) AS rate_limited
			FROM account_usage au
			FULL OUTER JOIN account_errors ae ON ae.account_id = au.account_id
		),
		ranked AS (
			SELECT
				m.*,
				SUM(m.requests) OVER () AS total_requests,
				SUM(m.tokens) OVER () AS total_tokens,
				SUM(m.account_cost) OVER () AS total_account_cost,
				SUM(m.user_cost) OVER () AS total_user_cost,
				COUNT(*) FILTER (WHERE m.requests > 0) OVER () AS total_accounts,
				SUM(m.errors) OVER () AS total_errors
			FROM merged m
		)
		SELECT
			r.account_id,
			COALESCE(a.name, ''),
			COALESCE(a.platform, ''),
			COALESCE(a.type, ''),
			COALESCE(a.status, ''),
			(a.id IS NULL OR a.deleted_at IS NOT NULL),
			COALESCE(BTRIM(a.credentials->>'base_url'), ''),
			r.requests, r.input_tokens, r.output_tokens, r.cache_tokens, r.tokens,
			r.account_cost, r.user_cost, r.avg_duration_ms, r.avg_first_token_ms,
			r.users, r.models, r.last_used_at, r.errors, r.rate_limited,
			r.total_requests, r.total_tokens, r.total_account_cost, r.total_user_cost, r.total_accounts, r.total_errors
		FROM ranked r
		LEFT JOIN accounts a ON a.id = r.account_id
		ORDER BY r.requests DESC, r.tokens DESC, r.account_id ASC
	`
	// 不在 SQL 中 LIMIT：按站点合并需要全部账号行，截断交给 service 层。
	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			result = nil
		}
	}()

	resp := &service.AccountRequestRankingResponse{Items: make([]service.AccountRequestRankingItem, 0)}
	for rows.Next() {
		var item service.AccountRequestRankingItem
		var deleted sql.NullBool
		var lastUsed sql.NullTime
		if err = rows.Scan(
			&item.AccountID, &item.Name, &item.Platform, &item.Type, &item.Status, &deleted, &item.BaseURL,
			&item.Requests, &item.InputTokens, &item.OutputTokens, &item.CacheTokens, &item.Tokens,
			&item.AccountCost, &item.UserCost, &item.AvgDurationMs, &item.AvgFirstTokenMs,
			&item.Users, &item.Models, &lastUsed, &item.Errors, &item.RateLimited,
			&resp.TotalRequests, &resp.TotalTokens, &resp.TotalAccountCost, &resp.TotalUserCost, &resp.TotalAccounts, &resp.TotalErrors,
		); err != nil {
			return nil, err
		}
		item.Deleted = deleted.Valid && deleted.Bool
		if lastUsed.Valid {
			t := lastUsed.Time
			item.LastUsedAt = &t
		}
		resp.Items = append(resp.Items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return resp, nil
}
