package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RequestAuditHandler 网关请求审计（完整输入/输出）管理接口。
type RequestAuditHandler struct {
	svc *service.RequestAuditService
}

func NewRequestAuditHandler(svc *service.RequestAuditService) *RequestAuditHandler {
	return &RequestAuditHandler{svc: svc}
}

// Service 供网关路由装配抓取中间件。
func (h *RequestAuditHandler) Service() *service.RequestAuditService {
	if h == nil {
		return nil
	}
	return h.svc
}

// GetSettings GET /api/v1/admin/request-audit/settings
func (h *RequestAuditHandler) GetSettings(c *gin.Context) {
	response.Success(c, h.svc.Settings(c.Request.Context()))
}

// UpdateSettings PUT /api/v1/admin/request-audit/settings
func (h *RequestAuditHandler) UpdateSettings(c *gin.Context) {
	var req service.RequestAuditSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	saved, err := h.svc.UpdateSettings(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, saved)
}

// GetByUsageLog GET /api/v1/admin/request-audit/usage/:id
func (h *RequestAuditHandler) GetByUsageLog(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid usage log id")
		return
	}
	item, err := h.svc.GetForUsageLog(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}
