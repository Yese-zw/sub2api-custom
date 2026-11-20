package service

import (
	"context"
	"errors"
	"math"
	"sort"
	"time"
)

// 调度中心对比与导出。
//
// 对比口径：按「请求发生时账号是否在调度池」把真实请求、上游错误、智商检测归到
// 调度池 / 账号池两侧（10 分钟粒度，取桶的中点判断归属）。调度中心不影响真实调度，
// 两侧流量都由现有系统分配，因此两侧指标可以直接反映调度中心挑出的账号是否更好。

const (
	SchedulingCenterExportSchema       = "sub2api.scheduling_center.export/v1"
	schedulingCenterReportBucket       = 10 * time.Minute
	schedulingCenterReportMaxHours     = 168
	schedulingCenterReportDefaultHours = 24
	schedulingCenterExportMaxEvents    = 50000
	schedulingCenterExportMaxIQRecords = 50000
)

var ErrSchedulingCenterInvalidRange = errors.New("hours must be between 1 and 168")

// ---------------------------------------------------------------------------
// 仓储返回的原始桶
// ---------------------------------------------------------------------------

type SchedulingCenterUsageBucket struct {
	AccountID           int64
	Bucket              time.Time
	Requests            int64
	InputTokens         int64
	CacheCreationTokens int64
	CacheReadTokens     int64
	FirstTokenSamples   int64
	FirstTokenSumMs     int64
	SlowRequests        int64
}

type SchedulingCenterErrorBucket struct {
	AccountID int64
	Bucket    time.Time
	Errors    int64
}

type SchedulingCenterIQBucket struct {
	AccountID         int64
	Bucket            time.Time
	Checks            int64
	Passed            int64
	Incorrect         int64
	Errored           int64
	FirstTokenSamples int64
	FirstTokenSumMs   int64
}

// SchedulingCenterMembershipEvent 是一次入池 / 出池（用于还原池归属时间线）。
type SchedulingCenterMembershipEvent struct {
	AccountID int64
	Action    string // enter | exit
	At        time.Time
}

type SchedulingCenterGroupBrief struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Status   string `json:"status"`
}

type SchedulingCenterAccountBrief struct {
	ID                     int64      `json:"id"`
	Name                   string     `json:"name"`
	Platform               string     `json:"platform"`
	Type                   string     `json:"type"`
	Status                 string     `json:"status"`
	Schedulable            bool       `json:"schedulable"`
	RateMultiplier         *float64   `json:"rate_multiplier"`
	RateLimitResetAt       *time.Time `json:"rate_limit_reset_at"`
	OverloadUntil          *time.Time `json:"overload_until"`
	TempUnschedulableUntil *time.Time `json:"temp_unschedulable_until"`
	ErrorMessage           string     `json:"error_message,omitempty"`
}

type SchedulingCenterExportIQRecord struct {
	AccountID    int64     `json:"account_id"`
	StartedAt    time.Time `json:"started_at"`
	Status       string    `json:"status"`
	IQStatus     string    `json:"iq_status"`
	FirstTokenMs *int64    `json:"first_token_ms"`
	LatencyMs    int64     `json:"latency_ms"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// SchedulingCenterReportRepository 是对比与导出需要的查询。
type SchedulingCenterReportRepository interface {
	GetUsageBuckets(ctx context.Context, accountIDs []int64, from, to time.Time, bucket time.Duration, slowMs int64) ([]SchedulingCenterUsageBucket, error)
	GetErrorBuckets(ctx context.Context, accountIDs []int64, from, to time.Time, bucket time.Duration) ([]SchedulingCenterErrorBucket, error)
	GetIQBuckets(ctx context.Context, accountIDs []int64, from, to time.Time, bucket time.Duration) ([]SchedulingCenterIQBucket, error)
	// ListMembershipEvents 返回每个账号在 from 之前最后一次入池/出池动作，以及 [from, to) 内的入池/出池（旧 → 新）。
	ListMembershipEvents(ctx context.Context, groupID int64, from, to time.Time) (map[int64]string, []SchedulingCenterMembershipEvent, error)
	ListEventsInRange(ctx context.Context, groupID int64, from, to time.Time, limit int) ([]SchedulingCenterEvent, error)
	ListIQRecordsInRange(ctx context.Context, accountIDs []int64, from, to time.Time, limit int) ([]SchedulingCenterExportIQRecord, error)
	GetGroupBrief(ctx context.Context, groupID int64) (*SchedulingCenterGroupBrief, error)
	ListGroupAccountsBrief(ctx context.Context, groupID int64) ([]SchedulingCenterAccountBrief, error)
}

// ---------------------------------------------------------------------------
// 指标
// ---------------------------------------------------------------------------

// SchedulingCenterSideMetrics 是一侧（调度池 / 账号池）的汇总指标。
type SchedulingCenterSideMetrics struct {
	Accounts            int      `json:"accounts"`
	Requests            int64    `json:"requests"`
	Errors              int64    `json:"errors"`
	ErrorRate           *float64 `json:"error_rate"`
	InputTokens         int64    `json:"input_tokens"`
	CacheCreationTokens int64    `json:"cache_creation_tokens"`
	CacheReadTokens     int64    `json:"cache_read_tokens"`
	CacheRate           *float64 `json:"cache_rate"`
	FirstTokenSamples   int64    `json:"first_token_samples"`
	AvgFirstTokenMs     *float64 `json:"avg_first_token_ms"`
	SlowRequests        int64    `json:"slow_requests"`
	SlowRate            *float64 `json:"slow_rate"`
	IQChecks            int64    `json:"iq_checks"`
	IQPassed            int64    `json:"iq_passed"`
	IQIncorrect         int64    `json:"iq_incorrect"`
	IQErrored           int64    `json:"iq_errored"`
	IQPassRate          *float64 `json:"iq_pass_rate"`
	IQAvgFirstTokenMs   *float64 `json:"iq_avg_first_token_ms"`

	firstTokenSumMs   int64
	iqFirstTokenN     int64
	iqFirstTokenSumMs int64
	accountSet        map[int64]struct{}
}

func (m *SchedulingCenterSideMetrics) markAccount(id int64) {
	if m.accountSet == nil {
		m.accountSet = map[int64]struct{}{}
	}
	m.accountSet[id] = struct{}{}
}

func (m *SchedulingCenterSideMetrics) addUsage(b SchedulingCenterUsageBucket) {
	m.Requests += b.Requests
	m.InputTokens += b.InputTokens
	m.CacheCreationTokens += b.CacheCreationTokens
	m.CacheReadTokens += b.CacheReadTokens
	m.FirstTokenSamples += b.FirstTokenSamples
	m.firstTokenSumMs += b.FirstTokenSumMs
	m.SlowRequests += b.SlowRequests
	m.markAccount(b.AccountID)
}

func (m *SchedulingCenterSideMetrics) addErrors(b SchedulingCenterErrorBucket) {
	m.Errors += b.Errors
	m.markAccount(b.AccountID)
}

func (m *SchedulingCenterSideMetrics) addIQ(b SchedulingCenterIQBucket) {
	m.IQChecks += b.Checks
	m.IQPassed += b.Passed
	m.IQIncorrect += b.Incorrect
	m.IQErrored += b.Errored
	m.iqFirstTokenN += b.FirstTokenSamples
	m.iqFirstTokenSumMs += b.FirstTokenSumMs
}

func schedulingCenterRatio(num, den int64, scale float64) *float64 {
	if den <= 0 {
		return nil
	}
	v := math.Round(float64(num)/float64(den)*scale*100) / 100
	return &v
}

func (m *SchedulingCenterSideMetrics) finalize() {
	m.Accounts = len(m.accountSet)
	m.ErrorRate = schedulingCenterRatio(m.Errors, m.Requests+m.Errors, 100)
	m.CacheRate = schedulingCenterRatio(m.CacheReadTokens, m.InputTokens+m.CacheCreationTokens+m.CacheReadTokens, 100)
	m.AvgFirstTokenMs = schedulingCenterRatio(m.firstTokenSumMs, m.FirstTokenSamples, 1)
	m.SlowRate = schedulingCenterRatio(m.SlowRequests, m.FirstTokenSamples, 100)
	m.IQPassRate = schedulingCenterRatio(m.IQPassed, m.IQPassed+m.IQIncorrect, 100)
	m.IQAvgFirstTokenMs = schedulingCenterRatio(m.iqFirstTokenSumMs, m.iqFirstTokenN, 1)
}

type SchedulingCenterSeriesPoint struct {
	Time        time.Time                   `json:"time"`
	Pool        SchedulingCenterSideMetrics `json:"pool"`
	AccountPool SchedulingCenterSideMetrics `json:"account_pool"`
}

type SchedulingCenterSnapshot struct {
	GroupAccounts   int            `json:"group_accounts"`
	PoolAccounts    int            `json:"pool_accounts"`
	AccountAccounts int            `json:"account_pool_accounts"`
	States          map[string]int `json:"states"`
}

type SchedulingCenterComparison struct {
	GroupID         int64                         `json:"group_id"`
	From            time.Time                     `json:"from"`
	To              time.Time                     `json:"to"`
	Hours           int                           `json:"hours"`
	BucketMinutes   int                           `json:"bucket_minutes"`
	SlowThresholdMs int64                         `json:"slow_threshold_ms"`
	Pool            SchedulingCenterSideMetrics   `json:"pool"`
	AccountPool     SchedulingCenterSideMetrics   `json:"account_pool"`
	Series          []SchedulingCenterSeriesPoint `json:"series"`
	Snapshot        SchedulingCenterSnapshot      `json:"snapshot"`
}

// ---------------------------------------------------------------------------
// 池归属时间线
// ---------------------------------------------------------------------------

type schedulingCenterInterval struct{ start, end time.Time }

type schedulingCenterTimeline map[int64][]schedulingCenterInterval

// buildSchedulingCenterTimeline 还原 [from, to) 内每个账号在调度池中的时间段。
// prior 为 from 之前最后一次动作；没有任何日志的当前成员（日志功能上线前加入的）按入池时间计。
func buildSchedulingCenterTimeline(from, to time.Time, prior map[int64]string, events []SchedulingCenterMembershipEvent, members []SchedulingCenterPoolMember) schedulingCenterTimeline {
	byAccount := map[int64][]SchedulingCenterMembershipEvent{}
	for _, e := range events {
		byAccount[e.AccountID] = append(byAccount[e.AccountID], e)
	}
	memberAt := map[int64]time.Time{}
	for _, m := range members {
		memberAt[m.AccountID] = m.EnteredAt
	}

	ids := map[int64]struct{}{}
	for id := range prior {
		ids[id] = struct{}{}
	}
	for id := range byAccount {
		ids[id] = struct{}{}
	}
	for id := range memberAt {
		ids[id] = struct{}{}
	}

	out := schedulingCenterTimeline{}
	for id := range ids {
		evs := byAccount[id]
		var (
			in    bool
			start time.Time
		)
		switch {
		case prior[id] == SchedulingCenterActionEnter:
			in, start = true, from
		case prior[id] == SchedulingCenterActionExit:
			in = false
		case len(evs) > 0:
			// 窗口前没有日志：第一条是出池说明此前在池中
			in = evs[0].Action == SchedulingCenterActionExit
			start = from
		default:
			if at, ok := memberAt[id]; ok && at.Before(to) {
				in = true
				start = at
				if start.Before(from) {
					start = from
				}
			}
		}
		for _, e := range evs {
			switch e.Action {
			case SchedulingCenterActionEnter:
				if !in {
					in, start = true, e.At
				}
			case SchedulingCenterActionExit:
				if in {
					if e.At.After(start) {
						out[id] = append(out[id], schedulingCenterInterval{start, e.At})
					}
					in = false
				}
			}
		}
		if in && to.After(start) {
			out[id] = append(out[id], schedulingCenterInterval{start, to})
		}
	}
	return out
}

func (tl schedulingCenterTimeline) inPool(accountID int64, at time.Time) bool {
	for _, iv := range tl[accountID] {
		if !at.Before(iv.start) && at.Before(iv.end) {
			return true
		}
	}
	return false
}

func (tl schedulingCenterTimeline) minutesInPool(accountID int64) float64 {
	total := time.Duration(0)
	for _, iv := range tl[accountID] {
		total += iv.end.Sub(iv.start)
	}
	return math.Round(total.Minutes()*10) / 10
}

// ---------------------------------------------------------------------------
// 对比
// ---------------------------------------------------------------------------

type schedulingCenterReportData struct {
	from, to    time.Time
	settings    *SchedulingCenterGroupSettings
	accountIDs  []int64
	members     []SchedulingCenterPoolMember
	states      []SchedulingCenterAccountState
	timeline    schedulingCenterTimeline
	usage       []SchedulingCenterUsageBucket
	errs        []SchedulingCenterErrorBucket
	iq          []SchedulingCenterIQBucket
	slowMs      int64
	bucket      time.Duration
	reportRepo  SchedulingCenterReportRepository
	accountsMap map[int64]struct{}
}

func (s *SchedulingCenterService) reportRepo() (SchedulingCenterReportRepository, error) {
	r, ok := s.repo.(SchedulingCenterReportRepository)
	if !ok {
		return nil, errors.New("scheduling center report is not supported by repository")
	}
	return r, nil
}

func normalizeSchedulingCenterHours(hours int) (int, error) {
	if hours == 0 {
		return schedulingCenterReportDefaultHours, nil
	}
	if hours < 0 || hours > schedulingCenterReportMaxHours {
		return 0, ErrSchedulingCenterInvalidRange
	}
	return hours, nil
}

func (s *SchedulingCenterService) loadReportData(ctx context.Context, groupID int64, hours int) (*schedulingCenterReportData, error) {
	rr, err := s.reportRepo()
	if err != nil {
		return nil, err
	}
	to := s.now().Truncate(time.Minute)
	from := to.Add(-time.Duration(hours) * time.Hour)

	settings, err := s.GetGroupSettings(ctx, groupID)
	if err != nil {
		return nil, err
	}
	accountIDs, err := s.repo.ListGroupAccountIDs(ctx, groupID)
	if err != nil {
		return nil, err
	}
	members, err := s.repo.ListPoolMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}
	states, err := s.repo.ListAccountStates(ctx, groupID)
	if err != nil {
		return nil, err
	}
	prior, events, err := rr.ListMembershipEvents(ctx, groupID, from, to)
	if err != nil {
		return nil, err
	}

	slowMs := int64(10000)
	if settings.ExitLatencyMs != nil {
		slowMs = *settings.ExitLatencyMs
	}
	d := &schedulingCenterReportData{
		from: from, to: to, settings: settings, accountIDs: accountIDs, members: members, states: states,
		timeline: buildSchedulingCenterTimeline(from, to, prior, events, members),
		slowMs:   slowMs, bucket: schedulingCenterReportBucket, reportRepo: rr,
		accountsMap: map[int64]struct{}{},
	}
	for _, id := range accountIDs {
		d.accountsMap[id] = struct{}{}
	}
	if len(accountIDs) == 0 {
		return d, nil
	}
	if d.usage, err = rr.GetUsageBuckets(ctx, accountIDs, from, to, d.bucket, slowMs); err != nil {
		return nil, err
	}
	if d.errs, err = rr.GetErrorBuckets(ctx, accountIDs, from, to, d.bucket); err != nil {
		return nil, err
	}
	if d.iq, err = rr.GetIQBuckets(ctx, accountIDs, from, to, d.bucket); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *schedulingCenterReportData) inPool(accountID int64, bucket time.Time) bool {
	return d.timeline.inPool(accountID, bucket.Add(d.bucket/2))
}

// buildComparison 把桶按池归属汇总为两侧总量与按小时的趋势。
func (d *schedulingCenterReportData) buildComparison(groupID int64, hours int) *SchedulingCenterComparison {
	c := &SchedulingCenterComparison{
		GroupID: groupID, From: d.from, To: d.to, Hours: hours,
		BucketMinutes: int(d.bucket / time.Minute), SlowThresholdMs: d.slowMs,
	}
	series := map[time.Time]*SchedulingCenterSeriesPoint{}
	point := func(t time.Time) *SchedulingCenterSeriesPoint {
		h := t.Truncate(time.Hour)
		p, ok := series[h]
		if !ok {
			p = &SchedulingCenterSeriesPoint{Time: h}
			series[h] = p
		}
		return p
	}
	side := func(pool bool, total *SchedulingCenterComparison, p *SchedulingCenterSeriesPoint) (*SchedulingCenterSideMetrics, *SchedulingCenterSideMetrics) {
		if pool {
			return &total.Pool, &p.Pool
		}
		return &total.AccountPool, &p.AccountPool
	}
	for _, b := range d.usage {
		t, s := side(d.inPool(b.AccountID, b.Bucket), c, point(b.Bucket))
		t.addUsage(b)
		s.addUsage(b)
	}
	for _, b := range d.errs {
		t, s := side(d.inPool(b.AccountID, b.Bucket), c, point(b.Bucket))
		t.addErrors(b)
		s.addErrors(b)
	}
	for _, b := range d.iq {
		t, s := side(d.inPool(b.AccountID, b.Bucket), c, point(b.Bucket))
		t.addIQ(b)
		s.addIQ(b)
	}

	// 补齐没有数据的小时，保证趋势连续
	for h := d.from.Truncate(time.Hour); h.Before(d.to); h = h.Add(time.Hour) {
		point(h)
	}
	c.Series = make([]SchedulingCenterSeriesPoint, 0, len(series))
	for _, p := range series {
		p.Pool.finalize()
		p.AccountPool.finalize()
		c.Series = append(c.Series, *p)
	}
	sort.Slice(c.Series, func(i, j int) bool { return c.Series[i].Time.Before(c.Series[j].Time) })
	c.Pool.finalize()
	c.AccountPool.finalize()

	memberSet := map[int64]struct{}{}
	for _, m := range d.members {
		if _, ok := d.accountsMap[m.AccountID]; ok {
			memberSet[m.AccountID] = struct{}{}
		}
	}
	c.Snapshot = SchedulingCenterSnapshot{
		GroupAccounts:   len(d.accountIDs),
		PoolAccounts:    len(memberSet),
		AccountAccounts: len(d.accountIDs) - len(memberSet),
		States:          map[string]int{},
	}
	for _, st := range d.states {
		if _, ok := d.accountsMap[st.AccountID]; ok {
			c.Snapshot.States[st.State]++
		}
	}
	return c
}

// GetComparison 返回分组最近 hours 小时调度池与账号池的真实数据对比。
func (s *SchedulingCenterService) GetComparison(ctx context.Context, groupID int64, hours int) (*SchedulingCenterComparison, error) {
	hours, err := normalizeSchedulingCenterHours(hours)
	if err != nil {
		return nil, err
	}
	d, err := s.loadReportData(ctx, groupID, hours)
	if err != nil {
		return nil, err
	}
	return d.buildComparison(groupID, hours), nil
}

// ---------------------------------------------------------------------------
// 导出
// ---------------------------------------------------------------------------

type SchedulingCenterExportAccount struct {
	SchedulingCenterAccountBrief
	CenterStatus     string                      `json:"center_status"` // in_pool | idle | cooling | probing | ready | manual_out
	PoolSource       string                      `json:"pool_source,omitempty"`
	MinutesInPool    float64                     `json:"minutes_in_pool"`
	InPool           SchedulingCenterSideMetrics `json:"in_pool"`
	OutOfPool        SchedulingCenterSideMetrics `json:"out_of_pool"`
	Events           int                         `json:"events"`
	BadEvents        int                         `json:"bad_events"`
	ExitReasons      map[string]int              `json:"exit_reasons"`
	EnteredPoolAt    *time.Time                  `json:"entered_pool_at,omitempty"`
	CooldownUntil    *time.Time                  `json:"cooldown_until,omitempty"`
	CenterReasonCode string                      `json:"center_reason_code,omitempty"`
}

type SchedulingCenterExport struct {
	Schema      string                           `json:"schema"`
	GeneratedAt time.Time                        `json:"generated_at"`
	Group       *SchedulingCenterGroupBrief      `json:"group"`
	Window      map[string]any                   `json:"window"`
	Notes       []string                         `json:"notes"`
	Settings    *SchedulingCenterGroupSettings   `json:"settings"`
	Comparison  *SchedulingCenterComparison      `json:"comparison"`
	Accounts    []SchedulingCenterExportAccount  `json:"accounts"`
	Events      []SchedulingCenterEvent          `json:"events"`
	IQRecords   []SchedulingCenterExportIQRecord `json:"iq_records"`
	Truncated   map[string]bool                  `json:"truncated"`
}

// Export 生成分组最近 hours 小时的完整调度中心数据，用于离线评估是否可以替换现有调度。
// 不含任何凭据或上游地址。
func (s *SchedulingCenterService) Export(ctx context.Context, groupID int64, hours int) (*SchedulingCenterExport, error) {
	hours, err := normalizeSchedulingCenterHours(hours)
	if err != nil {
		return nil, err
	}
	d, err := s.loadReportData(ctx, groupID, hours)
	if err != nil {
		return nil, err
	}
	group, err := d.reportRepo.GetGroupBrief(ctx, groupID)
	if err != nil {
		return nil, err
	}
	briefs, err := d.reportRepo.ListGroupAccountsBrief(ctx, groupID)
	if err != nil {
		return nil, err
	}
	events, err := d.reportRepo.ListEventsInRange(ctx, groupID, d.from, d.to, schedulingCenterExportMaxEvents+1)
	if err != nil {
		return nil, err
	}
	iqRecords, err := d.reportRepo.ListIQRecordsInRange(ctx, d.accountIDs, d.from, d.to, schedulingCenterExportMaxIQRecords+1)
	if err != nil {
		return nil, err
	}
	truncated := map[string]bool{
		"events":     len(events) > schedulingCenterExportMaxEvents,
		"iq_records": len(iqRecords) > schedulingCenterExportMaxIQRecords,
	}
	if truncated["events"] {
		events = events[:schedulingCenterExportMaxEvents]
	}
	if truncated["iq_records"] {
		iqRecords = iqRecords[:schedulingCenterExportMaxIQRecords]
	}
	if events == nil {
		events = []SchedulingCenterEvent{}
	}
	if iqRecords == nil {
		iqRecords = []SchedulingCenterExportIQRecord{}
	}

	// 账号级汇总
	accounts := make(map[int64]*SchedulingCenterExportAccount, len(briefs))
	order := make([]int64, 0, len(briefs))
	for _, b := range briefs {
		accounts[b.ID] = &SchedulingCenterExportAccount{
			SchedulingCenterAccountBrief: b,
			CenterStatus:                 "idle",
			MinutesInPool:                d.timeline.minutesInPool(b.ID),
			ExitReasons:                  map[string]int{},
		}
		order = append(order, b.ID)
	}
	for _, m := range d.members {
		if a, ok := accounts[m.AccountID]; ok {
			entered := m.EnteredAt
			a.CenterStatus, a.PoolSource, a.EnteredPoolAt = "in_pool", m.Source, &entered
		}
	}
	for _, st := range d.states {
		if a, ok := accounts[st.AccountID]; ok && a.CenterStatus != "in_pool" {
			a.CenterStatus, a.CooldownUntil, a.CenterReasonCode = st.State, st.CooldownUntil, st.ReasonCode
		}
	}
	pick := func(id int64, bucket time.Time) *SchedulingCenterSideMetrics {
		a, ok := accounts[id]
		if !ok {
			return nil
		}
		if d.inPool(id, bucket) {
			return &a.InPool
		}
		return &a.OutOfPool
	}
	for _, b := range d.usage {
		if m := pick(b.AccountID, b.Bucket); m != nil {
			m.addUsage(b)
		}
	}
	for _, b := range d.errs {
		if m := pick(b.AccountID, b.Bucket); m != nil {
			m.addErrors(b)
		}
	}
	for _, b := range d.iq {
		if m := pick(b.AccountID, b.Bucket); m != nil {
			m.addIQ(b)
		}
	}
	for _, e := range events {
		a, ok := accounts[e.AccountID]
		if !ok {
			continue
		}
		a.Events++
		if e.IsBad {
			a.BadEvents++
		}
		if e.Action == SchedulingCenterActionExit {
			a.ExitReasons[e.Reason]++
		}
	}
	out := make([]SchedulingCenterExportAccount, 0, len(order))
	for _, id := range order {
		a := accounts[id]
		a.InPool.finalize()
		a.OutOfPool.finalize()
		out = append(out, *a)
	}

	return &SchedulingCenterExport{
		Schema:      SchedulingCenterExportSchema,
		GeneratedAt: s.now(),
		Group:       group,
		Window: map[string]any{
			"from": d.from, "to": d.to, "hours": hours,
			"bucket_minutes": int(d.bucket / time.Minute),
		},
		Notes: []string{
			"Scheduling center is shadow-only: real traffic is routed by the existing scheduler, not by these pools.",
			"Real requests, upstream errors and IQ checks are attributed to the pool the account was in at that time (10-minute buckets, bucket midpoint).",
			"Upstream errors come from ops_error_logs (error_owner=provider, excluding business-limited); they are empty when ops monitoring is disabled.",
			"Rates are percentages (0-100); latencies are milliseconds; slow = streaming first token above slow_threshold_ms.",
			"Account status fields (status, schedulable, rate_limit_reset_at, overload_until, temp_unschedulable_until) are the existing scheduler's current state at export time.",
		},
		Settings:   d.settings,
		Comparison: d.buildComparison(groupID, hours),
		Accounts:   out,
		Events:     events,
		IQRecords:  iqRecords,
		Truncated:  truncated,
	}, nil
}
