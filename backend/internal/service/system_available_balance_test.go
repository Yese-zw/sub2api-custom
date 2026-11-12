package service

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type systemBalanceRepoStub struct {
	balance decimal.Decimal
	deltas  []decimal.Decimal
}

func (s *systemBalanceRepoStub) Get(context.Context) (decimal.Decimal, error) {
	return s.balance, nil
}

func (s *systemBalanceRepoStub) Add(_ context.Context, delta decimal.Decimal) (decimal.Decimal, error) {
	s.deltas = append(s.deltas, delta)
	s.balance = s.balance.Add(delta)
	return s.balance, nil
}

func TestSystemAvailableBalanceRechargeAndDeduct(t *testing.T) {
	repo := &systemBalanceRepoStub{balance: decimal.RequireFromString("10.5")}
	svc := NewSystemAvailableBalanceService(repo)

	got, err := svc.Recharge(context.Background(), "2.25")
	require.NoError(t, err)
	require.True(t, got.Equal(decimal.RequireFromString("12.75")))

	got, err = svc.Deduct(context.Background(), "20")
	require.NoError(t, err)
	require.True(t, got.Equal(decimal.RequireFromString("-7.25")))
	require.Equal(t, []decimal.Decimal{
		decimal.RequireFromString("2.25"),
		decimal.RequireFromString("-20"),
	}, repo.deltas)
}

func TestSystemAvailableBalanceRejectsNonPositiveAmount(t *testing.T) {
	svc := NewSystemAvailableBalanceService(&systemBalanceRepoStub{})
	for _, amount := range []string{"", "0", "-1", "abc", "1000000000001"} {
		_, err := svc.Recharge(context.Background(), amount)
		require.ErrorIs(t, err, ErrSystemAvailableBalanceAmount)
		_, err = svc.Deduct(context.Background(), amount)
		require.ErrorIs(t, err, ErrSystemAvailableBalanceAmount)
	}
}
