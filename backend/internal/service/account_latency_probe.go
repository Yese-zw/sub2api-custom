package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 调度中心测速：复用「账号管理 → 测试账号连接 → 常规请求」的完整请求路径（提示词 hi），
// 取其统计的首 Token 与总耗时。测速在只读账号仓储上运行：常规测试遇到 401/403/429 等
// 会改写账号状态（SetError / SetRateLimited / SetTempUnschedulable / ClearError / UpdateExtra），
// 只读仓储把这些写操作全部忽略，保证测速不影响真实调度。

// readOnlyAccountRepository 包装 AccountRepository：读操作透传，所有写操作静默忽略。
type readOnlyAccountRepository struct {
	AccountRepository
}

// NewReadOnlyAccountRepository 返回一个忽略所有写操作的账号仓储。
func NewReadOnlyAccountRepository(repo AccountRepository) AccountRepository {
	if repo == nil {
		return nil
	}
	if ro, ok := repo.(*readOnlyAccountRepository); ok {
		return ro
	}
	return &readOnlyAccountRepository{AccountRepository: repo}
}

func (r *readOnlyAccountRepository) Create(context.Context, *Account) error { return nil }
func (r *readOnlyAccountRepository) Update(context.Context, *Account) error { return nil }
func (r *readOnlyAccountRepository) Delete(context.Context, int64) error    { return nil }
func (r *readOnlyAccountRepository) UpdateLastUsed(context.Context, int64) error {
	return nil
}
func (r *readOnlyAccountRepository) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}
func (r *readOnlyAccountRepository) SetError(context.Context, int64, string) error { return nil }
func (r *readOnlyAccountRepository) ClearError(context.Context, int64) error       { return nil }
func (r *readOnlyAccountRepository) SetSchedulable(context.Context, int64, bool) error {
	return nil
}
func (r *readOnlyAccountRepository) AutoPauseExpiredAccounts(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (r *readOnlyAccountRepository) BindGroups(context.Context, int64, []int64) error { return nil }
func (r *readOnlyAccountRepository) SetRateLimited(context.Context, int64, time.Time) error {
	return nil
}
func (r *readOnlyAccountRepository) SetModelRateLimit(context.Context, int64, string, time.Time, ...string) error {
	return nil
}
func (r *readOnlyAccountRepository) SetOverloaded(context.Context, int64, time.Time) error {
	return nil
}
func (r *readOnlyAccountRepository) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	return nil
}
func (r *readOnlyAccountRepository) ClearTempUnschedulable(context.Context, int64) error { return nil }
func (r *readOnlyAccountRepository) ClearRateLimit(context.Context, int64) error         { return nil }
func (r *readOnlyAccountRepository) ClearAntigravityQuotaScopes(context.Context, int64) error {
	return nil
}
func (r *readOnlyAccountRepository) ClearModelRateLimits(context.Context, int64) error { return nil }
func (r *readOnlyAccountRepository) ClearModelRateLimit(context.Context, int64, string) error {
	return nil
}
func (r *readOnlyAccountRepository) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	return nil
}
func (r *readOnlyAccountRepository) UpdateSessionWindowEnd(context.Context, int64, time.Time) error {
	return nil
}
func (r *readOnlyAccountRepository) UpdateExtra(context.Context, int64, map[string]any) error {
	return nil
}
func (r *readOnlyAccountRepository) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	return 0, nil
}
func (r *readOnlyAccountRepository) IncrementQuotaUsed(context.Context, int64, float64) error {
	return nil
}
func (r *readOnlyAccountRepository) ResetQuotaUsedAndClearRateLimitCooldown(context.Context, int64) error {
	return nil
}
func (r *readOnlyAccountRepository) RevertProxyFallback(context.Context, int64) error { return nil }

// ReadOnlyClone 返回一个使用只读账号仓储的测试服务副本（不复制锁与缓存），用于不能有副作用的测速。
func (s *AccountTestService) ReadOnlyClone() *AccountTestService {
	if s == nil {
		return nil
	}
	clone := NewAccountTestService(
		NewReadOnlyAccountRepository(s.accountRepo),
		s.geminiTokenProvider,
		s.claudeTokenProvider,
		s.grokTokenProvider,
		s.antigravityGatewayService,
		s.httpUpstream,
		s.cfg,
		s.tlsFPProfileService,
	)
	clone.settingService = s.settingService
	clone.pluginManager = s.pluginManager
	clone.openaiGatewayService = s.openaiGatewayService
	clone.agentIdentityWS = s.agentIdentityWS
	clone.grokWSDialer = s.grokWSDialer
	return clone
}

// AccountLatencyProbeResult 是一次测速（常规测试，提示词 hi）的结果。
type AccountLatencyProbeResult struct {
	Success      bool
	FirstTokenMs *int64
	TotalMs      int64
	Error        string
}

// RunLatencyProbe 以后台方式执行一次常规连接测试（提示词使用默认的 hi），返回首 Token 与总耗时。
// 调用方应在 ReadOnlyClone 上调用，避免测试改写账号状态。
func (s *AccountTestService) RunLatencyProbe(ctx context.Context, accountID int64, modelID string) *AccountLatencyProbeResult {
	started := time.Now()
	w := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(w)
	ginCtx.Request = (&http.Request{}).WithContext(ctx)
	ginCtx.Set(accountTestBackgroundKey, true)

	testErr := s.TestAccountConnection(ginCtx, accountID, modelID, "", AccountTestModeDefault)
	result := parseLatencyProbeOutput(w.Body.String())
	if result.TotalMs <= 0 {
		result.TotalMs = time.Since(started).Milliseconds()
	}
	if testErr != nil && result.Error == "" {
		result.Error = testErr.Error()
	}
	if result.Error != "" {
		result.Success = false
	}
	if !result.Success {
		result.FirstTokenMs = nil
	}
	return result
}

func parseLatencyProbeOutput(body string) *AccountLatencyProbeResult {
	result := &AccountLatencyProbeResult{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var event TestEvent
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
			continue
		}
		switch event.Type {
		case "test_complete":
			result.Success = event.Success
			if event.TokenLatencyMs != nil {
				v := *event.TokenLatencyMs
				result.FirstTokenMs = &v
			}
			if event.TotalLatencyMs != nil {
				result.TotalMs = *event.TotalLatencyMs
			}
		case "error":
			result.Error = event.Error
			if event.TotalLatencyMs != nil {
				result.TotalMs = *event.TotalLatencyMs
			}
		}
	}
	return result
}
