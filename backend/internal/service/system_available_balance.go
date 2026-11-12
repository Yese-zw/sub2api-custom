package service

import (
	"context"
	"errors"
	"strings"

	"github.com/shopspring/decimal"
)

// ErrSystemAvailableBalanceAmount 表示充值或扣除金额不是正数。
var ErrSystemAvailableBalanceAmount = errors.New("system available balance amount must be positive")

// 单次充值或扣除的上限，避免误输入把账本打飞。
var maxSystemAvailableBalanceAdjust = decimal.NewFromInt(1_000_000_000_000)

// SystemAvailableBalanceRepository 持久化全站可用余额。
// 扣减发生在 usage_logs 插入触发器里，这里只负责读取和管理员的充值/扣除。
type SystemAvailableBalanceRepository interface {
	Get(ctx context.Context) (decimal.Decimal, error)
	Add(ctx context.Context, delta decimal.Decimal) (decimal.Decimal, error)
}

// SystemAvailableBalanceService 管理系统可用余额。
type SystemAvailableBalanceService struct {
	repo SystemAvailableBalanceRepository
}

func NewSystemAvailableBalanceService(repo SystemAvailableBalanceRepository) *SystemAvailableBalanceService {
	return &SystemAvailableBalanceService{repo: repo}
}

func (s *SystemAvailableBalanceService) Get(ctx context.Context) (decimal.Decimal, error) {
	if s == nil || s.repo == nil {
		return decimal.Zero, errors.New("system available balance is unavailable")
	}
	return s.repo.Get(ctx)
}

// Recharge 在当前余额上增加一笔正数，不能用来覆盖余额。
func (s *SystemAvailableBalanceService) Recharge(ctx context.Context, amount string) (decimal.Decimal, error) {
	delta, err := parsePositiveSystemBalanceAmount(amount)
	if err != nil {
		return decimal.Zero, err
	}
	return s.repo.Add(ctx, delta)
}

// Deduct 在当前余额上减掉一笔正数。余额允许变成负数，不拦截后续请求。
func (s *SystemAvailableBalanceService) Deduct(ctx context.Context, amount string) (decimal.Decimal, error) {
	delta, err := parsePositiveSystemBalanceAmount(amount)
	if err != nil {
		return decimal.Zero, err
	}
	return s.repo.Add(ctx, delta.Neg())
}

func parsePositiveSystemBalanceAmount(raw string) (decimal.Decimal, error) {
	amount, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil || !amount.IsPositive() || amount.GreaterThan(maxSystemAvailableBalanceAdjust) {
		return decimal.Zero, ErrSystemAvailableBalanceAmount
	}
	return amount, nil
}
