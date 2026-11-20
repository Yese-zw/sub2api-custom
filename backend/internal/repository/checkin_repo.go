package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const checkinSelectColumns = `
SELECT id,
       user_id,
       to_char(checkin_date, 'YYYY-MM-DD'),
       base_reward::double precision,
       streak_bonus::double precision,
       reward_amount::double precision,
       streak_days,
       ip_address,
       created_at
FROM user_checkins`

type checkinRepository struct {
	client *dbent.Client
}

// NewCheckinRepository creates the check-in repository.
func NewCheckinRepository(client *dbent.Client) service.CheckinRepository {
	return &checkinRepository{client: client}
}

func scanCheckinRows(rows *sql.Rows) ([]service.CheckinRecord, error) {
	defer func() { _ = rows.Close() }()
	out := make([]service.CheckinRecord, 0)
	for rows.Next() {
		var r service.CheckinRecord
		if err := rows.Scan(
			&r.ID,
			&r.UserID,
			&r.CheckinDate,
			&r.BaseReward,
			&r.StreakBonus,
			&r.RewardAmount,
			&r.StreakDays,
			&r.IPAddress,
			&r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (r *checkinRepository) GetLatest(ctx context.Context, userID int64) (*service.CheckinRecord, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, checkinSelectColumns+`
WHERE user_id = $1
ORDER BY checkin_date DESC
LIMIT 1`, userID)
	if err != nil {
		return nil, fmt.Errorf("query latest checkin: %w", err)
	}
	records, err := scanCheckinRows(rows)
	if err != nil {
		return nil, fmt.Errorf("scan latest checkin: %w", err)
	}
	if len(records) == 0 {
		return nil, nil
	}
	return &records[0], nil
}

func (r *checkinRepository) ListByDateRange(ctx context.Context, userID int64, from, to string) ([]service.CheckinRecord, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, checkinSelectColumns+`
WHERE user_id = $1
  AND checkin_date >= $2::date
  AND checkin_date <= $3::date
ORDER BY checkin_date ASC`, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query checkins by range: %w", err)
	}
	records, err := scanCheckinRows(rows)
	if err != nil {
		return nil, fmt.Errorf("scan checkins by range: %w", err)
	}
	return records, nil
}

func (r *checkinRepository) ListPaged(ctx context.Context, userID int64, page, pageSize int) ([]service.CheckinRecord, int64, error) {
	client := clientFromContext(ctx, r.client)
	total, err := scanInt64(ctx, client, `SELECT COUNT(*) FROM user_checkins WHERE user_id = $1`, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("count checkins: %w", err)
	}
	if total == 0 {
		return []service.CheckinRecord{}, 0, nil
	}
	rows, err := client.QueryContext(ctx, checkinSelectColumns+`
WHERE user_id = $1
ORDER BY checkin_date DESC
LIMIT $2 OFFSET $3`, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("query checkin history: %w", err)
	}
	records, err := scanCheckinRows(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("scan checkin history: %w", err)
	}
	return records, total, nil
}

func (r *checkinRepository) GetTotals(ctx context.Context, userID int64) (*service.CheckinUserTotals, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, `
SELECT COUNT(*)::integer, COALESCE(SUM(reward_amount), 0)::double precision
FROM user_checkins
WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("query checkin totals: %w", err)
	}
	defer func() { _ = rows.Close() }()
	totals := &service.CheckinUserTotals{}
	if rows.Next() {
		if err := rows.Scan(&totals.TotalDays, &totals.TotalReward); err != nil {
			return nil, fmt.Errorf("scan checkin totals: %w", err)
		}
	}
	return totals, rows.Err()
}

func (r *checkinRepository) CreateWithReward(ctx context.Context, record *service.CheckinRecord, ledgerCode string) (float64, error) {
	var newBalance float64
	err := r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		// The unique (user_id, checkin_date) key decides concurrent races:
		// ON CONFLICT DO NOTHING returns no row for the loser.
		rows, err := txClient.QueryContext(txCtx, `
INSERT INTO user_checkins (
    user_id, checkin_date, base_reward, streak_bonus, reward_amount, streak_days, ip_address, created_at
)
VALUES ($1, $2::date, $3, $4, $5, $6, $7, NOW())
ON CONFLICT (user_id, checkin_date) DO NOTHING
RETURNING id, created_at`,
			record.UserID,
			record.CheckinDate,
			record.BaseReward,
			record.StreakBonus,
			record.RewardAmount,
			record.StreakDays,
			record.IPAddress,
		)
		if err != nil {
			return fmt.Errorf("insert checkin: %w", err)
		}
		inserted := rows.Next()
		if inserted {
			if err := rows.Scan(&record.ID, &record.CreatedAt); err != nil {
				_ = rows.Close()
				return fmt.Errorf("scan inserted checkin: %w", err)
			}
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if !inserted {
			return service.ErrCheckinAlreadyDone
		}

		if record.RewardAmount > 0 {
			// Check-in rewards are gifts, not recharges: total_recharged is untouched.
			affected, err := txClient.User.Update().
				Where(user.IDEQ(record.UserID)).
				AddBalance(record.RewardAmount).
				Save(txCtx)
			if err != nil {
				return fmt.Errorf("credit checkin reward: %w", err)
			}
			if affected == 0 {
				return service.ErrUserNotFound
			}

			now := time.Now()
			if _, err := txClient.RedeemCode.Create().
				SetCode(ledgerCode).
				SetType(service.AdjustmentTypeCheckin).
				SetValue(record.RewardAmount).
				SetStatus(service.StatusUsed).
				SetUsedBy(record.UserID).
				SetUsedAt(now).
				SetNotes(fmt.Sprintf("checkin %s streak %d", record.CheckinDate, record.StreakDays)).
				Save(txCtx); err != nil {
				return fmt.Errorf("insert checkin ledger: %w", err)
			}
		}

		newBalance, err = queryUserBalance(txCtx, txClient, record.UserID)
		return err
	})
	if err != nil {
		return 0, err
	}
	return newBalance, nil
}

func (r *checkinRepository) withTx(ctx context.Context, fn func(txCtx context.Context, txClient *dbent.Client) error) error {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return fn(ctx, tx.Client())
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin checkin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	if err := fn(txCtx, tx.Client()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit checkin transaction: %w", err)
	}
	return nil
}
