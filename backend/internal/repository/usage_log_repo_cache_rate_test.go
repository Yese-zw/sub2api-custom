package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGetAccountCacheRateBatch(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := newUsageLogRepositoryWithSQL(nil, db)
	start := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`(?s)COUNT\(\*\).*SUM\(input_tokens\).*SUM\(cache_creation_tokens\).*SUM\(cache_read_tokens\).*account_id = ANY\(\$1\) AND created_at >= \$2`).
		WithArgs(sqlmock.AnyArg(), start).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "requests", "input_tokens", "cache_creation_tokens", "cache_read_tokens"}).
			AddRow(int64(7), int64(4), int64(20), int64(10), int64(70)))

	result, err := repo.GetAccountCacheRateBatch(context.Background(), []int64{7, 8}, start)
	require.NoError(t, err)
	require.Equal(t, service.AccountCacheRateTotals{Requests: 4, InputTokens: 20, CacheCreationTokens: 10, CacheReadTokens: 70}, result[7])
	_, ok := result[8]
	require.False(t, ok)
	require.NoError(t, mock.ExpectationsWereMet())

	empty, err := repo.GetAccountCacheRateBatch(context.Background(), nil, start)
	require.NoError(t, err)
	require.Empty(t, empty)
}
