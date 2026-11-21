package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// GetAccountConsumption 统计每个账号近 24 小时与近 7 天的账号口径消耗，用于估算上游余额可用天数。
func (r *usageLogRepository) GetAccountConsumption(ctx context.Context, now time.Time) (result []service.AccountConsumption, err error) {
	query := `
		SELECT
			account_id,
			COUNT(*) FILTER (WHERE created_at >= $2) AS requests_24h,
			COALESCE(SUM(COALESCE(account_stats_cost, total_cost) * COALESCE(account_rate_multiplier, 1)) FILTER (WHERE created_at >= $2), 0) AS cost_24h,
			COUNT(*) AS requests_7d,
			COALESCE(SUM(COALESCE(account_stats_cost, total_cost) * COALESCE(account_rate_multiplier, 1)), 0) AS cost_7d,
			MIN(created_at) AS first_used_at
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $3
		GROUP BY account_id
	`
	rows, err := r.sql.QueryContext(ctx, query, now.Add(-7*24*time.Hour), now.Add(-24*time.Hour), now)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			result = nil
		}
	}()
	result = make([]service.AccountConsumption, 0)
	for rows.Next() {
		var item service.AccountConsumption
		if err = rows.Scan(&item.AccountID, &item.Requests24h, &item.Cost24h, &item.Requests7d, &item.Cost7d, &item.FirstUsedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// GetAccountTrendPoints 按时间桶 × 账号聚合请求、账号成本与失败数，并带出账号 base_url 供按站点合并。
// granularity 只允许 hour / day（调用方保证）；loc 为 IANA 时区名（调用方校验）。
func (r *usageLogRepository) GetAccountTrendPoints(ctx context.Context, startTime, endTime time.Time, granularity, loc string) (result []service.AccountTrendPoint, err error) {
	query := `
		WITH usage_points AS (
			SELECT
				date_trunc($3, u.created_at AT TIME ZONE $4) AS bucket,
				u.account_id,
				COUNT(*) AS requests,
				COALESCE(SUM(COALESCE(u.account_stats_cost, u.total_cost) * COALESCE(u.account_rate_multiplier, 1)), 0) AS account_cost,
				COALESCE(SUM(u.input_tokens + u.output_tokens + u.cache_creation_tokens + u.cache_read_tokens), 0) AS tokens
			FROM usage_logs u
			WHERE u.created_at >= $1 AND u.created_at < $2
			GROUP BY 1, 2
		),
		error_points AS (
			SELECT
				date_trunc($3, e.created_at AT TIME ZONE $4) AS bucket,
				e.account_id,
				COUNT(*) AS errors
			FROM ops_error_logs e
			WHERE e.created_at >= $1 AND e.created_at < $2
				AND e.account_id IS NOT NULL
				AND COALESCE(e.status_code, 0) >= 400
				AND NOT e.is_business_limited
				AND NOT e.is_count_tokens
			GROUP BY 1, 2
		)
		SELECT
			TO_CHAR(COALESCE(up.bucket, ep.bucket), 'YYYY-MM-DD"T"HH24:MI:SS') AS bucket,
			COALESCE(up.account_id, ep.account_id) AS account_id,
			COALESCE(a.name, ''),
			COALESCE(BTRIM(a.credentials->>'base_url'), ''),
			COALESCE(up.requests, 0),
			COALESCE(up.account_cost, 0),
			COALESCE(up.tokens, 0),
			COALESCE(ep.errors, 0)
		FROM usage_points up
		FULL OUTER JOIN error_points ep ON ep.bucket = up.bucket AND ep.account_id = up.account_id
		LEFT JOIN accounts a ON a.id = COALESCE(up.account_id, ep.account_id)
		ORDER BY 1
	`
	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime, granularity, loc)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			result = nil
		}
	}()
	result = make([]service.AccountTrendPoint, 0)
	for rows.Next() {
		var p service.AccountTrendPoint
		if err = rows.Scan(&p.Bucket, &p.AccountID, &p.Name, &p.BaseURL, &p.Requests, &p.AccountCost, &p.Tokens, &p.Errors); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// GetRequestHeatmap 按星期（1=周一 … 7=周日）× 小时聚合请求数、账号成本与失败数。
func (r *usageLogRepository) GetRequestHeatmap(ctx context.Context, startTime, endTime time.Time, loc string) (result []service.HeatmapCell, err error) {
	query := `
		WITH usage_cells AS (
			SELECT
				EXTRACT(ISODOW FROM created_at AT TIME ZONE $3)::int AS dow,
				EXTRACT(HOUR FROM created_at AT TIME ZONE $3)::int AS hour,
				COUNT(*) AS requests,
				COALESCE(SUM(COALESCE(account_stats_cost, total_cost) * COALESCE(account_rate_multiplier, 1)), 0) AS account_cost
			FROM usage_logs
			WHERE created_at >= $1 AND created_at < $2
			GROUP BY 1, 2
		),
		error_cells AS (
			SELECT
				EXTRACT(ISODOW FROM created_at AT TIME ZONE $3)::int AS dow,
				EXTRACT(HOUR FROM created_at AT TIME ZONE $3)::int AS hour,
				COUNT(*) AS errors
			FROM ops_error_logs
			WHERE created_at >= $1 AND created_at < $2
				AND account_id IS NOT NULL
				AND COALESCE(status_code, 0) >= 400
				AND NOT is_business_limited
				AND NOT is_count_tokens
			GROUP BY 1, 2
		)
		SELECT
			COALESCE(u.dow, e.dow), COALESCE(u.hour, e.hour),
			COALESCE(u.requests, 0), COALESCE(u.account_cost, 0), COALESCE(e.errors, 0)
		FROM usage_cells u
		FULL OUTER JOIN error_cells e ON e.dow = u.dow AND e.hour = u.hour
		ORDER BY 1, 2
	`
	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime, loc)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			result = nil
		}
	}()
	result = make([]service.HeatmapCell, 0, 7*24)
	for rows.Next() {
		var c service.HeatmapCell
		if err = rows.Scan(&c.Weekday, &c.Hour, &c.Requests, &c.AccountCost, &c.Errors); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
