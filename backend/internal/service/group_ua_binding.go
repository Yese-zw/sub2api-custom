package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

const (
	maxGroupUABindingRules       = 50
	maxGroupUABindingPatterns    = 50
	maxGroupUABindingPatternLen  = 256
	maxGroupUABindingAccountsPer = 500
	maxGroupUABindingRuleNameLen = 100
)

// MatchUABindingAccountIDs 返回 User-Agent 命中的第一条规则的账号集合。
// 未开启、UA 为空或没有规则命中时返回 nil, false。
func (g *Group) MatchUABindingAccountIDs(userAgent string) ([]int64, bool) {
	if g == nil || !g.UABindingEnabled || len(g.UABindingRules) == 0 {
		return nil, false
	}
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return nil, false
	}
	for _, rule := range g.UABindingRules {
		if len(rule.AccountIDs) == 0 {
			continue
		}
		for _, pattern := range rule.Patterns {
			if matchUABindingPattern(pattern, ua) {
				return rule.AccountIDs, true
			}
		}
	}
	return nil, false
}

// matchUABindingPattern: 不含 * 时子串匹配；含 * 时通配符匹配整个 UA。均不区分大小写。
// lowerUA 必须已转小写。
func matchUABindingPattern(pattern, lowerUA string) bool {
	p := strings.ToLower(strings.TrimSpace(pattern))
	if p == "" {
		return false
	}
	if !strings.Contains(p, "*") {
		return strings.Contains(lowerUA, p)
	}
	return globMatchUA(p, lowerUA)
}

// globMatchUA 只支持 * 通配（匹配任意字符，含 /），其余字符按字面匹配。
func globMatchUA(pattern, s string) bool {
	parts := strings.Split(pattern, "*")
	if !strings.HasPrefix(s, parts[0]) {
		return false
	}
	s = s[len(parts[0]):]
	last := len(parts) - 1
	for i := 1; i < last; i++ {
		idx := strings.Index(s, parts[i])
		if idx < 0 {
			return false
		}
		s = s[idx+len(parts[i]):]
	}
	return strings.HasSuffix(s, parts[last])
}

// NormalizeGroupUABindingRules 清理并校验 UA 绑定规则：去空白、去重、丢弃空规则。
func NormalizeGroupUABindingRules(rules []GroupUABindingRule) ([]GroupUABindingRule, error) {
	if len(rules) > maxGroupUABindingRules {
		return nil, fmt.Errorf("ua_binding_rules: at most %d rules", maxGroupUABindingRules)
	}
	out := make([]GroupUABindingRule, 0, len(rules))
	for i, rule := range rules {
		name := strings.TrimSpace(rule.Name)
		if len([]rune(name)) > maxGroupUABindingRuleNameLen {
			return nil, fmt.Errorf("ua_binding_rules[%d]: name too long", i)
		}
		patterns := make([]string, 0, len(rule.Patterns))
		seenPattern := make(map[string]struct{}, len(rule.Patterns))
		for _, p := range rule.Patterns {
			p = strings.TrimSpace(p)
			if p == "" || strings.Trim(p, "*") == "" {
				continue
			}
			if len(p) > maxGroupUABindingPatternLen {
				return nil, fmt.Errorf("ua_binding_rules[%d]: pattern too long", i)
			}
			key := strings.ToLower(p)
			if _, dup := seenPattern[key]; dup {
				continue
			}
			seenPattern[key] = struct{}{}
			patterns = append(patterns, p)
		}
		ids := make([]int64, 0, len(rule.AccountIDs))
		seenID := make(map[int64]struct{}, len(rule.AccountIDs))
		for _, id := range rule.AccountIDs {
			if id <= 0 {
				continue
			}
			if _, dup := seenID[id]; dup {
				continue
			}
			seenID[id] = struct{}{}
			ids = append(ids, id)
		}
		if len(patterns) == 0 && len(ids) == 0 {
			continue
		}
		if len(patterns) == 0 {
			return nil, fmt.Errorf("ua_binding_rules[%d]: at least one User-Agent pattern is required", i)
		}
		if len(ids) == 0 {
			return nil, fmt.Errorf("ua_binding_rules[%d]: at least one account is required", i)
		}
		if len(patterns) > maxGroupUABindingPatterns {
			return nil, fmt.Errorf("ua_binding_rules[%d]: at most %d patterns", i, maxGroupUABindingPatterns)
		}
		if len(ids) > maxGroupUABindingAccountsPer {
			return nil, fmt.Errorf("ua_binding_rules[%d]: at most %d accounts", i, maxGroupUABindingAccountsPer)
		}
		out = append(out, GroupUABindingRule{Name: name, Patterns: patterns, AccountIDs: ids})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// 网关接入
// ---------------------------------------------------------------------------

// RequestUserAgentFromContext 返回鉴权中间件写入的客户端 User-Agent。
func RequestUserAgentFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	ua, _ := ctx.Value(ctxkey.UserAgent).(string)
	return ua
}

// uaBindingGroupLookup 按 ID 取分组（不需要账号计数）。
type uaBindingGroupLookup func(ctx context.Context, groupID int64) (*Group, error)

// 上下文里没有对应分组（降级分组、composite 目标分组等）时回源查询，结果在进程内短暂缓存，
// 避免一次请求内多次选号重复查库。
const uaBindingGroupCacheTTL = 5 * time.Second

type uaBindingCachedGroup struct {
	group   *Group
	expires time.Time
}

var uaBindingGroupCache sync.Map // groupID -> uaBindingCachedGroup

func lookupUABindingGroup(ctx context.Context, groupID int64, lookup uaBindingGroupLookup) (*Group, error) {
	now := time.Now()
	if v, ok := uaBindingGroupCache.Load(groupID); ok {
		if c := v.(uaBindingCachedGroup); now.Before(c.expires) {
			return c.group, nil
		}
	}
	g, err := lookup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	uaBindingGroupCache.Store(groupID, uaBindingCachedGroup{group: g, expires: now.Add(uaBindingGroupCacheTTL)})
	return g, nil
}

// uaBindingDecision 是一次请求在某分组下的 UA 绑定决策。
//   - allowed 非 nil：请求 UA 命中规则，只能调度到 allowed 内的检测账号；
//   - excluded 非 nil：请求 UA 未命中任何规则（含没有 UA），必须排除所有规则绑定的检测账号；
//   - 两者都为 nil：分组未开启或无规则，不限制。
type uaBindingDecision struct {
	allowed  map[int64]struct{}
	excluded map[int64]struct{}
}

func (d uaBindingDecision) matched() bool { return d.allowed != nil }

func (d uaBindingDecision) permits(accountID int64) bool {
	if d.allowed != nil {
		_, ok := d.allowed[accountID]
		return ok
	}
	if d.excluded != nil {
		_, ok := d.excluded[accountID]
		return !ok
	}
	return true
}

// uaBindingDecision 计算 userAgent 在本分组下的绑定决策。
func (g *Group) uaBindingDecision(userAgent string) uaBindingDecision {
	if g == nil || !g.UABindingEnabled || len(g.UABindingRules) == 0 {
		return uaBindingDecision{}
	}
	if ids, ok := g.MatchUABindingAccountIDs(userAgent); ok {
		set := make(map[int64]struct{}, len(ids))
		for _, id := range ids {
			set[id] = struct{}{}
		}
		return uaBindingDecision{allowed: set}
	}
	excluded := make(map[int64]struct{})
	for _, rule := range g.UABindingRules {
		if len(rule.Patterns) == 0 {
			continue
		}
		for _, id := range rule.AccountIDs {
			excluded[id] = struct{}{}
		}
	}
	if len(excluded) == 0 {
		return uaBindingDecision{}
	}
	return uaBindingDecision{excluded: excluded}
}

// uaBindingDecisionFor 返回本次请求在 groupID 下的 UA 绑定决策。
// 分组优先取请求上下文中的鉴权分组，ID 不一致（降级分组、composite 目标等）时回源查询。
func uaBindingDecisionFor(ctx context.Context, groupID *int64, lookup uaBindingGroupLookup) uaBindingDecision {
	if groupID == nil || *groupID <= 0 {
		return uaBindingDecision{}
	}
	var group *Group
	if ctxGroup, ok := ctx.Value(ctxkey.Group).(*Group); ok && IsGroupContextValid(ctxGroup) && ctxGroup.ID == *groupID {
		group = ctxGroup
	} else if lookup != nil {
		g, err := lookupUABindingGroup(ctx, *groupID, lookup)
		if err != nil {
			slog.Warn("ua_binding_group_lookup_failed", "group_id", *groupID, "error", err)
			return uaBindingDecision{}
		}
		group = g
	}
	return group.uaBindingDecision(RequestUserAgentFromContext(ctx))
}

// filterUABindingAccounts 按 UA 绑定决策过滤候选账号（严格，不回退）：
// 命中规则的 UA 只保留规则内检测账号；其余请求去掉所有检测账号。返回新切片，不修改入参。
func filterUABindingAccounts(ctx context.Context, groupID *int64, accounts []Account, lookup uaBindingGroupLookup) []Account {
	if len(accounts) == 0 {
		return accounts
	}
	d := uaBindingDecisionFor(ctx, groupID, lookup)
	if d.allowed == nil && d.excluded == nil {
		return accounts
	}
	filtered := make([]Account, 0, len(accounts))
	for i := range accounts {
		if d.permits(accounts[i].ID) {
			filtered = append(filtered, accounts[i])
		}
	}
	return filtered
}

// uaBindingBlocksSticky 报告粘性会话绑定的账号本次请求是否不可使用。
func uaBindingBlocksSticky(ctx context.Context, groupID *int64, accountID int64, lookup uaBindingGroupLookup) bool {
	if accountID <= 0 {
		return false
	}
	return !uaBindingDecisionFor(ctx, groupID, lookup).permits(accountID)
}

// ---- 各网关服务的接入 ----

func (s *GatewayService) uaBindingLookup(ctx context.Context, groupID int64) (*Group, error) {
	return s.resolveGroupByID(ctx, groupID)
}

// stickyAccountBlocked 报告粘性会话绑定的账号本次是否不可复用（调度池生效或 UA 绑定）。
func (s *GatewayService) stickyAccountBlocked(ctx context.Context, groupID *int64, accountID int64) bool {
	return schedulingCenterBlocksSticky(ctx, groupID, accountID) || uaBindingBlocksSticky(ctx, groupID, accountID, s.uaBindingLookup)
}

func (s *OpenAIGatewayService) uaBindingLookup(ctx context.Context, groupID int64) (*Group, error) {
	if s.schedulerSnapshot == nil {
		return nil, nil
	}
	return s.schedulerSnapshot.GetGroupByIDLite(ctx, groupID)
}

func (s *OpenAIGatewayService) stickyAccountBlocked(ctx context.Context, groupID *int64, accountID int64) bool {
	return schedulingCenterBlocksSticky(ctx, groupID, accountID) || uaBindingBlocksSticky(ctx, groupID, accountID, s.uaBindingLookup)
}

func (s *GeminiMessagesCompatService) uaBindingLookup(ctx context.Context, groupID int64) (*Group, error) {
	if s.groupRepo == nil {
		return nil, nil
	}
	return s.groupRepo.GetByIDLite(ctx, groupID)
}

func (s *GeminiMessagesCompatService) stickyAccountBlocked(ctx context.Context, groupID *int64, accountID int64) bool {
	return schedulingCenterBlocksSticky(ctx, groupID, accountID) || uaBindingBlocksSticky(ctx, groupID, accountID, s.uaBindingLookup)
}
