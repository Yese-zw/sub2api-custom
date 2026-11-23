package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestMatchUABindingPattern(t *testing.T) {
	cases := []struct {
		pattern, ua string
		want        bool
	}{
		{"claude-cli", "claude-cli/2.0.1 (external, cli)", true},
		{"Claude-CLI", "claude-cli/2.0.1", true},
		{"codex_cli_rs/*", "codex_cli_rs/0.50.0 (Mac OS 15.0)", true},
		{"codex_cli_rs/*", "my codex_cli_rs/0.50.0", false},
		{"*codex*", "my codex_cli_rs/0.50.0", true},
		{"curl/*.0", "curl/8.4.0", true},
		{"curl/*.0", "curl/8.4.1", false},
		{"python-requests", "curl/8.4.0", false},
		{"  ", "anything", false},
	}
	for _, c := range cases {
		require.Equal(t, c.want, matchUABindingPattern(c.pattern, toLowerForTest(c.ua)), "pattern=%q ua=%q", c.pattern, c.ua)
	}
}

func toLowerForTest(s string) string {
	b := []byte(s)
	for i, ch := range b {
		if ch >= 'A' && ch <= 'Z' {
			b[i] = ch + 32
		}
	}
	return string(b)
}

func TestGroupMatchUABindingAccountIDs(t *testing.T) {
	g := &Group{
		UABindingEnabled: true,
		UABindingRules: []GroupUABindingRule{
			{Patterns: []string{"claude-cli"}, AccountIDs: []int64{1, 2}},
			{Patterns: []string{"codex*"}, AccountIDs: []int64{3}},
		},
	}
	ids, ok := g.MatchUABindingAccountIDs("claude-cli/2.0")
	require.True(t, ok)
	require.Equal(t, []int64{1, 2}, ids)

	ids, ok = g.MatchUABindingAccountIDs("codex_cli_rs/1.0")
	require.True(t, ok)
	require.Equal(t, []int64{3}, ids)

	_, ok = g.MatchUABindingAccountIDs("curl/8")
	require.False(t, ok)

	g.UABindingEnabled = false
	_, ok = g.MatchUABindingAccountIDs("claude-cli/2.0")
	require.False(t, ok)

	var nilGroup *Group
	_, ok = nilGroup.MatchUABindingAccountIDs("claude-cli")
	require.False(t, ok)
}

func TestNormalizeGroupUABindingRules(t *testing.T) {
	out, err := NormalizeGroupUABindingRules([]GroupUABindingRule{
		{Name: " cli ", Patterns: []string{" claude-cli ", "CLAUDE-CLI", "", "*"}, AccountIDs: []int64{2, 2, 0, -1, 3}},
		{Patterns: nil, AccountIDs: nil},
	})
	require.NoError(t, err)
	require.Equal(t, []GroupUABindingRule{{Name: "cli", Patterns: []string{"claude-cli"}, AccountIDs: []int64{2, 3}}}, out)

	_, err = NormalizeGroupUABindingRules([]GroupUABindingRule{{Patterns: []string{"x"}}})
	require.Error(t, err)
	_, err = NormalizeGroupUABindingRules([]GroupUABindingRule{{AccountIDs: []int64{1}}})
	require.Error(t, err)
}

func TestFilterUABindingAccountsStrict(t *testing.T) {
	groupID := int64(7)
	group := &Group{
		ID:               groupID,
		Platform:         PlatformAnthropic,
		Status:           StatusActive,
		Hydrated:         true,
		UABindingEnabled: true,
		UABindingRules:   []GroupUABindingRule{{Patterns: []string{"claude-cli"}, AccountIDs: []int64{2, 99}}},
	}
	accounts := []Account{{ID: 1}, {ID: 2}, {ID: 3}}
	lookup := func(context.Context, int64) (*Group, error) { return group, nil }

	base := context.WithValue(context.Background(), ctxkey.Group, group)

	// 命中：只保留规则内账号
	ctx := context.WithValue(base, ctxkey.UserAgent, "claude-cli/2.0.1")
	got := filterUABindingAccounts(ctx, &groupID, accounts, lookup)
	require.Len(t, got, 1)
	require.Equal(t, int64(2), got[0].ID)
	require.True(t, uaBindingBlocksSticky(ctx, &groupID, 1, lookup))
	require.False(t, uaBindingBlocksSticky(ctx, &groupID, 2, lookup))

	// 命中但规则账号都不在候选中：严格返回空，不回退
	group2 := *group
	group2.UABindingRules = []GroupUABindingRule{{Patterns: []string{"claude-cli"}, AccountIDs: []int64{99}}}
	ctx2 := context.WithValue(context.WithValue(context.Background(), ctxkey.Group, &group2), ctxkey.UserAgent, "claude-cli/2.0.1")
	require.Empty(t, filterUABindingAccounts(ctx2, &groupID, accounts, lookup))

	// 未命中（正常请求）：排除检测账号 2、99
	ctx3 := context.WithValue(base, ctxkey.UserAgent, "curl/8.0")
	got = filterUABindingAccounts(ctx3, &groupID, accounts, lookup)
	require.Equal(t, []int64{1, 3}, accountIDsForTest(got))
	require.False(t, uaBindingBlocksSticky(ctx3, &groupID, 1, lookup))
	require.True(t, uaBindingBlocksSticky(ctx3, &groupID, 2, lookup))

	// 没有 UA：按正常请求处理，同样排除检测账号
	require.Equal(t, []int64{1, 3}, accountIDsForTest(filterUABindingAccounts(base, &groupID, accounts, lookup)))

	// 分组未开启：不限制
	disabled := *group
	disabled.UABindingEnabled = false
	ctx4 := context.WithValue(context.WithValue(context.Background(), ctxkey.Group, &disabled), ctxkey.UserAgent, "curl/8.0")
	require.Len(t, filterUABindingAccounts(ctx4, &groupID, accounts, lookup), 3)
	require.False(t, uaBindingBlocksSticky(ctx4, &groupID, 2, lookup))
}

func TestUABindingMultipleRulesExcludeAllDetectionAccounts(t *testing.T) {
	group := &Group{
		UABindingEnabled: true,
		UABindingRules: []GroupUABindingRule{
			{Patterns: []string{"scanner-a"}, AccountIDs: []int64{10}},
			{Patterns: []string{"scanner-b"}, AccountIDs: []int64{20}},
		},
	}
	d := group.uaBindingDecision("scanner-a/1.0")
	require.True(t, d.permits(10))
	require.False(t, d.permits(20))
	require.False(t, d.permits(1))

	d = group.uaBindingDecision("claude-cli/2.0")
	require.False(t, d.matched())
	require.False(t, d.permits(10))
	require.False(t, d.permits(20))
	require.True(t, d.permits(1))
}

func accountIDsForTest(accounts []Account) []int64 {
	out := make([]int64, 0, len(accounts))
	for _, a := range accounts {
		out = append(out, a.ID)
	}
	return out
}
