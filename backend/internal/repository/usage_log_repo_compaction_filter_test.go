package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppendNativeCompactionV2Filters(t *testing.T) {
	enabled, disabled := true, false
	tests := []struct {
		name          string
		filter        *bool
		alias         string
		args          []any
		wantCondition string
		wantArgs      []any
	}{
		{
			name: "omitted with no arguments",
		},
		{
			name:     "omitted preserves arguments",
			alias:    "ul",
			args:     []any{int64(42), "model"},
			wantArgs: []any{int64(42), "model"},
		},
		{
			name:          "enabled",
			filter:        &enabled,
			wantCondition: "native_compaction_v2 = $1",
			wantArgs:      []any{true},
		},
		{
			name:          "disabled is an explicit filter",
			filter:        &disabled,
			wantCondition: "native_compaction_v2 = $1",
			wantArgs:      []any{false},
		},
		{
			name:          "enabled with alias and preceding arguments",
			filter:        &enabled,
			alias:         "ul",
			args:          []any{int64(42), "model"},
			wantCondition: "ul.native_compaction_v2 = $3",
			wantArgs:      []any{int64(42), "model", true},
		},
		{
			name:          "disabled with alias and preceding arguments",
			filter:        &disabled,
			alias:         "ul",
			args:          []any{int64(42), "model"},
			wantCondition: "ul.native_compaction_v2 = $3",
			wantArgs:      []any{int64(42), "model", false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, withExistingCondition := range []bool{false, true} {
				var conditions, wantConditions []string
				if withExistingCondition {
					conditions = []string{"created_at IS NOT NULL"}
					wantConditions = []string{"created_at IS NOT NULL"}
				}
				if tt.wantCondition != "" {
					wantConditions = append(wantConditions, tt.wantCondition)
				}
				gotConditions, gotArgs := appendNativeCompactionV2WhereCondition(conditions, tt.args, tt.filter, tt.alias)
				require.Equal(t, wantConditions, gotConditions)
				require.Equal(t, tt.wantArgs, gotArgs)
			}

			query := "SELECT COUNT(*) FROM usage_logs ul WHERE created_at IS NOT NULL"
			wantQuery := query
			if tt.wantCondition != "" {
				wantQuery += " AND " + tt.wantCondition
			}
			gotQuery, gotArgs := appendNativeCompactionV2QueryFilter(query, tt.args, tt.filter, tt.alias)
			require.Equal(t, wantQuery, gotQuery)
			require.Equal(t, tt.wantArgs, gotArgs)
		})
	}
}
