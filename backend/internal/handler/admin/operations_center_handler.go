package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ListOperationsSites 返回按 base_url 去重后的上游站点及其缓存额度。
// GET /api/v1/admin/operations-center/sites
func (h *AccountHandler) ListOperationsSites(c *gin.Context) {
	ctx := c.Request.Context()
	all := make([]service.Account, 0)
	for page := 1; ; page++ {
		items, total, err := h.adminService.ListAccounts(ctx, page, dataPageCap, "", "apikey,upstream", "", "", 0, "", "name", "asc")
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		all = append(all, items...)
		if len(items) == 0 || int64(page*dataPageCap) >= total {
			break
		}
	}
	sites := service.BuildOperationsSites(all)
	response.Success(c, gin.H{
		"items": sites,
		"total": len(sites),
	})
}

// GetAccountRequestRanking 返回时间范围内各账号的请求数据，按请求数从高到低排序。
// 支持 start_time/end_time（RFC3339，精确到秒）；未传时回退到 start_date/end_date（按天，默认近 7 天）。
// group_by=site（默认，按 base_url 去重合并）| account（按账号）。
// GET /api/v1/admin/operations-center/account-ranking
func (h *DashboardHandler) GetAccountRequestRanking(c *gin.Context) {
	startTime, endTime, ok := parseOperationsTimeRange(c)
	if !ok {
		return
	}
	limit := 100
	if raw := c.Query("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 1000 {
		limit = 1000
	}

	groupBySite := strings.TrimSpace(c.DefaultQuery("group_by", "site")) != "account"
	ranking, err := h.dashboardService.GetAccountRequestRanking(c.Request.Context(), startTime, endTime, limit, groupBySite)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get account request ranking")
		return
	}
	response.Success(c, gin.H{
		"group_by":           ranking.GroupBy,
		"items":              ranking.Items,
		"sites":              ranking.Sites,
		"total_sites":        ranking.TotalSites,
		"total_requests":     ranking.TotalRequests,
		"total_tokens":       ranking.TotalTokens,
		"total_account_cost": ranking.TotalAccountCost,
		"total_user_cost":    ranking.TotalUserCost,
		"total_accounts":     ranking.TotalAccounts,
		"total_errors":       ranking.TotalErrors,
		"start_time":         startTime.UTC().Format(time.RFC3339),
		"end_time":           endTime.UTC().Format(time.RFC3339),
	})
}

// parseOperationsTimeRange 解析运营中心时间范围：优先 start_time/end_time（RFC3339），
// 否则回退到 start_date/end_date（按天，默认近 7 天）。解析失败时已写入 400 响应并返回 false。
func parseOperationsTimeRange(c *gin.Context) (time.Time, time.Time, bool) {
	startTime, endTime := parseTimeRange(c)
	if raw := strings.TrimSpace(c.Query("start_time")); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			response.BadRequest(c, "Invalid start_time")
			return startTime, endTime, false
		}
		startTime = t
		endTime = time.Now().Add(time.Minute)
		if rawEnd := strings.TrimSpace(c.Query("end_time")); rawEnd != "" {
			e, err := time.Parse(time.RFC3339, rawEnd)
			if err != nil {
				response.BadRequest(c, "Invalid end_time")
				return startTime, endTime, false
			}
			endTime = e
		}
	}
	if !endTime.After(startTime) {
		response.BadRequest(c, "end_time must be after start_time")
		return startTime, endTime, false
	}
	if endTime.Sub(startTime) > 366*24*time.Hour {
		response.BadRequest(c, "time range too large")
		return startTime, endTime, false
	}
	return startTime, endTime, true
}

// GetAccountConsumption 返回各账号近 24 小时 / 7 天的账号口径消耗（用于站点余额可用天数估算）。
// GET /api/v1/admin/operations-center/consumption
func (h *DashboardHandler) GetAccountConsumption(c *gin.Context) {
	items, err := h.dashboardService.GetAccountConsumption(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get account consumption")
		return
	}
	response.Success(c, gin.H{"items": items})
}

// GetSiteTrend 返回按 base_url 合并的站点请求 / 成本 / 失败趋势。
// granularity=hour|day（缺省：范围 ≤ 3 天按小时，否则按天）；top=前 N 个站点（默认 8，最大 20）。
// GET /api/v1/admin/operations-center/site-trend
func (h *DashboardHandler) GetSiteTrend(c *gin.Context) {
	startTime, endTime, ok := parseOperationsTimeRange(c)
	if !ok {
		return
	}
	granularity := strings.TrimSpace(c.Query("granularity"))
	if granularity != "hour" && granularity != "day" {
		granularity = "day"
		if endTime.Sub(startTime) <= 72*time.Hour {
			granularity = "hour"
		}
	}
	if granularity == "hour" && endTime.Sub(startTime) > 31*24*time.Hour {
		granularity = "day"
	}
	topN := 8
	if raw := c.Query("top"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			topN = v
		}
	}
	if topN > 20 {
		topN = 20
	}
	loc := strings.TrimSpace(c.Query("timezone"))
	trend, err := h.dashboardService.GetSiteTrend(c.Request.Context(), startTime, endTime, granularity, loc, topN)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get site trend")
		return
	}
	response.Success(c, trend)
}

// GetRequestHeatmap 返回按星期 × 小时聚合的请求热力图。
// GET /api/v1/admin/operations-center/heatmap
func (h *DashboardHandler) GetRequestHeatmap(c *gin.Context) {
	startTime, endTime, ok := parseOperationsTimeRange(c)
	if !ok {
		return
	}
	cells, err := h.dashboardService.GetRequestHeatmap(c.Request.Context(), startTime, endTime, strings.TrimSpace(c.Query("timezone")))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get request heatmap")
		return
	}
	response.Success(c, gin.H{"cells": cells})
}

// GetAccountHealth 返回所有账号的健康汇总、问题列表与闲置账号。
// GET /api/v1/admin/operations-center/account-health
func (h *AccountHandler) GetAccountHealth(c *gin.Context) {
	ctx := c.Request.Context()
	all := make([]service.Account, 0)
	for page := 1; ; page++ {
		items, total, err := h.adminService.ListAccounts(ctx, page, dataPageCap, "", "", "", "", 0, "", "name", "asc")
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		all = append(all, items...)
		if len(items) == 0 || int64(page*dataPageCap) >= total {
			break
		}
	}
	response.Success(c, service.BuildAccountHealth(all, time.Now()))
}
