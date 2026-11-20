package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GetCheckinSettings returns the daily check-in configuration.
// GET /api/v1/admin/settings/checkin
func (h *SettingHandler) GetCheckinSettings(c *gin.Context) {
	settings, err := h.settingService.GetCheckinSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

// UpdateCheckinSettings validates and saves the daily check-in configuration.
// PUT /api/v1/admin/settings/checkin
func (h *SettingHandler) UpdateCheckinSettings(c *gin.Context) {
	var req service.CheckinSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.settingService.SetCheckinSettings(c.Request.Context(), &req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	updated, err := h.settingService.GetCheckinSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, updated)
}
