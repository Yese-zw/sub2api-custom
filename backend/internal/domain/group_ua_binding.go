package domain

// GroupUABindingRule 把匹配的（黑名单）User-Agent 固定到分组内的指定检测账号。
//
// Patterns 中任意一条匹配请求 User-Agent 即命中该规则（不区分大小写）：
//   - 不含 * 时按子串匹配（如 "claude-cli" 匹配 "claude-cli/2.0.1 (external, cli)"）；
//   - 含 * 时按通配符匹配整个 User-Agent（如 "codex_cli_rs/*"）。
//
// 命中后该请求只能调度到 AccountIDs 中的账号，不会回退到分组内其他账号；
// 未命中任何规则的正常请求会排除所有规则中的 AccountIDs（检测账号）。
type GroupUABindingRule struct {
	Name       string   `json:"name,omitempty"`
	Patterns   []string `json:"patterns"`
	AccountIDs []int64  `json:"account_ids"`
}
