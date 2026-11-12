package admin

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type systemBalanceAdjustRequest struct {
	Amount json.Number `json:"amount"`
}

func (h *DashboardHandler) GetSystemAvailableBalance(c *gin.Context) {
	if h == nil || h.systemBalance == nil {
		response.Error(c, 503, "System available balance is unavailable")
		return
	}
	balance, err := h.systemBalance.Get(c.Request.Context())
	if err != nil {
		response.Error(c, 500, "Failed to get system available balance")
		return
	}
	response.Success(c, gin.H{"balance": balance.String()})
}

func (h *DashboardHandler) RechargeSystemAvailableBalance(c *gin.Context) {
	h.adjustSystemAvailableBalance(c, true)
}

func (h *DashboardHandler) DeductSystemAvailableBalance(c *gin.Context) {
	h.adjustSystemAvailableBalance(c, false)
}

func (h *DashboardHandler) adjustSystemAvailableBalance(c *gin.Context, recharge bool) {
	if h == nil || h.systemBalance == nil {
		response.Error(c, 503, "System available balance is unavailable")
		return
	}
	amount, ok := readSystemBalanceAmount(c)
	if !ok {
		return
	}
	adjust := h.systemBalance.Deduct
	if recharge {
		adjust = h.systemBalance.Recharge
	}
	balance, err := adjust(c.Request.Context(), amount)
	if errors.Is(err, service.ErrSystemAvailableBalanceAmount) {
		response.BadRequest(c, "Amount must be greater than zero")
		return
	}
	if err != nil {
		response.Error(c, 500, "Failed to adjust system available balance")
		return
	}
	response.Success(c, gin.H{"balance": balance.String()})
}

func readSystemBalanceAmount(c *gin.Context) (string, bool) {
	var req systemBalanceAdjustRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(c, "Invalid request body")
		return "", false
	}
	amount := strings.TrimSpace(req.Amount.String())
	if amount == "" {
		response.BadRequest(c, "Amount must be greater than zero")
		return "", false
	}
	return amount, true
}
