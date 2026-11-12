//go:build unit

package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration256TracksSystemAvailableBalanceIndependently(t *testing.T) {
	content, err := FS.ReadFile("256_system_available_balance.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS system_available_balance")
	require.Contains(t, sql, "CHECK (id = 1)")
	require.Contains(t, sql, "ON CONFLICT (id) DO NOTHING")
	require.Contains(t, sql, "COALESCE(NEW.account_stats_cost, NEW.total_cost, 0) * COALESCE(NEW.account_rate_multiplier, 1)")
	require.Contains(t, sql, "balance = balance - cost")
	require.Contains(t, sql, "AFTER INSERT ON usage_logs")
	require.NotContains(t, sql, "AFTER DELETE")
	require.NotContains(t, sql, "AFTER UPDATE")
}
