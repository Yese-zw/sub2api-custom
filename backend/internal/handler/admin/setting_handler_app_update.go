package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// GetAppUpdateSettings 获取桌面 App 更新配置
// GET /api/v1/admin/settings/app-update
func (h *SettingHandler) GetAppUpdateSettings(c *gin.Context) {
	settings, err := h.settingService.GetAppUpdateSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"settings": settings,
		"targets":  service.AppUpdateTargets,
	})
}

// UpdateAppUpdateSettings 更新桌面 App 更新配置
// PUT /api/v1/admin/settings/app-update
func (h *SettingHandler) UpdateAppUpdateSettings(c *gin.Context) {
	var req service.AppUpdateSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.settingService.SetAppUpdateSettings(c.Request.Context(), &req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	updated, err := h.settingService.GetAppUpdateSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"settings": updated,
		"targets":  service.AppUpdateTargets,
	})
}
