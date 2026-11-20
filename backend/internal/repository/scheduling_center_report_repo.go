package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

// 调度中心对比与导出查询（schedulingCenterRepository 同时实现 service.SchedulingCenterReportRepository）。

var _ service.SchedulingCenterReportRepository = (*schedulingCenterRepository)(nil)

func bucketSeconds(bucket time.Duration) int64 {
	s := int64(bucket / time.Second)
	if s <= 0 {
		return 600
	}
	return s
}

func (r *schedulingCenterRepository) GetUsageBuckets(ctx context.Context, accountIDs []int64, from, to time.Time, bucket time.Duration, slowMs int64) ([]service.SchedulingCenterUsageBucket, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT account_id,
		       to_timestamp(floor(extract(epoch FROM created_at)::double precision / $4::double precision) * $4::double precision) AS bucket,
		       COUNT(*),
		       COALESCE(SUM(input_tokens), 0),
		       COALESCE(SUM(cache_creation_tokens), 0),
		       COALESCE(SUM(cache_read_tokens), 0),
		       COUNT(first_token_ms),
		       COALESCE(SUM(first_token_ms), 0),
		       COUNT(*) FILTER (WHERE first_token_ms > $5)
		FROM usage_logs
		WHERE account_id = ANY($1) AND created_at >= $2 AND created_at < $3
		GROUP BY account_id, bucket`, pq.Array(accountIDs), from, to, bucketSeconds(bucket), slowMs)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.SchedulingCenterUsageBucket, 0)
	for rows.Next() {
		var b service.SchedulingCenterUsageBucket
		if err := rows.Scan(&b.AccountID, &b.Bucket, &b.Requests, &b.InputTokens, &b.CacheCreationTokens,
			&b.CacheReadTokens, &b.FirstTokenSamples, &b.FirstTokenSumMs, &b.SlowRequests); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *schedulingCenterRepository) GetErrorBuckets(ctx context.Context, accountIDs []int64, from, to time.Time, bucket time.Duration) ([]service.SchedulingCenterErrorBucket, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT account_id,
		       to_timestamp(floor(extract(epoch FROM created_at)::double precision / $4::double precision) * $4::double precision) AS bucket,
		       COUNT(*)
		FROM ops_error_logs
		WHERE account_id = ANY($1) AND created_at >= $2 AND created_at < $3
		  AND error_owner = 'provider' AND NOT is_business_limited
		GROUP BY account_id, bucket`, pq.Array(accountIDs), from, to, bucketSeconds(bucket))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.SchedulingCenterErrorBucket, 0)
	for rows.Next() {
		var b service.SchedulingCenterErrorBucket
		if err := rows.Scan(&b.AccountID, &b.Bucket, &b.Errors); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *schedulingCenterRepository) GetIQBuckets(ctx context.Context, accountIDs []int64, from, to time.Time, bucket time.Duration) ([]service.SchedulingCenterIQBucket, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT p.account_id,
		       to_timestamp(floor(extract(epoch FROM r.created_at)::double precision / $4::double precision) * $4::double precision) AS bucket,
		       COUNT(*),
		       COUNT(*) FILTER (WHERE r.iq_status = 'passed'),
		       COUNT(*) FILTER (WHERE r.iq_status = 'incorrect'),
		       COUNT(*) FILTER (WHERE r.iq_status = 'error'),
		       COUNT(r.first_token_ms),
		       COALESCE(SUM(r.first_token_ms), 0)
		FROM scheduled_test_results r
		JOIN scheduled_test_plans p ON p.id = r.plan_id
		WHERE p.account_id = ANY($1) AND r.iq_status <> '' AND r.created_at >= $2 AND r.created_at < $3
		GROUP BY p.account_id, bucket`, pq.Array(accountIDs), from, to, bucketSeconds(bucket))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.SchedulingCenterIQBucket, 0)
	for rows.Next() {
		var b service.SchedulingCenterIQBucket
		if err := rows.Scan(&b.AccountID, &b.Bucket, &b.Checks, &b.Passed, &b.Incorrect, &b.Errored,
			&b.FirstTokenSamples, &b.FirstTokenSumMs); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *schedulingCenterRepository) ListMembershipEvents(ctx context.Context, groupID int64, from, to time.Time) (map[int64]string, []service.SchedulingCenterMembershipEvent, error) {
	prior := map[int64]string{}
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT DISTINCT ON (account_id) account_id, action
		FROM scheduling_center_pool_events
		WHERE group_id = $1 AND action IN ('enter', 'exit') AND created_at < $2
		ORDER BY account_id, created_at DESC, id DESC`, groupID, from)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var (
			id     int64
			action string
		)
		if err := rows.Scan(&id, &action); err != nil {
			_ = rows.Close()
			return nil, nil, err
		}
		prior[id] = action
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, nil, err
	}
	_ = rows.Close()

	rows, err = r.db(ctx).QueryContext(ctx, `
		SELECT account_id, action, created_at
		FROM scheduling_center_pool_events
		WHERE group_id = $1 AND action IN ('enter', 'exit') AND created_at >= $2 AND created_at < $3
		ORDER BY created_at, id`, groupID, from, to)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()
	events := make([]service.SchedulingCenterMembershipEvent, 0)
	for rows.Next() {
		var e service.SchedulingCenterMembershipEvent
		if err := rows.Scan(&e.AccountID, &e.Action, &e.At); err != nil {
			return nil, nil, err
		}
		events = append(events, e)
	}
	return prior, events, rows.Err()
}

func (r *schedulingCenterRepository) ListEventsInRange(ctx context.Context, groupID int64, from, to time.Time, limit int) ([]service.SchedulingCenterEvent, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT e.id, e.group_id, e.account_id, a.name, e.action, e.source, e.reason, e.detail, e.is_bad, e.created_at
		FROM scheduling_center_pool_events e
		JOIN accounts a ON a.id = e.account_id
		WHERE e.group_id = $1 AND e.created_at >= $2 AND e.created_at < $3
		ORDER BY e.created_at, e.id
		LIMIT $4`, groupID, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.SchedulingCenterEvent, 0)
	for rows.Next() {
		var (
			e      service.SchedulingCenterEvent
			detail []byte
		)
		if err := rows.Scan(&e.ID, &e.GroupID, &e.AccountID, &e.AccountName, &e.Action, &e.Source, &e.Reason, &detail, &e.IsBad, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Detail = map[string]any{}
		if len(detail) > 0 {
			_ = json.Unmarshal(detail, &e.Detail)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *schedulingCenterRepository) ListIQRecordsInRange(ctx context.Context, accountIDs []int64, from, to time.Time, limit int) ([]service.SchedulingCenterExportIQRecord, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT p.account_id, r.started_at, r.status, r.iq_status, r.first_token_ms, r.latency_ms, LEFT(r.error_message, 300)
		FROM scheduled_test_results r
		JOIN scheduled_test_plans p ON p.id = r.plan_id
		WHERE p.account_id = ANY($1) AND r.iq_status <> '' AND r.created_at >= $2 AND r.created_at < $3
		ORDER BY r.created_at, r.id
		LIMIT $4`, pq.Array(accountIDs), from, to, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.SchedulingCenterExportIQRecord, 0)
	for rows.Next() {
		var rec service.SchedulingCenterExportIQRecord
		if err := rows.Scan(&rec.AccountID, &rec.StartedAt, &rec.Status, &rec.IQStatus, &rec.FirstTokenMs, &rec.LatencyMs, &rec.ErrorMessage); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *schedulingCenterRepository) GetGroupBrief(ctx context.Context, groupID int64) (*service.SchedulingCenterGroupBrief, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT id, name, platform, status FROM groups WHERE id = $1`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return &service.SchedulingCenterGroupBrief{ID: groupID}, rows.Err()
	}
	var g service.SchedulingCenterGroupBrief
	if err := rows.Scan(&g.ID, &g.Name, &g.Platform, &g.Status); err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *schedulingCenterRepository) ListGroupAccountsBrief(ctx context.Context, groupID int64) ([]service.SchedulingCenterAccountBrief, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT a.id, a.name, a.platform, a.type, a.status, a.schedulable, a.rate_multiplier::double precision,
		       a.rate_limit_reset_at, a.overload_until, a.temp_unschedulable_until, LEFT(COALESCE(a.error_message, ''), 300)
		FROM account_groups ag
		JOIN accounts a ON a.id = ag.account_id AND a.deleted_at IS NULL
		WHERE ag.group_id = $1
		ORDER BY a.id`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.SchedulingCenterAccountBrief, 0)
	for rows.Next() {
		var (
			b                          service.SchedulingCenterAccountBrief
			multiplier                 sql.NullFloat64
			rateLimit, overload, unsch sql.NullTime
		)
		if err := rows.Scan(&b.ID, &b.Name, &b.Platform, &b.Type, &b.Status, &b.Schedulable, &multiplier,
			&rateLimit, &overload, &unsch, &b.ErrorMessage); err != nil {
			return nil, err
		}
		if multiplier.Valid {
			v := multiplier.Float64
			b.RateMultiplier = &v
		}
		if rateLimit.Valid {
			v := rateLimit.Time
			b.RateLimitResetAt = &v
		}
		if overload.Valid {
			v := overload.Time
			b.OverloadUntil = &v
		}
		if unsch.Valid {
			v := unsch.Time
			b.TempUnschedulableUntil = &v
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
