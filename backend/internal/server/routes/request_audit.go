package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"

	"github.com/gin-gonic/gin"
)

func registerRequestAuditRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	if h == nil || h.Admin == nil || h.Admin.RequestAudit == nil {
		return
	}
	g := admin.Group("/request-audit")
	{
		g.GET("/settings", h.Admin.RequestAudit.GetSettings)
		g.PUT("/settings", h.Admin.RequestAudit.UpdateSettings)
		g.GET("/usage/:id", h.Admin.RequestAudit.GetByUsageLog)
	}
}
