package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// SchedulingCenterHandler 调度中心（独立管理视图，不影响网关调度）。
type SchedulingCenterHandler struct {
	svc *service.SchedulingCenterService
}

func NewSchedulingCenterHandler(svc *service.SchedulingCenterService) *SchedulingCenterHandler {
	return &SchedulingCenterHandler{svc: svc}
}

func parseSchedulingCenterID(c *gin.Context, what string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid "+what+" id")
		return 0, false
	}
	return id, true
}

func (h *SchedulingCenterHandler) respondPool(c *gin.Context, groupID int64) {
	view, err := h.svc.GetPoolView(c.Request.Context(), groupID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, view)
}

// ListEvents GET /admin/scheduling-center/groups/:id/events
func (h *SchedulingCenterHandler) ListEvents(c *gin.Context) {
	groupID, ok := parseSchedulingCenterID(c, "group")
	if !ok {
		return
	}
	page, pageSize := response.ParsePagination(c)
	events, total, err := h.svc.ListEvents(c.Request.Context(), groupID, c.Query("keyword"), page, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Paginated(c, events, total, page, pageSize)
}

// GetPool GET /admin/scheduling-center/groups/:id/pool
func (h *SchedulingCenterHandler) GetPool(c *gin.Context) {
	groupID, ok := parseSchedulingCenterID(c, "group")
	if !ok {
		return
	}
	h.respondPool(c, groupID)
}

type schedulingCenterMoveRequest struct {
	AccountIDs []int64 `json:"account_ids" binding:"required"`
	Pool       string  `json:"pool" binding:"required"`
}

// MovePool PUT /admin/scheduling-center/groups/:id/pool
func (h *SchedulingCenterHandler) MovePool(c *gin.Context) {
	groupID, ok := parseSchedulingCenterID(c, "group")
	if !ok {
		return
	}
	var req schedulingCenterMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	if err := h.svc.MoveAccounts(c.Request.Context(), groupID, req.AccountIDs, req.Pool); err != nil {
		if errors.Is(err, service.ErrSchedulingCenterInvalidPool) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	h.respondPool(c, groupID)
}

type schedulingCenterMetricsRequest struct {
	AccountIDs []int64 `json:"account_ids"`
	// GroupID 可选：按该分组的「首 Token 检测样本数」计算
	GroupID int64 `json:"group_id"`
}

// GetMetrics POST /admin/scheduling-center/metrics
func (h *SchedulingCenterHandler) GetMetrics(c *gin.Context) {
	var req schedulingCenterMetricsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	metrics, err := h.svc.GetAccountMetricsForGroup(c.Request.Context(), req.GroupID, req.AccountIDs)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"metrics": metrics})
}

// ListSettings GET /admin/scheduling-center/settings
func (h *SchedulingCenterHandler) ListSettings(c *gin.Context) {
	list, err := h.svc.ListGroupSettings(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"settings": list})
}

// GetSettings GET /admin/scheduling-center/groups/:id/settings
func (h *SchedulingCenterHandler) GetSettings(c *gin.Context) {
	groupID, ok := parseSchedulingCenterID(c, "group")
	if !ok {
		return
	}
	settings, err := h.svc.GetGroupSettings(c.Request.Context(), groupID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, settings)
}

type schedulingCenterSettingsRequest struct {
	Enabled         bool     `json:"enabled"`
	CronExpression  string   `json:"cron_expression"`
	ModelID         string   `json:"model_id"`
	RuleEnabled     bool     `json:"rule_enabled"`
	MinCacheRate    *float64 `json:"min_cache_rate"`
	RequireIQPass   bool     `json:"require_iq_pass"`
	MaxFirstTokenMs *int64   `json:"max_first_token_ms"`

	ExitEnabled          bool     `json:"exit_enabled"`
	ExitCacheDropPoints  *float64 `json:"exit_cache_drop_points"`
	ExitCacheMinRequests int      `json:"exit_cache_min_requests"`
	ExitIQWindow         int      `json:"exit_iq_window"`
	ExitIQMaxDegraded    *int     `json:"exit_iq_max_degraded"`
	ExitLatencyMs        *int64   `json:"exit_latency_ms"`
	ExitLatencyCount     int      `json:"exit_latency_count"`
	ExitErrorCount       *int     `json:"exit_error_count"`
	ExitLatencyWindow    int      `json:"exit_latency_window"`
	ExitErrorWindow      int      `json:"exit_error_window"`
	RecoverSuccessCount  int      `json:"recover_success_count"`

	FirstTokenTestSamples  int    `json:"first_token_test_samples"`
	EntryCacheMinRequests  int    `json:"entry_cache_min_requests"`
	EntryIQPassCount       int    `json:"entry_iq_pass_count"`
	ExitTrafficWindowHours int    `json:"exit_traffic_window_hours"`
	BadWindowMinutes       int    `json:"bad_window_minutes"`
	CooldownMinutes        []int  `json:"cooldown_minutes"`
	RecoverLatencyMs       *int64 `json:"recover_latency_ms"`

	FallbackEnabled       bool     `json:"fallback_enabled"`
	FallbackMinAccounts   int      `json:"fallback_min_accounts"`
	FallbackOrder         []string `json:"fallback_order"`
	FallbackRequireIQPass bool     `json:"fallback_require_iq_pass"`

	// ManualOutHours 为指针：旧客户端不传时保持默认 24 小时，传 0 表示永久。
	ManualOutHours *int `json:"manual_out_hours"`

	// RecordRetention 不传或为 0 时使用默认 100 条。
	RecordRetention int `json:"record_retention"`

	// EventRetentionDays 不传或为 0 时使用默认 30 天。
	EventRetentionDays int `json:"event_retention_days"`

	// EnforceEnabled 调度池生效：开启后网关在该分组只从调度池选账号（池内无可用账号时回退原逻辑）。
	EnforceEnabled bool `json:"enforce_enabled"`
}

// UpdateSettings PUT /admin/scheduling-center/groups/:id/settings
func (h *SchedulingCenterHandler) UpdateSettings(c *gin.Context) {
	groupID, ok := parseSchedulingCenterID(c, "group")
	if !ok {
		return
	}
	var req schedulingCenterSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	saved, err := h.svc.UpdateGroupSettings(c.Request.Context(), service.SchedulingCenterGroupSettings{
		GroupID:         groupID,
		Enabled:         req.Enabled,
		CronExpression:  req.CronExpression,
		ModelID:         req.ModelID,
		RuleEnabled:     req.RuleEnabled,
		MinCacheRate:    req.MinCacheRate,
		RequireIQPass:   req.RequireIQPass,
		MaxFirstTokenMs: req.MaxFirstTokenMs,

		ExitEnabled:          req.ExitEnabled,
		ExitCacheDropPoints:  req.ExitCacheDropPoints,
		ExitCacheMinRequests: req.ExitCacheMinRequests,
		ExitIQWindow:         req.ExitIQWindow,
		ExitIQMaxDegraded:    req.ExitIQMaxDegraded,
		ExitLatencyMs:        req.ExitLatencyMs,
		ExitLatencyCount:     req.ExitLatencyCount,
		ExitErrorCount:       req.ExitErrorCount,
		ExitLatencyWindow:    req.ExitLatencyWindow,
		ExitErrorWindow:      req.ExitErrorWindow,
		RecoverSuccessCount:  req.RecoverSuccessCount,

		FirstTokenTestSamples:  req.FirstTokenTestSamples,
		EntryCacheMinRequests:  req.EntryCacheMinRequests,
		EntryIQPassCount:       req.EntryIQPassCount,
		ExitTrafficWindowHours: req.ExitTrafficWindowHours,
		BadWindowMinutes:       req.BadWindowMinutes,
		CooldownMinutes:        req.CooldownMinutes,
		RecoverLatencyMs:       req.RecoverLatencyMs,

		FallbackEnabled:       req.FallbackEnabled,
		FallbackMinAccounts:   req.FallbackMinAccounts,
		FallbackOrder:         req.FallbackOrder,
		FallbackRequireIQPass: req.FallbackRequireIQPass,

		ManualOutHours: manualOutHoursOrDefault(req.ManualOutHours),

		RecordRetention: req.RecordRetention,

		EventRetentionDays: req.EventRetentionDays,

		EnforceEnabled: req.EnforceEnabled,
	})
	if err != nil {
		if errors.Is(err, service.ErrSchedulingCenterInvalidCron) || errors.Is(err, service.ErrSchedulingCenterInvalidRule) ||
			errors.Is(err, service.ErrSchedulingCenterInvalidModel) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, saved)
}

// ListAccountRecords GET /admin/scheduling-center/accounts/:id/records
func (h *SchedulingCenterHandler) ListAccountRecords(c *gin.Context) {
	accountID, ok := parseSchedulingCenterID(c, "account")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	records, err := h.svc.ListAccountTestRecords(c.Request.Context(), accountID, limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"records": records})
}

func parseSchedulingCenterHours(c *gin.Context) int {
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "0"))
	return hours
}

// GetComparison GET /admin/scheduling-center/groups/:id/comparison?hours=24
func (h *SchedulingCenterHandler) GetComparison(c *gin.Context) {
	groupID, ok := parseSchedulingCenterID(c, "group")
	if !ok {
		return
	}
	result, err := h.svc.GetComparison(c.Request.Context(), groupID, parseSchedulingCenterHours(c))
	if err != nil {
		if errors.Is(err, service.ErrSchedulingCenterInvalidRange) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

// Export GET /admin/scheduling-center/groups/:id/export?hours=72
func (h *SchedulingCenterHandler) Export(c *gin.Context) {
	groupID, ok := parseSchedulingCenterID(c, "group")
	if !ok {
		return
	}
	result, err := h.svc.Export(c.Request.Context(), groupID, parseSchedulingCenterHours(c))
	if err != nil {
		if errors.Is(err, service.ErrSchedulingCenterInvalidRange) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func manualOutHoursOrDefault(v *int) int {
	if v == nil {
		return 24
	}
	return *v
}

// ResetGroup POST /admin/scheduling-center/groups/:id/reset
func (h *SchedulingCenterHandler) ResetGroup(c *gin.Context) {
	groupID, ok := parseSchedulingCenterID(c, "group")
	if !ok {
		return
	}
	var opts service.SchedulingCenterResetOptions
	if err := c.ShouldBindJSON(&opts); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	result, err := h.svc.ResetGroup(c.Request.Context(), groupID, opts)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSchedulingCenterResetNothing):
			response.BadRequest(c, err.Error())
		case errors.Is(err, service.ErrSchedulingCenterBusy):
			response.Error(c, http.StatusConflict, err.Error())
		default:
			response.InternalError(c, err.Error())
		}
		return
	}
	response.Success(c, result)
}
