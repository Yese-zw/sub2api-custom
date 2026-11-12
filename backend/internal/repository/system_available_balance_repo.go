package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
)

type systemAvailableBalanceRepository struct {
	db *sql.DB
}

func NewSystemAvailableBalanceRepository(db *sql.DB) service.SystemAvailableBalanceRepository {
	return &systemAvailableBalanceRepository{db: db}
}

func (r *systemAvailableBalanceRepository) Get(ctx context.Context) (decimal.Decimal, error) {
	if r == nil || r.db == nil {
		return decimal.Zero, errors.New("system available balance database is unavailable")
	}
	var raw string
	err := r.db.QueryRowContext(ctx, `
		SELECT balance::text
		FROM system_available_balance
		WHERE id = 1
	`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return decimal.Zero, nil
	}
	if err != nil {
		return decimal.Zero, fmt.Errorf("get system available balance: %w", err)
	}
	amount, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse system available balance: %w", err)
	}
	return amount, nil
}

func (r *systemAvailableBalanceRepository) Add(ctx context.Context, delta decimal.Decimal) (decimal.Decimal, error) {
	if r == nil || r.db == nil {
		return decimal.Zero, errors.New("system available balance database is unavailable")
	}
	var raw string
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO system_available_balance (id, balance, updated_at)
		VALUES (1, $1::numeric, NOW())
		ON CONFLICT (id) DO UPDATE
		SET balance = system_available_balance.balance + EXCLUDED.balance,
		    updated_at = NOW()
		RETURNING balance::text
	`, delta.String()).Scan(&raw)
	if err != nil {
		return decimal.Zero, fmt.Errorf("adjust system available balance: %w", err)
	}
	amount, err := decimal.NewFromString(raw)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse system available balance: %w", err)
	}
	return amount, nil
}
