package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// 调度中心：独立于网关调度的管理视图。
//
//   - 分组下的账号分为「账号池」（默认）和「调度池」，只做归类与展示，
//     不影响 GatewayService / OpenAIAccountScheduler 的选号行为。
//   - 分组开启调度后，为分组内每个账号维护一个智商检测定时测试（保留 100 条记录），
//     该计划不会暂停账号，也不会自动恢复账号。
//   - 分组可设置入池规则（缓存率 / 降智 / 首 Token），满足规则的账号自动进入调度池。
//   - 分组可设置出池规则（24h 缓存率 / 检测降智次数 / 真实请求连续慢 / 真实请求连续错误），
//     触发后移出调度池并按 3 小时内的「坏」次数退避冷却（5 / 15 / 30 / 60 分钟封顶），
//     冷却结束后需主动探测通过、真实请求连续正常才能重新入池。详见 scheduling_center_lifecycle.go。

const (
	SchedulingCenterPoolScheduling = "scheduling"
	SchedulingCenterPoolAccount    = "account"

	SchedulingCenterSourceManual = "manual"
	SchedulingCenterSourceRule   = "rule"
	// SchedulingCenterSourceFixed 是 OAuth / Setup Token 账号的来源：调度中心只处理 Key 类型账号，
	// 这些账号固定在调度池中（不落库、不检测、不参与出入池规则，也不能手动移出）。
	SchedulingCenterSourceFixed = "fixed"

	SchedulingCenterFirstTokenSourceTest    = "test"
	SchedulingCenterFirstTokenSourceTraffic = "traffic"

	SchedulingCenterDefaultCron = "*/30 * * * *"
	// SchedulingCenterPlanMaxResults 是检测记录保留条数的默认值；RecordRetention 可在 [Min, Max] 内配置。
	SchedulingCenterPlanMaxResults     = 100
	SchedulingCenterMinRecordRetention = 10
	SchedulingCenterMaxRecordRetention = 1000

	// 调度日志保留天数：下限 8 天，保证 7 天的对比 / 导出窗口和「坏」次数窗口（最长 7 天）都有完整日志。
	SchedulingCenterDefaultEventRetentionDays = 30
	SchedulingCenterMinEventRetentionDays     = 8
	SchedulingCenterMaxEventRetentionDays     = 365
	schedulingCenterMetricsWindow             = 24 * time.Hour
	schedulingCenterMaxBatchAccounts          = 1000
	schedulingCenterRuleInterval              = time.Minute
	schedulingCenterSyncEveryNTicks           = 5
	schedulingCenterWorkerStartDelay          = 30 * time.Second
	schedulingCenterWorkerTickTimeout         = 2 * time.Minute
	schedulingCenterMaxModelIDLen             = 100
)

var (
	ErrSchedulingCenterInvalidPool  = errors.New("invalid pool, expected scheduling or account")
	ErrSchedulingCenterInvalidCron  = errors.New("invalid cron expression")
	ErrSchedulingCenterInvalidRule  = errors.New("invalid rule value")
	ErrSchedulingCenterInvalidModel = errors.New("invalid model id")
)

// SchedulingCenterAccountStats 是从 usage_logs 汇总的账号窗口统计。
type SchedulingCenterAccountStats struct {
	Requests            int64
	InputTokens         int64
	CacheCreationTokens int64
	CacheReadTokens     int64
	FirstTokenSamples   int64
	AvgFirstTokenMs     *float64
}

// SchedulingCenterIQResult 是账号最近一次智商检测（降智检测）结果。
type SchedulingCenterIQResult struct {
	Status    string
	Message   string
	CheckedAt time.Time
}

// SchedulingCenterGroupSettings 是分组的调度中心设置。
type SchedulingCenterGroupSettings struct {
	GroupID         int64    `json:"group_id"`
	Enabled         bool     `json:"enabled"`
	CronExpression  string   `json:"cron_expression"`
	ModelID         string   `json:"model_id"`
	RuleEnabled     bool     `json:"rule_enabled"`
	MinCacheRate    *float64 `json:"min_cache_rate"`
	RequireIQPass   bool     `json:"require_iq_pass"`
	MaxFirstTokenMs *int64   `json:"max_first_token_ms"`

	// 出池规则：满足任意一条即出池；指针为 nil 表示该条件关闭。
	ExitEnabled          bool     `json:"exit_enabled"`
	ExitCacheDropPoints  *float64 `json:"exit_cache_drop_points"`
	ExitCacheMinRequests int      `json:"exit_cache_min_requests"`
	ExitIQWindow         int      `json:"exit_iq_window"`
	ExitIQMaxDegraded    *int     `json:"exit_iq_max_degraded"`
	ExitLatencyMs        *int64   `json:"exit_latency_ms"`
	ExitLatencyCount     int      `json:"exit_latency_count"`  // 最近 ExitLatencyWindow 次流式请求中首 Token 过慢达到该次数即出池
	ExitLatencyWindow    int      `json:"exit_latency_window"` // 首 Token 规则观察的最近流式请求次数
	ExitErrorCount       *int     `json:"exit_error_count"`    // 最近 ExitErrorWindow 次真实请求中上游错误达到该次数即出池，nil 不检查
	ExitErrorWindow      int      `json:"exit_error_window"`   // 错误规则观察的最近真实请求次数
	// RecoverSuccessCount 冷却结束、探测通过后，真实请求需连续正常的次数。
	RecoverSuccessCount int `json:"recover_success_count"`

	// 其余规则参数（均可按分组配置）
	FirstTokenTestSamples  int    `json:"first_token_test_samples"`  // 首 Token 取最近 N 次检测平均
	EntryCacheMinRequests  int    `json:"entry_cache_min_requests"`  // 入池缓存率至少需要的 24h 请求数
	EntryIQPassCount       int    `json:"entry_iq_pass_count"`       // 入池要求最近 N 次检测都通过
	ExitTrafficWindowHours int    `json:"exit_traffic_window_hours"` // 出池真实请求只看最近 N 小时
	BadWindowMinutes       int    `json:"bad_window_minutes"`        // 统计「坏」次数的窗口
	CooldownMinutes        []int  `json:"cooldown_minutes"`          // 第 1、2、3… 次坏的冷却分钟数，最后一个封顶
	RecoverLatencyMs       *int64 `json:"recover_latency_ms"`        // 探测与实测的首 Token 上限，nil 不检查

	// 兜底：调度池账号数低于 FallbackMinAccounts 时，从账号池按 FallbackOrder 挑选账号补入。
	FallbackEnabled       bool     `json:"fallback_enabled"`
	FallbackMinAccounts   int      `json:"fallback_min_accounts"`
	FallbackOrder         []string `json:"fallback_order"`           // rate_multiplier / first_token / cache_rate 的优先顺序
	FallbackRequireIQPass bool     `json:"fallback_require_iq_pass"` // 兜底账号要求最近一次智商检测通过

	// ManualOutHours 手动移出的有效期（小时）：到期后入池规则 / 兜底可再次选中该账号；0 表示永久。
	ManualOutHours int `json:"manual_out_hours"`

	// RecordRetention 每个账号保留的检测记录条数（同步到托管检测计划的 max_results）。
	RecordRetention int `json:"record_retention"`

	// EventRetentionDays 调度日志保留天数，后台定期清理更早的日志。
	EventRetentionDays int `json:"event_retention_days"`

	// EnforceEnabled 调度池生效（灰度开关）：开启后网关只从本分组调度池选账号，池内无可用账号时回退原逻辑。
	// 仅在 Enabled 为 true 时生效；关闭时调度中心保持影子模式，不影响真实调度。
	EnforceEnabled bool `json:"enforce_enabled"`

	UpdatedAt time.Time `json:"updated_at"`
}

// HasExitCondition 报告出池规则是否至少配置了一个可执行的条件。
// 缓存率条件依赖入池缓存率门槛（出池门槛 = 入池门槛 − 下降点数）。
func (s *SchedulingCenterGroupSettings) HasExitCondition() bool {
	if s == nil {
		return false
	}
	return (s.ExitCacheDropPoints != nil && s.MinCacheRate != nil) ||
		s.ExitIQMaxDegraded != nil || s.ExitLatencyMs != nil || s.ExitErrorCount != nil
}

// EntryRuleActive 报告入池规则是否开启且至少有一个条件。
func (s *SchedulingCenterGroupSettings) EntryRuleActive() bool {
	return s != nil && s.RuleEnabled && s.HasRuleCondition()
}

// HasRuleCondition 报告规则是否至少配置了一个条件；没有条件时不自动入池。
func (s *SchedulingCenterGroupSettings) HasRuleCondition() bool {
	return s != nil && (s.MinCacheRate != nil || s.RequireIQPass || s.MaxFirstTokenMs != nil)
}

func defaultSchedulingCenterGroupSettings(groupID int64) *SchedulingCenterGroupSettings {
	drop := 20.0
	iqMax := 3
	latency := int64(10000)
	errCount := 3
	recoverLatency := int64(10000)
	return &SchedulingCenterGroupSettings{
		GroupID:              groupID,
		CronExpression:       SchedulingCenterDefaultCron,
		ExitCacheDropPoints:  &drop,
		ExitCacheMinRequests: 20,
		ExitIQWindow:         10,
		ExitIQMaxDegraded:    &iqMax,
		ExitLatencyMs:        &latency,
		ExitLatencyCount:     3,
		ExitLatencyWindow:    3,
		ExitErrorWindow:      3,
		ExitErrorCount:       &errCount,
		RecoverSuccessCount:  3,

		FirstTokenTestSamples:  10,
		EntryCacheMinRequests:  1,
		EntryIQPassCount:       1,
		ExitTrafficWindowHours: 24,
		BadWindowMinutes:       180,
		CooldownMinutes:        []int{5, 15, 30, 60},
		RecoverLatencyMs:       &recoverLatency,

		FallbackEnabled:       true,
		FallbackMinAccounts:   1,
		FallbackOrder:         DefaultSchedulingCenterFallbackOrder(),
		FallbackRequireIQPass: true,

		ManualOutHours: 24,

		RecordRetention: SchedulingCenterPlanMaxResults,

		EventRetentionDays: SchedulingCenterDefaultEventRetentionDays,
	}
}

type SchedulingCenterPoolMember struct {
	AccountID int64     `json:"account_id"`
	Source    string    `json:"source"`
	EnteredAt time.Time `json:"entered_at"`
}

// SchedulingCenterManagedPlan 是调度中心为账号创建的定时测试计划。
type SchedulingCenterManagedPlan struct {
	AccountID      int64
	PlanID         int64
	GroupID        int64
	CronExpression string
	ModelID        string
	IQStream       bool
	MaxResults     int
}

// SchedulingCenterTestRecord 是账号的一次智商检测记录。
type SchedulingCenterTestRecord struct {
	ID           int64     `json:"id"`
	Status       string    `json:"status"`
	FirstTokenMs *int64    `json:"first_token_ms"`
	ProbeTotalMs *int64    `json:"probe_total_ms"`
	LatencyMs    int64     `json:"latency_ms"`
	IQStatus     string    `json:"iq_status"`
	IQMessage    string    `json:"iq_message"`
	ErrorMessage string    `json:"error_message"`
	StartedAt    time.Time `json:"started_at"`
}

type SchedulingCenterRepository interface {
	// ListPoolMembers 只返回当前仍属于该分组的成员。
	ListPoolMembers(ctx context.Context, groupID int64) ([]SchedulingCenterPoolMember, error)
	// AddToSchedulingPool / RemoveFromSchedulingPool 返回实际写入 / 删除的账号 ID。
	AddToSchedulingPool(ctx context.Context, groupID int64, accountIDs []int64, source string) ([]int64, error)
	RemoveFromSchedulingPool(ctx context.Context, groupID int64, accountIDs []int64) ([]int64, error)

	ListAccountStates(ctx context.Context, groupID int64) ([]SchedulingCenterAccountState, error)
	GetAccountState(ctx context.Context, groupID, accountID int64) (*SchedulingCenterAccountState, error)
	UpsertAccountState(ctx context.Context, st *SchedulingCenterAccountState) error
	// CompareAndSetProbingState 仅当账号仍处于 probing 且 probe_started_at 未变时写入，返回是否写入。
	CompareAndSetProbingState(ctx context.Context, st *SchedulingCenterAccountState, expectProbeStartedAt time.Time) (bool, error)
	DeleteAccountStates(ctx context.Context, groupID int64, accountIDs []int64) error

	InsertEvents(ctx context.Context, events []SchedulingCenterEvent) error
	ListEvents(ctx context.Context, groupID int64, keyword string, limit, offset int) ([]SchedulingCenterEvent, int64, error)
	CountBadEvents(ctx context.Context, groupID int64, accountIDs []int64, since time.Time) (map[int64]int, error)
	// PruneEvents 删除分组中早于 before 的调度日志，返回删除行数。
	PruneEvents(ctx context.Context, groupID int64, before time.Time) (int64, error)

	// GetRecentRequests 返回每个账号在各自 since 之后的最近 limit 次真实请求（新 → 旧），
	// 成功来自 usage_logs，失败来自 ops_error_logs 中上游原因的错误。
	GetRecentRequests(ctx context.Context, since map[int64]time.Time, limit int) (map[int64][]SchedulingCenterRequestSample, error)
	// GetRecentIQStatuses 返回每个账号在各自 since 之后的最近 limit 次智商检测结论（新 → 旧）。
	GetRecentIQStatuses(ctx context.Context, since map[int64]time.Time, limit int) (map[int64][]string, error)
	GetManagedPlanID(ctx context.Context, accountID int64) (int64, bool, error)
	// GetNextPlanRunAt 返回分组托管检测计划中最早的 next_run_at；没有计划时返回 nil。
	GetNextPlanRunAt(ctx context.Context, groupID int64) (*time.Time, error)
	// ListForeignPlanOwners 返回本分组账号中检测计划归属于其他分组的账号及其生效设置。
	ListForeignPlanOwners(ctx context.Context, groupID int64) ([]SchedulingCenterPlanOwner, error)
	// ResetGroupData 按选项删除分组的调度池成员、账号状态、调度日志与托管计划的检测记录，返回各项删除行数。
	ResetGroupData(ctx context.Context, groupID int64, opts SchedulingCenterResetOptions) (*SchedulingCenterResetResult, error)

	GetAccountStats(ctx context.Context, accountIDs []int64, since time.Time) (map[int64]SchedulingCenterAccountStats, error)
	GetLatestIQResults(ctx context.Context, accountIDs []int64) (map[int64]SchedulingCenterIQResult, error)
	GetRecentTestFirstTokenAvg(ctx context.Context, accountIDs []int64, lastN int) (map[int64]float64, error)
	ListAccountTestRecords(ctx context.Context, accountID int64, limit int) ([]SchedulingCenterTestRecord, error)

	ListGroupSettings(ctx context.Context) ([]SchedulingCenterGroupSettings, error)
	// ListEnforcedPools 返回开启调度池生效的分组及其网关可用账号（池成员 + 探测通过的恢复中账号）。
	ListEnforcedPools(ctx context.Context) (map[int64][]int64, error)
	GetGroupSettings(ctx context.Context, groupID int64) (*SchedulingCenterGroupSettings, error)
	UpsertGroupSettings(ctx context.Context, settings *SchedulingCenterGroupSettings) (*SchedulingCenterGroupSettings, error)
	ListGroupAccountIDs(ctx context.Context, groupID int64) ([]int64, error)
	// ListGroupKeyAccountIDs 只返回分组中调度中心管理的 Key 类型账号（不含 OAuth / Setup Token）。
	ListGroupKeyAccountIDs(ctx context.Context, groupID int64) ([]int64, error)

	ListManagedPlans(ctx context.Context) ([]SchedulingCenterManagedPlan, error)
	InsertManagedPlan(ctx context.Context, accountID, planID, groupID int64) error
	UpdateManagedPlanGroup(ctx context.Context, accountID, groupID int64) error
}

// SchedulingCenterAccountMetrics 是返回给前端的单账号指标。
type SchedulingCenterAccountMetrics struct {
	AccountID           int64      `json:"account_id"`
	CacheRate           *float64   `json:"cache_rate"`
	Requests            int64      `json:"requests"`
	FirstTokenMs        *float64   `json:"first_token_ms"`
	FirstTokenSource    string     `json:"first_token_source,omitempty"`
	TestFirstTokenMs    *float64   `json:"test_first_token_ms"`
	TrafficFirstTokenMs *float64   `json:"traffic_first_token_ms"`
	IQStatus            string     `json:"iq_status"`
	IQMessage           string     `json:"iq_message,omitempty"`
	IQCheckedAt         *time.Time `json:"iq_checked_at,omitempty"`
}

type SchedulingCenterService struct {
	repo           SchedulingCenterRepository
	scheduledSvc   *ScheduledTestService
	accountTestSvc *AccountTestService
	// latencyProbe 使用只读账号仓储，测速不会改写账号状态。
	latencyProbe *AccountTestService
	db           *sql.DB
	// gate 为网关提供调度池生效分组的内存快照；调度池或设置变化后通知它刷新。
	gate *SchedulingCenterGate
	// opsSvc 用于判断运维监控是否开启：未开启时没有上游错误日志，「连续上游错误」规则不会触发。
	opsSvc *OpsService

	// measure 执行一次恢复探测测量（默认使用 AccountTestService，单测可替换）；probeShares 让多个分组共用同一次测量。
	measure      schedulingCenterMeasureFunc
	probeShareMu sync.Mutex
	probeShares  map[string]*schedulingCenterProbeShare
	probeSem     chan struct{}
	probeSync    bool // 单测：同步执行探测
	now          func() time.Time

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	doneCh    chan struct{}
}

func NewSchedulingCenterService(repo SchedulingCenterRepository, scheduledSvc *ScheduledTestService, accountTestSvc *AccountTestService, db *sql.DB) *SchedulingCenterService {
	s := &SchedulingCenterService{
		repo:           repo,
		scheduledSvc:   scheduledSvc,
		accountTestSvc: accountTestSvc,
		latencyProbe:   accountTestSvc.ReadOnlyClone(),
		db:             db,
		probeSem:       make(chan struct{}, schedulingCenterProbeConcurrency),
		now:            time.Now,
		stopCh:         make(chan struct{}),
		doneCh:         make(chan struct{}),
	}
	s.measure = s.defaultMeasure
	s.probeShares = map[string]*schedulingCenterProbeShare{}
	s.gate = NewSchedulingCenterGate(repo)
	return s
}

// ---------------------------------------------------------------------------
// 调度池
// ---------------------------------------------------------------------------

func (s *SchedulingCenterService) ListPoolMembers(ctx context.Context, groupID int64) ([]SchedulingCenterPoolMember, error) {
	members, err := s.repo.ListPoolMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if members == nil {
		members = []SchedulingCenterPoolMember{}
	}
	return members, nil
}

// ---------------------------------------------------------------------------
// 指标
// ---------------------------------------------------------------------------

// GetAccountMetrics 返回账号近 24 小时缓存率、首 Token 耗时与最近一次降智检测结果。
// 首 Token 优先取最近 testSamples 次智商检测的平均值，没有检测数据时退回近 24 小时真实请求平均值；
// 入池规则使用同一个值，保证列表所见即规则判断所用。
// GetAccountMetricsForGroup 按分组设置的检测样本数计算指标；groupID <= 0 时使用默认值。
func (s *SchedulingCenterService) GetAccountMetricsForGroup(ctx context.Context, groupID int64, accountIDs []int64) (map[int64]*SchedulingCenterAccountMetrics, error) {
	samples := defaultSchedulingCenterGroupSettings(0).FirstTokenTestSamples
	if groupID > 0 {
		gs, err := s.GetGroupSettings(ctx, groupID)
		if err != nil {
			return nil, err
		}
		if gs.FirstTokenTestSamples > 0 {
			samples = gs.FirstTokenTestSamples
		}
	}
	return s.GetAccountMetrics(ctx, accountIDs, samples)
}

func (s *SchedulingCenterService) GetAccountMetrics(ctx context.Context, accountIDs []int64, testSamples int) (map[int64]*SchedulingCenterAccountMetrics, error) {
	if testSamples <= 0 {
		testSamples = defaultSchedulingCenterGroupSettings(0).FirstTokenTestSamples
	}
	ids := normalizeSchedulingCenterIDs(accountIDs)
	result := make(map[int64]*SchedulingCenterAccountMetrics, len(ids))
	for _, id := range ids {
		result[id] = &SchedulingCenterAccountMetrics{AccountID: id}
	}
	if len(ids) == 0 {
		return result, nil
	}

	stats, err := s.repo.GetAccountStats(ctx, ids, time.Now().Add(-schedulingCenterMetricsWindow))
	if err != nil {
		return nil, err
	}
	for id, st := range stats {
		m, ok := result[id]
		if !ok {
			continue
		}
		m.Requests = st.Requests
		m.CacheRate = cacheRatePercent(AccountCacheRateTotals{
			Requests:            st.Requests,
			InputTokens:         st.InputTokens,
			CacheCreationTokens: st.CacheCreationTokens,
			CacheReadTokens:     st.CacheReadTokens,
		})
		m.TrafficFirstTokenMs = st.AvgFirstTokenMs
	}

	testTTFT, err := s.repo.GetRecentTestFirstTokenAvg(ctx, ids, testSamples)
	if err != nil {
		return nil, err
	}
	for id, v := range testTTFT {
		if m, ok := result[id]; ok {
			value := v
			m.TestFirstTokenMs = &value
		}
	}

	iq, err := s.repo.GetLatestIQResults(ctx, ids)
	if err != nil {
		return nil, err
	}
	for id, r := range iq {
		m, ok := result[id]
		if !ok {
			continue
		}
		checkedAt := r.CheckedAt
		m.IQStatus = r.Status
		m.IQMessage = r.Message
		m.IQCheckedAt = &checkedAt
	}

	for _, m := range result {
		switch {
		case m.TestFirstTokenMs != nil:
			m.FirstTokenMs, m.FirstTokenSource = m.TestFirstTokenMs, SchedulingCenterFirstTokenSourceTest
		case m.TrafficFirstTokenMs != nil:
			m.FirstTokenMs, m.FirstTokenSource = m.TrafficFirstTokenMs, SchedulingCenterFirstTokenSourceTraffic
		}
	}
	return result, nil
}

// ListAccountTestRecords 返回账号最近的智商检测记录（最多 100 条）。
func (s *SchedulingCenterService) ListAccountTestRecords(ctx context.Context, accountID int64, limit int) ([]SchedulingCenterTestRecord, error) {
	if limit <= 0 {
		limit = SchedulingCenterPlanMaxResults
	}
	if limit > SchedulingCenterMaxRecordRetention {
		limit = SchedulingCenterMaxRecordRetention
	}
	records, err := s.repo.ListAccountTestRecords(ctx, accountID, limit)
	if err != nil {
		return nil, err
	}
	if records == nil {
		records = []SchedulingCenterTestRecord{}
	}
	return records, nil
}

// ---------------------------------------------------------------------------
// 分组设置
// ---------------------------------------------------------------------------

func (s *SchedulingCenterService) ListGroupSettings(ctx context.Context) ([]SchedulingCenterGroupSettings, error) {
	list, err := s.repo.ListGroupSettings(ctx)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []SchedulingCenterGroupSettings{}
	}
	return list, nil
}

func (s *SchedulingCenterService) GetGroupSettings(ctx context.Context, groupID int64) (*SchedulingCenterGroupSettings, error) {
	settings, err := s.repo.GetGroupSettings(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return defaultSchedulingCenterGroupSettings(groupID), nil
	}
	return settings, nil
}

// UpdateGroupSettings 保存分组设置，随后立即同步定时测试计划并重新评估入池规则。
func (s *SchedulingCenterService) UpdateGroupSettings(ctx context.Context, in SchedulingCenterGroupSettings) (*SchedulingCenterGroupSettings, error) {
	defer s.gate.Notify()
	in.CronExpression = strings.TrimSpace(in.CronExpression)
	if in.CronExpression == "" {
		in.CronExpression = SchedulingCenterDefaultCron
	}
	in.ModelID = strings.TrimSpace(in.ModelID)
	if len(in.ModelID) > schedulingCenterMaxModelIDLen {
		return nil, fmt.Errorf("%w: model_id too long", ErrSchedulingCenterInvalidModel)
	}
	if _, err := computeNextRun(in.CronExpression, time.Now()); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSchedulingCenterInvalidCron, err)
	}
	if in.MinCacheRate != nil && (*in.MinCacheRate < 0 || *in.MinCacheRate > 100) {
		return nil, fmt.Errorf("%w: min_cache_rate must be between 0 and 100", ErrSchedulingCenterInvalidRule)
	}
	if in.MaxFirstTokenMs != nil && *in.MaxFirstTokenMs <= 0 {
		return nil, fmt.Errorf("%w: max_first_token_ms must be positive", ErrSchedulingCenterInvalidRule)
	}
	if err := validateSchedulingCenterExitSettings(&in); err != nil {
		return nil, err
	}
	if err := validateSchedulingCenterFallbackSettings(&in); err != nil {
		return nil, err
	}

	saved, err := s.repo.UpsertGroupSettings(ctx, &in)
	if err != nil {
		return nil, err
	}

	// 设置已落库；同步失败只记日志，后台任务会在下一轮补齐。
	if err := s.SyncPlans(ctx); err != nil {
		logger.LegacyPrintf("service.scheduling_center", "[SchedulingCenter] sync plans after settings update failed: %v", err)
	}
	if err := s.EvaluateRules(ctx); err != nil {
		logger.LegacyPrintf("service.scheduling_center", "[SchedulingCenter] evaluate rules after settings update failed: %v", err)
	}
	return saved, nil
}

// ---------------------------------------------------------------------------
// 定时测试计划同步
// ---------------------------------------------------------------------------

// SyncPlans 让「开启调度的分组内的账号」与调度中心管理的智商检测计划保持一致：
// 缺的创建、cron 或检测模型变了的更新、不再需要的删除。账号属于多个开启分组时取分组 ID 最小的设置。
func (s *SchedulingCenterService) SyncPlans(ctx context.Context) error {
	if s.scheduledSvc == nil {
		return nil
	}
	return s.withLock(ctx, "scheduling_center:sync_plans", func() error {
		settings, err := s.repo.ListGroupSettings(ctx)
		if err != nil {
			return err
		}
		sort.Slice(settings, func(i, j int) bool { return settings[i].GroupID < settings[j].GroupID })

		type desiredPlan struct {
			groupID int64
			cron    string
			model   string
			keep    int
		}
		desired := make(map[int64]desiredPlan)
		for _, gs := range settings {
			if !gs.Enabled {
				continue
			}
			// 只为 Key 类型账号建检测计划；OAuth / Setup Token 账号固定在调度池，不检测
			accountIDs, err := s.repo.ListGroupKeyAccountIDs(ctx, gs.GroupID)
			if err != nil {
				return err
			}
			for _, id := range accountIDs {
				if _, exists := desired[id]; !exists {
					desired[id] = desiredPlan{groupID: gs.GroupID, cron: gs.CronExpression, model: gs.ModelID, keep: schedulingCenterRecordRetention(&gs)}
				}
			}
		}

		managed, err := s.repo.ListManagedPlans(ctx)
		if err != nil {
			return err
		}
		existing := make(map[int64]SchedulingCenterManagedPlan, len(managed))
		for _, mp := range managed {
			existing[mp.AccountID] = mp
		}

		var errs []error
		for accountID, mp := range existing {
			want, ok := desired[accountID]
			if !ok {
				if err := s.scheduledSvc.DeletePlan(ctx, mp.PlanID); err != nil {
					errs = append(errs, fmt.Errorf("delete plan %d: %w", mp.PlanID, err))
				}
				continue
			}
			if mp.CronExpression != want.cron || mp.ModelID != want.model || !mp.IQStream || mp.MaxResults != want.keep {
				plan, err := s.scheduledSvc.GetPlan(ctx, mp.PlanID)
				if err != nil {
					errs = append(errs, fmt.Errorf("get plan %d: %w", mp.PlanID, err))
					continue
				}
				plan.CronExpression = want.cron
				plan.ModelID = want.model
				plan.IQStream = true
				plan.MaxResults = want.keep
				if _, err := s.scheduledSvc.UpdatePlan(ctx, plan); err != nil {
					errs = append(errs, fmt.Errorf("update plan %d: %w", mp.PlanID, err))
					continue
				}
			}
			if mp.GroupID != want.groupID {
				if err := s.repo.UpdateManagedPlanGroup(ctx, accountID, want.groupID); err != nil {
					errs = append(errs, err)
				}
			}
		}

		for accountID, want := range desired {
			if _, ok := existing[accountID]; ok {
				continue
			}
			plan, err := s.scheduledSvc.CreatePlan(ctx, &ScheduledTestPlan{
				AccountID:      accountID,
				ModelID:        want.model,
				CronExpression: want.cron,
				Enabled:        true,
				MaxResults:     want.keep,
				AutoRecover:    false,
				IQCheck:        true,
				IQStopOnFail:   false,
				IQStream:       true,
			})
			if err != nil {
				errs = append(errs, fmt.Errorf("create plan for account %d: %w", accountID, err))
				continue
			}
			if err := s.repo.InsertManagedPlan(ctx, accountID, plan.ID, want.groupID); err != nil {
				_ = s.scheduledSvc.DeletePlan(ctx, plan.ID)
				errs = append(errs, fmt.Errorf("record plan for account %d: %w", accountID, err))
			}
		}
		return errors.Join(errs...)
	})
}

// ---------------------------------------------------------------------------
// 入池规则
// ---------------------------------------------------------------------------

// SchedulingCenterRuleSatisfied 判断账号指标是否满足分组规则（所有已配置条件都要满足）。
// recentIQ 为最近的智商检测结论（新 → 旧）。缺少对应数据视为不满足；规则没有配置任何条件时一律不满足。
func SchedulingCenterRuleSatisfied(settings *SchedulingCenterGroupSettings, m *SchedulingCenterAccountMetrics, recentIQ []string) bool {
	if !settings.HasRuleCondition() || m == nil {
		return false
	}
	if settings.MinCacheRate != nil {
		minReq := int64(max(settings.EntryCacheMinRequests, 1))
		insufficient := m.CacheRate == nil || m.Requests < minReq
		// 调度池生效后池外账号拿不到真实流量，24h 缓存率会一直缺数据；此时不因缺数据卡住入池，
		// 入池后由「缓存率过低」出池规则兜住。
		if insufficient && !(settings.Enabled && settings.EnforceEnabled) {
			return false
		}
		if !insufficient && *m.CacheRate < *settings.MinCacheRate {
			return false
		}
	}
	if settings.RequireIQPass {
		need := max(settings.EntryIQPassCount, 1)
		if len(recentIQ) < need {
			return false
		}
		for _, status := range recentIQ[:need] {
			if status != IQVerdictPassed {
				return false
			}
		}
	}
	if settings.MaxFirstTokenMs != nil && (m.FirstTokenMs == nil || *m.FirstTokenMs > float64(*settings.MaxFirstTokenMs)) {
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// 后台任务
// ---------------------------------------------------------------------------

// Start 启动后台任务：每分钟评估出入池规则与账号恢复，每 5 分钟同步一次定时测试计划
// （覆盖分组成员变化）。多实例部署时由 PG advisory lock 保证同一时刻只有一个实例执行。
func (s *SchedulingCenterService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		go s.run()
	})
}

func (s *SchedulingCenterService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
		select {
		case <-s.doneCh:
		case <-time.After(5 * time.Second):
		}
		s.gate.Stop()
	})
}

func (s *SchedulingCenterService) run() {
	defer close(s.doneCh)
	select {
	case <-time.After(schedulingCenterWorkerStartDelay):
	case <-s.stopCh:
		return
	}
	ticker := time.NewTicker(schedulingCenterRuleInterval)
	defer ticker.Stop()
	tick := 0
	for {
		s.runTick(tick%schedulingCenterSyncEveryNTicks == 0)
		tick++
		select {
		case <-ticker.C:
		case <-s.stopCh:
			return
		}
	}
}

func (s *SchedulingCenterService) runTick(syncPlans bool) {
	ctx, cancel := context.WithTimeout(context.Background(), schedulingCenterWorkerTickTimeout)
	defer cancel()
	if syncPlans {
		if err := s.SyncPlans(ctx); err != nil {
			logger.LegacyPrintf("service.scheduling_center", "[SchedulingCenter] sync plans failed: %v", err)
		}
	}
	if err := s.EvaluateRules(ctx); err != nil {
		logger.LegacyPrintf("service.scheduling_center", "[SchedulingCenter] evaluate rules failed: %v", err)
	}
	if syncPlans {
		if err := s.PruneEvents(ctx); err != nil {
			logger.LegacyPrintf("service.scheduling_center", "[SchedulingCenter] prune events failed: %v", err)
		}
	}
}

// PruneEvents 按各分组的 EventRetentionDays 删除过期的调度日志（包括已关闭调度的分组）。
func (s *SchedulingCenterService) PruneEvents(ctx context.Context) error {
	return s.withLock(ctx, "scheduling_center:prune_events", func() error {
		all, err := s.repo.ListGroupSettings(ctx)
		if err != nil {
			return err
		}
		now := s.now()
		for i := range all {
			before := now.Add(-time.Duration(schedulingCenterEventRetentionDays(&all[i])) * 24 * time.Hour)
			n, err := s.repo.PruneEvents(ctx, all[i].GroupID, before)
			if err != nil {
				return err
			}
			if n > 0 {
				logger.LegacyPrintf("service.scheduling_center", "[SchedulingCenter] pruned %d events of group %d", n, all[i].GroupID)
			}
		}
		return nil
	})
}

// SetOpsService 注入运维监控服务（可为 nil）。
func (s *SchedulingCenterService) SetOpsService(ops *OpsService) {
	if s != nil {
		s.opsSvc = ops
	}
}

// OpsMonitoringEnabled 报告运维监控是否开启（错误日志是否在记录）；未注入时视为开启。
func (s *SchedulingCenterService) OpsMonitoringEnabled(ctx context.Context) bool {
	if s == nil || s.opsSvc == nil {
		return true
	}
	return s.opsSvc.IsMonitoringEnabled(ctx)
}

// IsSchedulingCenterFixedAccountType 报告账号类型是否固定在调度池（OAuth / Setup Token），
// 调度中心只处理其余的 Key 类型账号。
func IsSchedulingCenterFixedAccountType(accountType string) bool {
	return accountType == AccountTypeOAuth || accountType == AccountTypeSetupToken
}

// schedulingCenterEventRetentionDays 返回分组的调度日志保留天数（未设置时为默认值）。
func schedulingCenterEventRetentionDays(gs *SchedulingCenterGroupSettings) int {
	if gs == nil || gs.EventRetentionDays <= 0 {
		return SchedulingCenterDefaultEventRetentionDays
	}
	return gs.EventRetentionDays
}

// withLock 用 PG advisory lock 串行化后台任务与设置变更触发的同步；拿不到锁说明
// 其他实例/请求正在执行，本次直接跳过。没有 DB（单测）时直接执行。
func (s *SchedulingCenterService) withLock(ctx context.Context, key string, fn func() error) error {
	if s.db == nil {
		return fn()
	}
	release, acquired, err := tryAcquireDBAdvisoryLockWithError(ctx, s.db, hashAdvisoryLockID(key))
	if err != nil {
		return err
	}
	if !acquired {
		return nil
	}
	defer release()
	return fn()
}

func normalizeSchedulingCenterIDs(ids []int64) []int64 {
	out := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
		if len(out) >= schedulingCenterMaxBatchAccounts {
			break
		}
	}
	return out
}

// schedulingCenterRecordRetention 返回分组的检测记录保留条数（未设置时为默认值）。
func schedulingCenterRecordRetention(gs *SchedulingCenterGroupSettings) int {
	if gs == nil || gs.RecordRetention <= 0 {
		return SchedulingCenterPlanMaxResults
	}
	return gs.RecordRetention
}
