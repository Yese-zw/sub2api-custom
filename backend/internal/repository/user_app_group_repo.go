package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var _ service.UserAppGroupStore = (*userRepository)(nil)

// GetAppGroupID returns the desktop App group the user picked (0 = follow the
// system default). users.app_group_id is a raw-SQL column (migration 242).
func (r *userRepository) GetAppGroupID(ctx context.Context, userID int64) (int64, error) {
	if r.sql == nil {
		return 0, fmt.Errorf("sql executor is not configured")
	}
	rows, err := r.sql.QueryContext(ctx,
		`SELECT app_group_id FROM users WHERE id = $1 AND deleted_at IS NULL`, userID)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, service.ErrUserNotFound
	}
	var groupID sql.NullInt64
	if err := rows.Scan(&groupID); err != nil {
		return 0, err
	}
	return groupID.Int64, rows.Err()
}

// SetAppGroupID stores the desktop App group for the user (0 clears it).
func (r *userRepository) SetAppGroupID(ctx context.Context, userID, groupID int64) error {
	if r.sql == nil {
		return fmt.Errorf("sql executor is not configured")
	}
	result, err := r.sql.ExecContext(ctx,
		`UPDATE users SET app_group_id = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		userID, groupID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrUserNotFound
	}
	return nil
}
