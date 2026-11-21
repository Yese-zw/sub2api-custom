package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type schedulingCenterRepository struct {
	client *dbent.Client
}

// NewSchedulingCenterRepository creates the scheduling center repository.
func NewSchedulingCenterRepository(client *dbent.Client) service.SchedulingCenterRepository {
	return &schedulingCenterRepository{client: client}
}

func (r *schedulingCenterRepository) db(ctx context.Context) *dbent.Client {
	return clientFromContext(ctx, r.client)
}

// ---------------------------------------------------------------------------
// 调度池
// ---------------------------------------------------------------------------

func (r *schedulingCenterRepository) ListPoolMembers(ctx context.Context, groupID int64) ([]service.SchedulingCenterPoolMember, error) {
	// Key 类型账号来自调度池表；OAuth / Setup Token 账号固定在调度池，按加入分组的时间虚拟加入。
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT account_id, source, created_at FROM (
			SELECT m.account_id, m.source, m.created_at
			FROM scheduling_center_pool_members m
			JOIN account_groups ag ON ag.group_id = m.group_id AND ag.account_id = m.account_id
			JOIN accounts a ON a.id = m.account_id
			WHERE m.group_id = $1 AND a.type NOT IN `+schedulingCenterFixedTypesSQL+`
			UNION ALL
			SELECT ag.account_id, $2, ag.created_at
			FROM account_groups ag
			JOIN accounts a ON a.id = ag.account_id AND a.deleted_at IS NULL
			WHERE ag.group_id = $1 AND a.type IN `+schedulingCenterFixedTypesSQL+`
		) x
		ORDER BY created_at, account_id`, groupID, service.SchedulingCenterSourceFixed)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.SchedulingCenterPoolMember, 0)
	for rows.Next() {
		var m service.SchedulingCenterPoolMember
		if err := rows.Scan(&m.AccountID, &m.Source, &m.EnteredAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func scanInt64Column(rows *sql.Rows) ([]int64, error) {
	defer func() { _ = rows.Close() }()
	out := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *schedulingCenterRepository) AddToSchedulingPool(ctx context.Context, groupID int64, accountIDs []int64, source string) ([]int64, error) {
	if len(accountIDs) == 0 {
		return []int64{}, nil
	}
	// 只接受确实属于该分组的账号。手动移入会把已有的 rule 成员转为 manual；
	// 规则 / 恢复移入不会覆盖已有成员的来源。新入池时 created_at 即入池时间。
	conflict := `DO NOTHING`
	if source == service.SchedulingCenterSourceManual {
		conflict = `DO UPDATE SET source = EXCLUDED.source`
	}
	// OAuth / Setup Token 账号固定在调度池，不写入调度池表。
	rows, err := r.db(ctx).QueryContext(ctx, `
		INSERT INTO scheduling_center_pool_members (group_id, account_id, source)
		SELECT ag.group_id, ag.account_id, $3
		FROM account_groups ag
		JOIN accounts a ON a.id = ag.account_id
		WHERE ag.group_id = $1 AND ag.account_id = ANY($2) AND a.type NOT IN `+schedulingCenterFixedTypesSQL+`
		ON CONFLICT (group_id, account_id) `+conflict+`
		RETURNING account_id`, groupID, pq.Array(accountIDs), source)
	if err != nil {
		return nil, err
	}
	return scanInt64Column(rows)
}

func (r *schedulingCenterRepository) RemoveFromSchedulingPool(ctx context.Context, groupID int64, accountIDs []int64) ([]int64, error) {
	if len(accountIDs) == 0 {
		return []int64{}, nil
	}
	rows, err := r.db(ctx).QueryContext(ctx, `
		DELETE FROM scheduling_center_pool_members
		WHERE group_id = $1 AND account_id = ANY($2)
		RETURNING account_id`, groupID, pq.Array(accountIDs))
	if err != nil {
		return nil, err
	}
	return scanInt64Column(rows)
}

// ---------------------------------------------------------------------------
// 指标
// ---------------------------------------------------------------------------

func (r *schedulingCenterRepository) GetAccountStats(ctx context.Context, accountIDs []int64, since time.Time) (map[int64]service.SchedulingCenterAccountStats, error) {
	result := make(map[int64]service.SchedulingCenterAccountStats, len(accountIDs))
	if len(accountIDs) == 0 {
		return result, nil
	}
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT
			account_id,
			COUNT(*) AS requests,
			COALESCE(SUM(input_tokens), 0) AS input_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) AS cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) AS cache_read_tokens,
			COUNT(first_token_ms) AS first_token_samples,
			AVG(first_token_ms)::double precision AS avg_first_token_ms
		FROM usage_logs
		WHERE account_id = ANY($1) AND created_at >= $2
		GROUP BY account_id`, pq.Array(accountIDs), since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			accountID int64
			st        service.SchedulingCenterAccountStats
			avgTTFT   sql.NullFloat64
		)
		if err := rows.Scan(
			&accountID,
			&st.Requests,
			&st.InputTokens,
			&st.CacheCreationTokens,
			&st.CacheReadTokens,
			&st.FirstTokenSamples,
			&avgTTFT,
		); err != nil {
			return nil, err
		}
		if avgTTFT.Valid {
			v := math.Round(avgTTFT.Float64)
			st.AvgFirstTokenMs = &v
		}
		result[accountID] = st
	}
	return result, rows.Err()
}

func (r *schedulingCenterRepository) GetLatestIQResults(ctx context.Context, accountIDs []int64) (map[int64]service.SchedulingCenterIQResult, error) {
	result := make(map[int64]service.SchedulingCenterIQResult, len(accountIDs))
	if len(accountIDs) == 0 {
		return result, nil
	}
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT DISTINCT ON (p.account_id)
			p.account_id, r.iq_status, r.iq_message, r.finished_at
		FROM scheduled_test_results r
		JOIN scheduled_test_plans p ON p.id = r.plan_id
		WHERE p.account_id = ANY($1) AND r.iq_status <> ''
		ORDER BY p.account_id, r.created_at DESC`, pq.Array(accountIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			accountID int64
			res       service.SchedulingCenterIQResult
		)
		if err := rows.Scan(&accountID, &res.Status, &res.Message, &res.CheckedAt); err != nil {
			return nil, err
		}
		result[accountID] = res
	}
	return result, rows.Err()
}

func (r *schedulingCenterRepository) GetRecentTestFirstTokenAvg(ctx context.Context, accountIDs []int64, lastN int) (map[int64]float64, error) {
	result := make(map[int64]float64, len(accountIDs))
	if len(accountIDs) == 0 {
		return result, nil
	}
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT account_id, AVG(first_token_ms)::double precision
		FROM (
			SELECT p.account_id, r.first_token_ms,
			       ROW_NUMBER() OVER (PARTITION BY p.account_id ORDER BY r.created_at DESC) AS rn
			FROM scheduled_test_results r
			JOIN scheduled_test_plans p ON p.id = r.plan_id
			WHERE p.account_id = ANY($1) AND r.first_token_ms IS NOT NULL
		) recent
		WHERE rn <= $2
		GROUP BY account_id`, pq.Array(accountIDs), lastN)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			accountID int64
			avg       float64
		)
		if err := rows.Scan(&accountID, &avg); err != nil {
			return nil, err
		}
		result[accountID] = math.Round(avg)
	}
	return result, rows.Err()
}

func (r *schedulingCenterRepository) ListAccountTestRecords(ctx context.Context, accountID int64, limit int) ([]service.SchedulingCenterTestRecord, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT r.id, r.status, r.first_token_ms, r.probe_total_ms, r.latency_ms, r.iq_status, r.iq_message, r.error_message, r.started_at
		FROM scheduled_test_results r
		JOIN scheduled_test_plans p ON p.id = r.plan_id
		WHERE p.account_id = $1 AND p.iq_check = true
		ORDER BY r.created_at DESC
		LIMIT $2`, accountID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.SchedulingCenterTestRecord, 0)
	for rows.Next() {
		var rec service.SchedulingCenterTestRecord
		if err := rows.Scan(&rec.ID, &rec.Status, &rec.FirstTokenMs, &rec.ProbeTotalMs, &rec.LatencyMs, &rec.IQStatus, &rec.IQMessage, &rec.ErrorMessage, &rec.StartedAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// 分组设置
// ---------------------------------------------------------------------------

const schedulingCenterSettingsColumns = `
	s.group_id, s.enabled, s.cron_expression, s.model_id, s.rule_enabled,
	s.min_cache_rate, s.require_iq_pass, s.max_first_token_ms,
	s.exit_enabled, s.exit_cache_drop_points, s.exit_cache_min_requests, s.exit_iq_window,
	s.exit_iq_max_degraded, s.exit_latency_ms, s.exit_latency_count, s.exit_error_count,
	s.exit_latency_window, s.exit_error_window,
	s.recover_success_count,
	s.first_token_test_samples, s.entry_cache_min_requests, s.entry_iq_pass_count,
	s.exit_traffic_window_hours, s.bad_window_minutes, s.cooldown_minutes, s.recover_latency_ms,
	s.fallback_enabled, s.fallback_min_accounts, s.fallback_order, s.fallback_require_iq_pass,
	s.manual_out_hours, s.record_retention, s.event_retention_days, s.enforce_enabled,
	s.updated_at`

func scanSchedulingCenterSettings(row interface{ Scan(dest ...any) error }) (*service.SchedulingCenterGroupSettings, error) {
	var (
		gs           service.SchedulingCenterGroupSettings
		minCacheRate sql.NullFloat64
		maxTTFT      sql.NullInt64
		exitDrop     sql.NullFloat64
		exitIQMax    sql.NullInt64
		exitLatency  sql.NullInt64
		exitErrors   sql.NullInt64
		cooldowns    pq.Int64Array
		recoverTTFT  sql.NullInt64
		fbOrder      pq.StringArray
	)
	if err := row.Scan(&gs.GroupID, &gs.Enabled, &gs.CronExpression, &gs.ModelID, &gs.RuleEnabled,
		&minCacheRate, &gs.RequireIQPass, &maxTTFT,
		&gs.ExitEnabled, &exitDrop, &gs.ExitCacheMinRequests, &gs.ExitIQWindow,
		&exitIQMax, &exitLatency, &gs.ExitLatencyCount, &exitErrors,
		&gs.ExitLatencyWindow, &gs.ExitErrorWindow,
		&gs.RecoverSuccessCount,
		&gs.FirstTokenTestSamples, &gs.EntryCacheMinRequests, &gs.EntryIQPassCount,
		&gs.ExitTrafficWindowHours, &gs.BadWindowMinutes, &cooldowns, &recoverTTFT,
		&gs.FallbackEnabled, &gs.FallbackMinAccounts, &fbOrder, &gs.FallbackRequireIQPass,
		&gs.ManualOutHours, &gs.RecordRetention, &gs.EventRetentionDays, &gs.EnforceEnabled,
		&gs.UpdatedAt); err != nil {
		return nil, err
	}
	gs.FallbackOrder = append([]string{}, fbOrder...)
	gs.CooldownMinutes = make([]int, 0, len(cooldowns))
	for _, m := range cooldowns {
		gs.CooldownMinutes = append(gs.CooldownMinutes, int(m))
	}
	if recoverTTFT.Valid {
		v := recoverTTFT.Int64
		gs.RecoverLatencyMs = &v
	}
	if exitDrop.Valid {
		v := exitDrop.Float64
		gs.ExitCacheDropPoints = &v
	}
	if exitIQMax.Valid {
		v := int(exitIQMax.Int64)
		gs.ExitIQMaxDegraded = &v
	}
	if exitLatency.Valid {
		v := exitLatency.Int64
		gs.ExitLatencyMs = &v
	}
	if exitErrors.Valid {
		v := int(exitErrors.Int64)
		gs.ExitErrorCount = &v
	}
	if minCacheRate.Valid {
		v := minCacheRate.Float64
		gs.MinCacheRate = &v
	}
	if maxTTFT.Valid {
		v := maxTTFT.Int64
		gs.MaxFirstTokenMs = &v
	}
	return &gs, nil
}

func (r *schedulingCenterRepository) ListGroupSettings(ctx context.Context) ([]service.SchedulingCenterGroupSettings, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT `+schedulingCenterSettingsColumns+`
		FROM scheduling_center_group_settings s
		JOIN groups g ON g.id = s.group_id AND g.deleted_at IS NULL
		ORDER BY s.group_id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.SchedulingCenterGroupSettings, 0)
	for rows.Next() {
		gs, err := scanSchedulingCenterSettings(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *gs)
	}
	return out, rows.Err()
}

func (r *schedulingCenterRepository) GetGroupSettings(ctx context.Context, groupID int64) (*service.SchedulingCenterGroupSettings, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT `+schedulingCenterSettingsColumns+`
		FROM scheduling_center_group_settings s
		WHERE s.group_id = $1`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, rows.Err()
	}
	return scanSchedulingCenterSettings(rows)
}

func (r *schedulingCenterRepository) UpsertGroupSettings(ctx context.Context, in *service.SchedulingCenterGroupSettings) (*service.SchedulingCenterGroupSettings, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		INSERT INTO scheduling_center_group_settings AS s
			(group_id, enabled, cron_expression, model_id, rule_enabled, min_cache_rate, require_iq_pass, max_first_token_ms,
			 exit_enabled, exit_cache_drop_points, exit_cache_min_requests, exit_iq_window, exit_iq_max_degraded,
			 exit_latency_ms, exit_latency_count, exit_error_count, recover_success_count,
			 first_token_test_samples, entry_cache_min_requests, entry_iq_pass_count,
			 exit_traffic_window_hours, bad_window_minutes, cooldown_minutes, recover_latency_ms,
			 fallback_enabled, fallback_min_accounts, fallback_order, fallback_require_iq_pass, manual_out_hours,
			 record_retention, event_retention_days, exit_latency_window, exit_error_window, enforce_enabled, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17,
		        $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, NOW())
		ON CONFLICT (group_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			cron_expression = EXCLUDED.cron_expression,
			model_id = EXCLUDED.model_id,
			rule_enabled = EXCLUDED.rule_enabled,
			min_cache_rate = EXCLUDED.min_cache_rate,
			require_iq_pass = EXCLUDED.require_iq_pass,
			max_first_token_ms = EXCLUDED.max_first_token_ms,
			exit_enabled = EXCLUDED.exit_enabled,
			exit_cache_drop_points = EXCLUDED.exit_cache_drop_points,
			exit_cache_min_requests = EXCLUDED.exit_cache_min_requests,
			exit_iq_window = EXCLUDED.exit_iq_window,
			exit_iq_max_degraded = EXCLUDED.exit_iq_max_degraded,
			exit_latency_ms = EXCLUDED.exit_latency_ms,
			exit_latency_count = EXCLUDED.exit_latency_count,
			exit_error_count = EXCLUDED.exit_error_count,
			exit_latency_window = EXCLUDED.exit_latency_window,
			exit_error_window = EXCLUDED.exit_error_window,
			recover_success_count = EXCLUDED.recover_success_count,
			first_token_test_samples = EXCLUDED.first_token_test_samples,
			entry_cache_min_requests = EXCLUDED.entry_cache_min_requests,
			entry_iq_pass_count = EXCLUDED.entry_iq_pass_count,
			exit_traffic_window_hours = EXCLUDED.exit_traffic_window_hours,
			bad_window_minutes = EXCLUDED.bad_window_minutes,
			cooldown_minutes = EXCLUDED.cooldown_minutes,
			recover_latency_ms = EXCLUDED.recover_latency_ms,
			fallback_enabled = EXCLUDED.fallback_enabled,
			fallback_min_accounts = EXCLUDED.fallback_min_accounts,
			fallback_order = EXCLUDED.fallback_order,
			fallback_require_iq_pass = EXCLUDED.fallback_require_iq_pass,
			manual_out_hours = EXCLUDED.manual_out_hours,
			record_retention = EXCLUDED.record_retention,
			event_retention_days = EXCLUDED.event_retention_days,
			enforce_enabled = EXCLUDED.enforce_enabled,
			updated_at = NOW()
		RETURNING `+schedulingCenterSettingsColumns,
		in.GroupID, in.Enabled, in.CronExpression, in.ModelID, in.RuleEnabled, in.MinCacheRate, in.RequireIQPass, in.MaxFirstTokenMs,
		in.ExitEnabled, in.ExitCacheDropPoints, in.ExitCacheMinRequests, in.ExitIQWindow, in.ExitIQMaxDegraded,
		in.ExitLatencyMs, in.ExitLatencyCount, in.ExitErrorCount, in.RecoverSuccessCount,
		in.FirstTokenTestSamples, in.EntryCacheMinRequests, in.EntryIQPassCount,
		in.ExitTrafficWindowHours, in.BadWindowMinutes, schedulingCenterCooldownArray(in.CooldownMinutes), in.RecoverLatencyMs,
		in.FallbackEnabled, in.FallbackMinAccounts, pq.StringArray(in.FallbackOrder), in.FallbackRequireIQPass,
		in.ManualOutHours, in.RecordRetention, in.EventRetentionDays, in.ExitLatencyWindow, in.ExitErrorWindow, in.EnforceEnabled)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, errors.New("upsert scheduling center settings returned no row")
	}
	return scanSchedulingCenterSettings(rows)
}

// schedulingCenterFixedTypesSQL 是固定在调度池的账号类型（OAuth / Setup Token）。
const schedulingCenterFixedTypesSQL = `('` + service.AccountTypeOAuth + `', '` + service.AccountTypeSetupToken + `')`

func (r *schedulingCenterRepository) ListGroupKeyAccountIDs(ctx context.Context, groupID int64) ([]int64, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT ag.account_id
		FROM account_groups ag
		JOIN accounts a ON a.id = ag.account_id AND a.deleted_at IS NULL
		WHERE ag.group_id = $1 AND a.type NOT IN `+schedulingCenterFixedTypesSQL+`
		ORDER BY ag.account_id`, groupID)
	if err != nil {
		return nil, err
	}
	return scanInt64Column(rows)
}

func (r *schedulingCenterRepository) ListGroupAccountIDs(ctx context.Context, groupID int64) ([]int64, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT ag.account_id
		FROM account_groups ag
		JOIN accounts a ON a.id = ag.account_id AND a.deleted_at IS NULL
		WHERE ag.group_id = $1
		ORDER BY ag.account_id`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// 托管的定时测试计划
// ---------------------------------------------------------------------------

func (r *schedulingCenterRepository) ListManagedPlans(ctx context.Context) ([]service.SchedulingCenterManagedPlan, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT m.account_id, m.plan_id, m.group_id, p.cron_expression, p.model_id, p.iq_stream, p.max_results
		FROM scheduling_center_account_plans m
		JOIN scheduled_test_plans p ON p.id = m.plan_id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.SchedulingCenterManagedPlan, 0)
	for rows.Next() {
		var mp service.SchedulingCenterManagedPlan
		if err := rows.Scan(&mp.AccountID, &mp.PlanID, &mp.GroupID, &mp.CronExpression, &mp.ModelID, &mp.IQStream, &mp.MaxResults); err != nil {
			return nil, err
		}
		out = append(out, mp)
	}
	return out, rows.Err()
}

func (r *schedulingCenterRepository) InsertManagedPlan(ctx context.Context, accountID, planID, groupID int64) error {
	_, err := r.db(ctx).ExecContext(ctx, `
		INSERT INTO scheduling_center_account_plans (account_id, plan_id, group_id)
		VALUES ($1, $2, $3)`, accountID, planID, groupID)
	return err
}

func (r *schedulingCenterRepository) UpdateManagedPlanGroup(ctx context.Context, accountID, groupID int64) error {
	_, err := r.db(ctx).ExecContext(ctx, `
		UPDATE scheduling_center_account_plans SET group_id = $2 WHERE account_id = $1`, accountID, groupID)
	return err
}

func (r *schedulingCenterRepository) GetManagedPlanID(ctx context.Context, accountID int64) (int64, bool, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT plan_id FROM scheduling_center_account_plans WHERE account_id = $1`, accountID)
	if err != nil {
		return 0, false, err
	}
	ids, err := scanInt64Column(rows)
	if err != nil || len(ids) == 0 {
		return 0, false, err
	}
	return ids[0], true, nil
}

// ---------------------------------------------------------------------------
// 账号状态
// ---------------------------------------------------------------------------

const schedulingCenterStateColumns = `
	group_id, account_id, state, prev_source, reason_code, cooldown_until,
	probe_status, probe_started_at, probe_passed_at, recover_ok_count, updated_at`

func scanSchedulingCenterState(row interface{ Scan(dest ...any) error }) (*service.SchedulingCenterAccountState, error) {
	var (
		st                                  service.SchedulingCenterAccountState
		cooldownUntil, probeStart, probePas sql.NullTime
	)
	if err := row.Scan(&st.GroupID, &st.AccountID, &st.State, &st.PrevSource, &st.ReasonCode, &cooldownUntil,
		&st.ProbeStatus, &probeStart, &probePas, &st.RecoverOKCount, &st.UpdatedAt); err != nil {
		return nil, err
	}
	if cooldownUntil.Valid {
		v := cooldownUntil.Time
		st.CooldownUntil = &v
	}
	if probeStart.Valid {
		v := probeStart.Time
		st.ProbeStartedAt = &v
	}
	if probePas.Valid {
		v := probePas.Time
		st.ProbePassedAt = &v
	}
	return &st, nil
}

func (r *schedulingCenterRepository) ListAccountStates(ctx context.Context, groupID int64) ([]service.SchedulingCenterAccountState, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT `+schedulingCenterStateColumns+`
		FROM scheduling_center_account_states
		WHERE group_id = $1
		ORDER BY account_id`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.SchedulingCenterAccountState, 0)
	for rows.Next() {
		st, err := scanSchedulingCenterState(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *st)
	}
	return out, rows.Err()
}

func (r *schedulingCenterRepository) GetAccountState(ctx context.Context, groupID, accountID int64) (*service.SchedulingCenterAccountState, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT `+schedulingCenterStateColumns+`
		FROM scheduling_center_account_states
		WHERE group_id = $1 AND account_id = $2`, groupID, accountID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return nil, rows.Err()
	}
	return scanSchedulingCenterState(rows)
}

func (r *schedulingCenterRepository) UpsertAccountState(ctx context.Context, st *service.SchedulingCenterAccountState) error {
	_, err := r.db(ctx).ExecContext(ctx, `
		INSERT INTO scheduling_center_account_states
			(group_id, account_id, state, prev_source, reason_code, cooldown_until,
			 probe_status, probe_started_at, probe_passed_at, recover_ok_count, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (group_id, account_id) DO UPDATE SET
			state = EXCLUDED.state,
			prev_source = EXCLUDED.prev_source,
			reason_code = EXCLUDED.reason_code,
			cooldown_until = EXCLUDED.cooldown_until,
			probe_status = EXCLUDED.probe_status,
			probe_started_at = EXCLUDED.probe_started_at,
			probe_passed_at = EXCLUDED.probe_passed_at,
			recover_ok_count = EXCLUDED.recover_ok_count,
			updated_at = NOW()`,
		st.GroupID, st.AccountID, st.State, st.PrevSource, st.ReasonCode, st.CooldownUntil,
		st.ProbeStatus, st.ProbeStartedAt, st.ProbePassedAt, st.RecoverOKCount)
	return err
}

func (r *schedulingCenterRepository) CompareAndSetProbingState(ctx context.Context, st *service.SchedulingCenterAccountState, expectProbeStartedAt time.Time) (bool, error) {
	res, err := r.db(ctx).ExecContext(ctx, `
		UPDATE scheduling_center_account_states SET
			state = $3, prev_source = $4, reason_code = $5, cooldown_until = $6,
			probe_status = $7, probe_started_at = $8, probe_passed_at = $9, recover_ok_count = $10,
			updated_at = NOW()
		WHERE group_id = $1 AND account_id = $2 AND state = $11 AND probe_started_at = $12`,
		st.GroupID, st.AccountID, st.State, st.PrevSource, st.ReasonCode, st.CooldownUntil,
		st.ProbeStatus, st.ProbeStartedAt, st.ProbePassedAt, st.RecoverOKCount,
		service.SchedulingCenterStateProbing, expectProbeStartedAt)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (r *schedulingCenterRepository) DeleteAccountStates(ctx context.Context, groupID int64, accountIDs []int64) error {
	if len(accountIDs) == 0 {
		return nil
	}
	_, err := r.db(ctx).ExecContext(ctx, `
		DELETE FROM scheduling_center_account_states
		WHERE group_id = $1 AND account_id = ANY($2)`, groupID, pq.Array(accountIDs))
	return err
}

// ---------------------------------------------------------------------------
// 调度日志
// ---------------------------------------------------------------------------

func (r *schedulingCenterRepository) InsertEvents(ctx context.Context, events []service.SchedulingCenterEvent) error {
	if len(events) == 0 {
		return nil
	}
	var (
		sb   strings.Builder
		args = make([]any, 0, len(events)*8)
	)
	sb.WriteString(`INSERT INTO scheduling_center_pool_events
		(group_id, account_id, action, source, reason, detail, is_bad, created_at) VALUES `)
	for i, e := range events {
		detail, err := json.Marshal(e.Detail)
		if err != nil {
			return err
		}
		createdAt := e.CreatedAt
		if createdAt.IsZero() {
			createdAt = time.Now()
		}
		if i > 0 {
			sb.WriteString(", ")
		}
		base := i * 8
		fmt.Fprintf(&sb, "($%d, $%d, $%d, $%d, $%d, $%d::jsonb, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8)
		args = append(args, e.GroupID, e.AccountID, e.Action, e.Source, e.Reason, string(detail), e.IsBad, createdAt)
	}
	_, err := r.db(ctx).ExecContext(ctx, sb.String(), args...)
	return err
}

func (r *schedulingCenterRepository) ListEvents(ctx context.Context, groupID int64, keyword string, limit, offset int) ([]service.SchedulingCenterEvent, int64, error) {
	where := `e.group_id = $1`
	args := []any{groupID}
	if keyword != "" {
		escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(keyword)
		args = append(args, "%"+escaped+"%", keyword)
		where += ` AND (a.name ILIKE $2 OR a.id::text = $3)`
	}

	var total int64
	countRows, err := r.db(ctx).QueryContext(ctx, `
		SELECT COUNT(*) FROM scheduling_center_pool_events e
		JOIN accounts a ON a.id = e.account_id
		WHERE `+where, args...)
	if err != nil {
		return nil, 0, err
	}
	if countRows.Next() {
		if err := countRows.Scan(&total); err != nil {
			_ = countRows.Close()
			return nil, 0, err
		}
	}
	_ = countRows.Close()

	args = append(args, limit, offset)
	rows, err := r.db(ctx).QueryContext(ctx, fmt.Sprintf(`
		SELECT e.id, e.group_id, e.account_id, a.name, e.action, e.source, e.reason, e.detail, e.is_bad, e.created_at
		FROM scheduling_center_pool_events e
		JOIN accounts a ON a.id = e.account_id
		WHERE %s
		ORDER BY e.created_at DESC, e.id DESC
		LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.SchedulingCenterEvent, 0)
	for rows.Next() {
		var (
			e      service.SchedulingCenterEvent
			detail []byte
		)
		if err := rows.Scan(&e.ID, &e.GroupID, &e.AccountID, &e.AccountName, &e.Action, &e.Source, &e.Reason, &detail, &e.IsBad, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		e.Detail = map[string]any{}
		if len(detail) > 0 {
			_ = json.Unmarshal(detail, &e.Detail)
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

// PruneEvents 删除分组中早于 before 的调度日志（走 group_id, created_at 索引）。
func (r *schedulingCenterRepository) PruneEvents(ctx context.Context, groupID int64, before time.Time) (int64, error) {
	res, err := r.db(ctx).ExecContext(ctx,
		`DELETE FROM scheduling_center_pool_events WHERE group_id = $1 AND created_at < $2`, groupID, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *schedulingCenterRepository) CountBadEvents(ctx context.Context, groupID int64, accountIDs []int64, since time.Time) (map[int64]int, error) {
	result := make(map[int64]int, len(accountIDs))
	if len(accountIDs) == 0 {
		return result, nil
	}
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT account_id, COUNT(*)
		FROM scheduling_center_pool_events
		WHERE group_id = $1 AND account_id = ANY($2) AND is_bad AND created_at >= $3
		GROUP BY account_id`, groupID, pq.Array(accountIDs), since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			id    int64
			count int
		)
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		result[id] = count
	}
	return result, rows.Err()
}

// ---------------------------------------------------------------------------
// 真实请求 / 检测样本
// ---------------------------------------------------------------------------

func splitSinceMap(since map[int64]time.Time) ([]int64, []time.Time) {
	ids := make([]int64, 0, len(since))
	times := make([]time.Time, 0, len(since))
	for id, t := range since {
		ids = append(ids, id)
		times = append(times, t)
	}
	return ids, times
}

func (r *schedulingCenterRepository) GetRecentRequests(ctx context.Context, since map[int64]time.Time, limit int) (map[int64][]service.SchedulingCenterRequestSample, error) {
	result := make(map[int64][]service.SchedulingCenterRequestSample, len(since))
	if len(since) == 0 {
		return result, nil
	}
	ids, times := splitSinceMap(since)
	// 每个账号分别取 usage_logs（成功）与 ops_error_logs（上游原因的失败）各最近 limit 条，
	// 都走 (account_id, created_at) 索引，再在外层合并排序。
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT t.account_id, x.at, x.ok, x.stream, x.ttft
		FROM unnest($1::bigint[], $2::timestamptz[]) AS t(account_id, since)
		CROSS JOIN LATERAL (
			(SELECT ul.created_at AS at, true AS ok, ul.stream AS stream, ul.first_token_ms::bigint AS ttft
			 FROM usage_logs ul
			 WHERE ul.account_id = t.account_id AND ul.created_at >= t.since
			 ORDER BY ul.created_at DESC
			 LIMIT $3)
			UNION ALL
			(SELECT e.created_at, false, COALESCE(e.stream, false), NULL::bigint
			 FROM ops_error_logs e
			 WHERE e.account_id = t.account_id AND e.created_at >= t.since
			   AND e.error_owner = 'provider' AND NOT e.is_business_limited
			 ORDER BY e.created_at DESC
			 LIMIT $3)
		) x
		ORDER BY t.account_id, x.at DESC`, pq.Array(ids), pq.Array(times), limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			accountID int64
			sample    service.SchedulingCenterRequestSample
			ttft      sql.NullInt64
		)
		if err := rows.Scan(&accountID, &sample.At, &sample.OK, &sample.Stream, &ttft); err != nil {
			return nil, err
		}
		if len(result[accountID]) >= limit {
			continue
		}
		if ttft.Valid {
			v := ttft.Int64
			sample.FirstTokenMs = &v
		}
		result[accountID] = append(result[accountID], sample)
	}
	return result, rows.Err()
}

func (r *schedulingCenterRepository) GetRecentIQStatuses(ctx context.Context, since map[int64]time.Time, limit int) (map[int64][]string, error) {
	result := make(map[int64][]string, len(since))
	if len(since) == 0 {
		return result, nil
	}
	ids, times := splitSinceMap(since)
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT t.account_id, x.iq_status
		FROM unnest($1::bigint[], $2::timestamptz[]) AS t(account_id, since)
		CROSS JOIN LATERAL (
			SELECT r.iq_status, r.created_at
			FROM scheduled_test_results r
			JOIN scheduled_test_plans p ON p.id = r.plan_id
			WHERE p.account_id = t.account_id AND r.iq_status <> '' AND r.created_at >= t.since
			ORDER BY r.created_at DESC
			LIMIT $3
		) x
		ORDER BY t.account_id, x.created_at DESC`, pq.Array(ids), pq.Array(times), limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			accountID int64
			status    string
		)
		if err := rows.Scan(&accountID, &status); err != nil {
			return nil, err
		}
		result[accountID] = append(result[accountID], status)
	}
	return result, rows.Err()
}

func schedulingCenterCooldownArray(minutes []int) pq.Int64Array {
	out := make(pq.Int64Array, 0, len(minutes))
	for _, m := range minutes {
		out = append(out, int64(m))
	}
	return out
}

func (r *schedulingCenterRepository) GetNextPlanRunAt(ctx context.Context, groupID int64) (*time.Time, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT MIN(p.next_run_at)
		FROM scheduling_center_account_plans m
		JOIN scheduled_test_plans p ON p.id = m.plan_id
		WHERE m.group_id = $1 AND p.enabled`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var next sql.NullTime
	if rows.Next() {
		if err := rows.Scan(&next); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil || !next.Valid {
		return nil, err
	}
	v := next.Time
	return &v, nil
}

func (r *schedulingCenterRepository) ResetGroupData(ctx context.Context, groupID int64, opts service.SchedulingCenterResetOptions) (*service.SchedulingCenterResetResult, error) {
	result := &service.SchedulingCenterResetResult{}
	exec := func(query string, target *int64) error {
		res, err := r.db(ctx).ExecContext(ctx, query, groupID)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		*target = n
		return nil
	}
	if opts.ClearPool {
		if err := exec(`DELETE FROM scheduling_center_pool_members WHERE group_id = $1`, &result.PoolMembers); err != nil {
			return nil, err
		}
	}
	if opts.ClearStates {
		if err := exec(`DELETE FROM scheduling_center_account_states WHERE group_id = $1`, &result.States); err != nil {
			return nil, err
		}
	}
	if opts.ClearEvents {
		if err := exec(`DELETE FROM scheduling_center_pool_events WHERE group_id = $1`, &result.Events); err != nil {
			return nil, err
		}
	}
	if opts.ClearRecords {
		if err := exec(`
			DELETE FROM scheduled_test_results
			WHERE plan_id IN (SELECT plan_id FROM scheduling_center_account_plans WHERE group_id = $1)`, &result.Records); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (r *schedulingCenterRepository) ListForeignPlanOwners(ctx context.Context, groupID int64) ([]service.SchedulingCenterPlanOwner, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT m.account_id, m.group_id, COALESCE(g.name, ''), p.cron_expression, p.model_id, p.max_results
		FROM scheduling_center_account_plans m
		JOIN account_groups ag ON ag.account_id = m.account_id AND ag.group_id = $1
		JOIN scheduled_test_plans p ON p.id = m.plan_id
		LEFT JOIN groups g ON g.id = m.group_id
		WHERE m.group_id <> $1
		ORDER BY m.account_id`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.SchedulingCenterPlanOwner, 0)
	for rows.Next() {
		var o service.SchedulingCenterPlanOwner
		if err := rows.Scan(&o.AccountID, &o.GroupID, &o.GroupName, &o.CronExpression, &o.ModelID, &o.MaxResults); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ListEnforcedPools 返回开启「调度池生效」的分组及其网关可用账号：调度池成员 + 探测已通过、正在观察真实请求的恢复中账号
// + 固定在调度池的 OAuth / Setup Token 账号（不写入调度池表，这里按分组成员关系补上）。
// 只返回仍属于该分组的账号；开启但池为空的分组也返回（空集合），由网关回退到原逻辑。
func (r *schedulingCenterRepository) ListEnforcedPools(ctx context.Context) (map[int64][]int64, error) {
	rows, err := r.db(ctx).QueryContext(ctx, `
		SELECT s.group_id, x.account_id
		FROM scheduling_center_group_settings s
		LEFT JOIN LATERAL (
			SELECT m.account_id FROM scheduling_center_pool_members m WHERE m.group_id = s.group_id
			UNION
			SELECT st.account_id FROM scheduling_center_account_states st
			WHERE st.group_id = s.group_id AND st.state = 'probing' AND st.probe_status = 'passed'
			UNION
			SELECT ag.account_id FROM account_groups ag
			JOIN accounts a ON a.id = ag.account_id AND a.deleted_at IS NULL
			WHERE ag.group_id = s.group_id AND a.type IN `+schedulingCenterFixedTypesSQL+`
		) x ON TRUE
		WHERE s.enabled AND s.enforce_enabled`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make(map[int64][]int64)
	for rows.Next() {
		var groupID int64
		var accountID sql.NullInt64
		if err := rows.Scan(&groupID, &accountID); err != nil {
			return nil, err
		}
		if _, ok := out[groupID]; !ok {
			out[groupID] = []int64{}
		}
		if accountID.Valid {
			out[groupID] = append(out[groupID], accountID.Int64)
		}
	}
	return out, rows.Err()
}
