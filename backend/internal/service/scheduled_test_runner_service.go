package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/robfig/cron/v3"
)

const scheduledTestDefaultMaxWorkers = 10

// ScheduledTestRunnerService periodically scans due test plans and executes them.
type ScheduledTestRunnerService struct {
	planRepo       ScheduledTestPlanRepository
	scheduledSvc   *ScheduledTestService
	accountTestSvc *AccountTestService
	rateLimitSvc   *RateLimitService
	cfg            *config.Config
	// latencyProbe 是只读账号仓储上的测试服务，用于调度中心计划的测速（不改写账号状态）。
	latencyProbe *AccountTestService

	cron      *cron.Cron
	startOnce sync.Once
	stopOnce  sync.Once
}

// NewScheduledTestRunnerService creates a new runner.
func NewScheduledTestRunnerService(
	planRepo ScheduledTestPlanRepository,
	scheduledSvc *ScheduledTestService,
	accountTestSvc *AccountTestService,
	rateLimitSvc *RateLimitService,
	cfg *config.Config,
) *ScheduledTestRunnerService {
	return &ScheduledTestRunnerService{
		planRepo:       planRepo,
		scheduledSvc:   scheduledSvc,
		accountTestSvc: accountTestSvc,
		rateLimitSvc:   rateLimitSvc,
		cfg:            cfg,
		latencyProbe:   accountTestSvc.ReadOnlyClone(),
	}
}

// Start begins the cron ticker (every minute).
func (s *ScheduledTestRunnerService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		loc := time.Local
		if s.cfg != nil {
			if parsed, err := time.LoadLocation(s.cfg.Timezone); err == nil && parsed != nil {
				loc = parsed
			}
		}

		c := cron.New(cron.WithParser(scheduledTestCronParser), cron.WithLocation(loc))
		_, err := c.AddFunc("* * * * *", func() { s.runScheduled() })
		if err != nil {
			logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] not started (invalid schedule): %v", err)
			return
		}
		s.cron = c
		s.cron.Start()
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] started (tick=every minute)")
	})
}

// Stop gracefully shuts down the cron scheduler.
func (s *ScheduledTestRunnerService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cron != nil {
			ctx := s.cron.Stop()
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] cron stop timed out")
			}
		}
	})
}

func (s *ScheduledTestRunnerService) runScheduled() {
	// Delay 10s so execution lands at ~:10 of each minute instead of :00.
	time.Sleep(10 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	now := time.Now()
	plans, err := s.planRepo.ListDue(ctx, now)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] ListDue error: %v", err)
		return
	}
	if len(plans) == 0 {
		return
	}

	logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] found %d due plans", len(plans))

	sem := make(chan struct{}, scheduledTestDefaultMaxWorkers)
	var wg sync.WaitGroup

	for _, plan := range plans {
		sem <- struct{}{}
		wg.Add(1)
		go func(p *ScheduledTestPlan) {
			defer wg.Done()
			defer func() { <-sem }()
			s.runOnePlan(ctx, p)
		}(plan)
	}

	wg.Wait()
}

func (s *ScheduledTestRunnerService) runOnePlan(ctx context.Context, plan *ScheduledTestPlan) {
	var result *ScheduledTestResult
	if plan.IQCheck {
		// 开启智商检测后不再做普通连通性测试，仅运行智商检测。
		result = s.runIQCheckOnly(ctx, plan)
		if plan.IQStopOnFail && result.IQStatus == IQVerdictIncorrect {
			s.pauseAccountForIQ(ctx, plan)
		}
	} else {
		var err error
		result, err = s.accountTestSvc.RunTestBackground(ctx, plan.AccountID, plan.ModelID)
		if err != nil {
			logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d RunTestBackground error: %v", plan.ID, err)
			return
		}
	}

	if err := s.scheduledSvc.SaveResult(ctx, plan.ID, plan.MaxResults, result); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d SaveResult error: %v", plan.ID, err)
	}

	// Auto-recover account if test succeeded and auto_recover is enabled.
	if result.Status == "success" && plan.AutoRecover {
		s.tryRecoverAccount(ctx, plan.AccountID, plan.ID)
	}

	nextRun, err := computeNextRun(plan.CronExpression, time.Now())
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d computeNextRun error: %v", plan.ID, err)
		return
	}

	if err := s.planRepo.UpdateAfterRun(ctx, plan.ID, time.Now(), nextRun); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d UpdateAfterRun error: %v", plan.ID, err)
	}
}

// tryRecoverAccount attempts to recover an account from recoverable runtime state.
func (s *ScheduledTestRunnerService) tryRecoverAccount(ctx context.Context, accountID int64, planID int64) {
	if s.rateLimitSvc == nil {
		return
	}

	recovery, err := s.rateLimitSvc.RecoverAccountAfterSuccessfulTest(ctx, accountID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover failed: %v", planID, err)
		return
	}
	if recovery == nil {
		return
	}

	if recovery.ClearedError {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover: account=%d recovered from error status", planID, accountID)
	}
	if recovery.ClearedRateLimit {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover: account=%d cleared rate-limit/runtime state", planID, accountID)
	}
}

// runIQCheckOnly runs the local candy IQ check and converts it into a scheduled test result.
// The result is "success" when the check ran to a verdict (passed or incorrect); a
// failed request or empty answer is "failed".
func (s *ScheduledTestRunnerService) runIQCheckOnly(ctx context.Context, plan *ScheduledTestPlan) *ScheduledTestResult {
	result := s.runIQCheck(ctx, plan)
	if plan.IQStream && s.latencyProbe != nil {
		// 调度中心计划：首 Token 不取智商题（推理模型会先思考整道题），
		// 而是按「测试账号连接 → 常规请求（hi）」单独测速；不支持智商检测的账号同样测速。
		applyLatencyProbe(result, s.latencyProbe.RunLatencyProbe(ctx, plan.AccountID, plan.ModelID))
	}
	return result
}

func (s *ScheduledTestRunnerService) runIQCheck(ctx context.Context, plan *ScheduledTestPlan) *ScheduledTestResult {
	startedAt := time.Now()
	result := &ScheduledTestResult{StartedAt: startedAt}
	task, err := s.accountTestSvc.RunIQCheck(ctx, plan.AccountID, plan.ModelID, plan.IQStream)
	result.FinishedAt = time.Now()
	result.LatencyMs = result.FinishedAt.Sub(startedAt).Milliseconds()
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d iq check error: %v", plan.ID, err)
		result.Status = "failed"
		result.ErrorMessage = err.Error()
		result.IQStatus = IQVerdictError
		result.IQMessage = "检测失败: " + err.Error()
		return result
	}
	result.Status = "success"
	if task.Status != "succeeded" {
		result.Status = "failed"
		result.ErrorMessage = task.Error
	}
	result.ResponseText = task.Answer
	if task.FirstTokenMs > 0 {
		ft := task.FirstTokenMs
		result.FirstTokenMs = &ft
	}
	result.IQStatus = task.Verdict
	result.IQMessage = IQCheckVerdictLabel(task)
	return result
}

// applyLatencyProbe 把测速结果写入检测结果：首 Token、测速总耗时；测速失败时首 Token 为空并在说明中注明。
func applyLatencyProbe(result *ScheduledTestResult, probe *AccountLatencyProbeResult) {
	result.FirstTokenMs = nil
	if probe == nil {
		return
	}
	total := probe.TotalMs
	result.ProbeTotalMs = &total
	if probe.Success && probe.FirstTokenMs != nil {
		ft := *probe.FirstTokenMs
		result.FirstTokenMs = &ft
		return
	}
	msg := strings.TrimSpace(probe.Error)
	if msg == "" {
		msg = "未返回内容"
	}
	result.IQMessage += "；测速失败：" + truncateForIQ(msg, 200)
}

// pauseAccountForIQ temporarily stops scheduling the account after a failed IQ check.
// Scheduling resumes automatically once the plan's stop duration has elapsed.
func (s *ScheduledTestRunnerService) pauseAccountForIQ(ctx context.Context, plan *ScheduledTestPlan) {
	minutes := plan.IQStopMinutes
	if minutes <= 0 {
		minutes = 60
	}
	until := time.Now().Add(time.Duration(minutes) * time.Minute)
	reason := fmt.Sprintf("智商检测未通过，已停止调度 %d 分钟（定时测试计划 #%d）", minutes, plan.ID)
	if err := s.accountTestSvc.accountRepo.SetTempUnschedulable(ctx, plan.AccountID, until, reason); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d pause account=%d failed: %v", plan.ID, plan.AccountID, err)
		return
	}
	logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d iq check failed: account=%d paused until %s", plan.ID, plan.AccountID, until.Format(time.RFC3339))
}
