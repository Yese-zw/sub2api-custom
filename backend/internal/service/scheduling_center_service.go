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

	SchedulingCenterFirstTokenSourceTest    = "test"
	SchedulingCenterFirstTokenSourceTraffic = "traffic"

	SchedulingCenterDefaultCron       = "*/30 * * * *"
	SchedulingCenterPlanMaxResults    = 100
	schedulingCenterMetricsWindow     = 24 * time.Hour
	schedulingCenterMaxBatchAccounts  = 1000
	schedulingCenterRuleInterval      = time.Minute
	schedulingCenterSyncEveryNTicks   = 5
	schedulingCenterWorkerStartDelay  = 30 * time.Second
	schedulingCenterWorkerTickTimeout = 2 * time.Minute
	schedulingCenterMaxModelIDLen     = 100
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
	ExitLatencyCount     int      `json:"exit_latency_count"`
	ExitErrorCount       *int     `json:"exit_error_count"`
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
		ExitErrorCount:       &errCount,
		RecoverSuccessCount:  3,

		FirstTokenTestSamples:  10,
		EntryCacheMinRequests:  1,
		EntryIQPassCount:       1,
		ExitTrafficWindowHours: 24,
		BadWindowMinutes:       180,
		CooldownMinutes:        []int{5, 15, 30, 60},
		RecoverLatencyMs:       &recoverLatency,
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
}

// SchedulingCenterTestRecord 是账号的一次智商检测记录。
type SchedulingCenterTestRecord struct {
	ID           int64     `json:"id"`
	Status       string    `json:"status"`
	FirstTokenMs *int64    `json:"first_token_ms"`
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

	// GetRecentRequests 返回每个账号在各自 since 之后的最近 limit 次真实请求（新 → 旧），
	// 成功来自 usage_logs，失败来自 ops_error_logs 中上游原因的错误。
	GetRecentRequests(ctx context.Context, since map[int64]time.Time, limit int) (map[int64][]SchedulingCenterRequestSample, error)
	// GetRecentIQStatuses 返回每个账号在各自 since 之后的最近 limit 次智商检测结论（新 → 旧）。
	GetRecentIQStatuses(ctx context.Context, since map[int64]time.Time, limit int) (map[int64][]string, error)
	GetManagedPlanID(ctx context.Context, accountID int64) (int64, bool, error)

	GetAccountStats(ctx context.Context, accountIDs []int64, since time.Time) (map[int64]SchedulingCenterAccountStats, error)
	GetLatestIQResults(ctx context.Context, accountIDs []int64) (map[int64]SchedulingCenterIQResult, error)
	GetRecentTestFirstTokenAvg(ctx context.Context, accountIDs []int64, lastN int) (map[int64]float64, error)
	ListAccountTestRecords(ctx context.Context, accountID int64, limit int) ([]SchedulingCenterTestRecord, error)

	ListGroupSettings(ctx context.Context) ([]SchedulingCenterGroupSettings, error)
	GetGroupSettings(ctx context.Context, groupID int64) (*SchedulingCenterGroupSettings, error)
	UpsertGroupSettings(ctx context.Context, settings *SchedulingCenterGroupSettings) (*SchedulingCenterGroupSettings, error)
	ListGroupAccountIDs(ctx context.Context, groupID int64) ([]int64, error)

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
	db             *sql.DB

	// probe 执行一次恢复探测；默认使用 AccountTestService，单测可替换。
	probe     schedulingCenterProbeFunc
	probeSem  chan struct{}
	probeSync bool // 单测：同步执行探测
	now       func() time.Time

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
		db:             db,
		probeSem:       make(chan struct{}, schedulingCenterProbeConcurrency),
		now:            time.Now,
		stopCh:         make(chan struct{}),
		doneCh:         make(chan struct{}),
	}
	s.probe = s.defaultProbe
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
	if limit <= 0 || limit > SchedulingCenterPlanMaxResults {
		limit = SchedulingCenterPlanMaxResults
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
		}
		desired := make(map[int64]desiredPlan)
		for _, gs := range settings {
			if !gs.Enabled {
				continue
			}
			accountIDs, err := s.repo.ListGroupAccountIDs(ctx, gs.GroupID)
			if err != nil {
				return err
			}
			for _, id := range accountIDs {
				if _, exists := desired[id]; !exists {
					desired[id] = desiredPlan{groupID: gs.GroupID, cron: gs.CronExpression, model: gs.ModelID}
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
			if mp.CronExpression != want.cron || mp.ModelID != want.model || !mp.IQStream {
				plan, err := s.scheduledSvc.GetPlan(ctx, mp.PlanID)
				if err != nil {
					errs = append(errs, fmt.Errorf("get plan %d: %w", mp.PlanID, err))
					continue
				}
				plan.CronExpression = want.cron
				plan.ModelID = want.model
				plan.IQStream = true
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
				MaxResults:     SchedulingCenterPlanMaxResults,
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
		if m.CacheRate == nil || m.Requests < minReq || *m.CacheRate < *settings.MinCacheRate {
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
