// Package operatoraccess 定义「运营(operator)」身份的后台权限模型。
//
// 权限以「标签页.操作」命名（如 users.view、accounts.delete）。接口是否放行由
// 路由前缀 + HTTP 方法推导出的操作决定，个别高危/特殊接口通过 overrides 单独登记；
// 未归属任何标签页的接口（系统设置、备份、插件、数据导入导出等）一律仅管理员可用。
package operatoraccess

import (
	"slices"
	"strings"
)

const adminPrefix = "/api/v1/admin"

// Action 是某个标签页下可授予的一项操作。
type Action struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Tab 对应后台侧边栏中的一个标签页。
type Tab struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Actions []Action `json:"actions"`
}

type tabRule struct {
	Tab
	// Prefixes 是归属该标签页的路由前缀（相对 /api/v1/admin）。
	Prefixes []string
	// ByMethod 描述该标签页下各 HTTP 方法默认对应的操作 ID（不含标签页前缀）。
	ByMethod map[string]string
}

var (
	crud = map[string]string{"GET": "view", "POST": "create", "PUT": "edit", "PATCH": "edit", "DELETE": "delete"}
	rw   = map[string]string{"GET": "view", "POST": "edit", "PUT": "edit", "PATCH": "edit", "DELETE": "edit"}
	view = map[string]string{"GET": "view"}
)

var (
	actView   = Action{"view", "查看"}
	actCreate = Action{"create", "新增"}
	actEdit   = Action{"edit", "编辑"}
	actDelete = Action{"delete", "删除"}
)

// adminOnly 标记永远不下放给运营的接口。
const adminOnly = "-"

var tabs = []tabRule{
	{Tab{"dashboard", "仪表盘", []Action{actView}}, []string{"/dashboard"}, view},
	{Tab{"profit", "利润", []Action{actView}}, nil, view},
	{Tab{"ops", "运维监控", []Action{actView, actEdit}}, []string{"/ops"}, rw},
	{Tab{"users", "用户管理", []Action{actView, actCreate, actEdit, actDelete, {"balance", "调整余额"}}}, []string{"/users", "/user-attributes", "/api-keys"}, crud},
	{Tab{"groups", "分组管理", []Action{actView, actCreate, actEdit, actDelete}}, []string{"/groups"}, crud},
	{Tab{"channels", "渠道定价", []Action{actView, actCreate, actEdit, actDelete}}, []string{"/channels"}, crud},
	{Tab{"channel_monitor", "渠道监控", []Action{actView, actCreate, actEdit, actDelete}}, []string{"/channel-monitors", "/channel-monitor-templates", "/channel-monitor-v2"}, crud},
	{Tab{"subscriptions", "订阅管理", []Action{actView, {"assign", "分配订阅"}, {"manage", "延期/撤销/重置"}}}, []string{"/subscriptions"}, map[string]string{"GET": "view", "POST": "manage", "PUT": "manage", "DELETE": "manage"}},
	{Tab{"accounts", "账号管理", []Action{actView, actCreate, actEdit, actDelete}}, []string{"/accounts", "/openai", "/gemini", "/antigravity", "/grok", "/cn-providers", "/scheduled-test-plans", "/error-passthrough-rules", "/tls-fingerprint-profiles"}, map[string]string{"GET": "view", "POST": "edit", "PUT": "edit", "PATCH": "edit", "DELETE": "delete"}},
	{Tab{"announcements", "公告管理", []Action{actView, actCreate, actEdit, actDelete}}, []string{"/announcements"}, crud},
	{Tab{"proxies", "代理管理", []Action{actView, actCreate, actEdit, actDelete}}, []string{"/proxies"}, crud},
	{Tab{"risk_control", "风控中心", []Action{actView, actEdit}}, []string{"/risk-control"}, rw},
	{Tab{"prompt_audit", "提示词审计", []Action{actView, actEdit, actDelete}}, []string{"/prompt-audit"}, map[string]string{"GET": "view", "POST": "edit", "PUT": "edit", "DELETE": "delete"}},
	{Tab{"redeem", "卡密管理", []Action{actView, actCreate, actEdit, actDelete}}, []string{"/redeem-codes"}, crud},
	{Tab{"promo", "优惠码管理", []Action{actView, actCreate, actEdit, actDelete}}, []string{"/promo-codes"}, crud},
	{Tab{"affiliates", "邀请返利", []Action{actView, actEdit}}, []string{"/affiliates"}, rw},
	{Tab{"orders", "支付订单", []Action{actView, {"manage", "取消/重试"}, {"refund", "退款"}, {"plans", "管理支付套餐"}}}, []string{"/payment"}, map[string]string{"GET": "view", "POST": "manage", "PUT": "manage", "DELETE": "manage"}},
	{Tab{"usage", "使用记录", []Action{actView, actEdit}}, []string{"/usage"}, rw},
	{Tab{"audit_logs", "审计日志", []Action{actView}}, []string{"/audit-logs"}, view},
}

// overrides 以「METHOD 相对路径」精确登记例外；值为完整权限 ID 或 adminOnly。
var overrides = map[string]string{
	"GET /dashboard/profit-trend":                 "profit.view",
	"POST /dashboard/users-usage":                 "dashboard.view",
	"POST /dashboard/api-keys-usage":              "dashboard.view",
	"POST /dashboard/aggregation/backfill":        adminOnly,
	"POST /users/:id/balance":                     "users.balance",
	"POST /users/:id/auth-identities":             adminOnly,
	"GET /users/:id/operator-permissions":         adminOnly,
	"PUT /users/:id/operator-permissions":         adminOnly,
	"POST /users/batch-concurrency":               "users.edit",
	"POST /users/batch-limits":                    "users.edit",
	"POST /users/:id/replace-group":               "users.edit",
	"POST /users/:id/platform-quotas/reset":       "users.edit",
	"POST /user-attributes/batch":                 "users.view",
	"POST /groups/:id/composite-routes/preview":   "groups.view",
	"POST /groups/:id/duplicate":                  "groups.create",
	"POST /accounts":                              "accounts.create",
	"POST /accounts/batch":                        "accounts.create",
	"POST /accounts/:id/duplicate":                "accounts.create",
	"POST /accounts/import/codex-session":         "accounts.create",
	"POST /accounts/sync/crs":                     "accounts.create",
	"POST /accounts/usage/batch":                  "accounts.view",
	"POST /accounts/today-stats/batch":            "accounts.view",
	"POST /accounts/check-mixed-channel":          "accounts.view",
	"POST /accounts/sync/crs/preview":             "accounts.view",
	"POST /accounts/models/sync-upstream-preview": "accounts.view",
	"POST /accounts/batch-delete":                 "accounts.delete",
	"GET /accounts/data":                          adminOnly, // 导出含上游凭证原文
	"POST /accounts/data":                         adminOnly,
	"GET /proxies/data":                           adminOnly, // 导出含代理账号密码
	"POST /proxies/data":                          adminOnly,
	"POST /proxies/:id/test":                      "proxies.edit",
	"POST /proxies/:id/quality-check":             "proxies.edit",
	"POST /proxies/batch-delete":                  "proxies.delete",
	"POST /proxies/batch":                         "proxies.create",
	"GET /redeem-codes/export":                    "redeem.view",
	"POST /redeem-codes/batch-delete":             "redeem.delete",
	"POST /redeem-codes/batch-update":             "redeem.edit",
	"POST /redeem-codes/:id/expire":               "redeem.edit",
	"POST /channel-monitors/:id/duplicate":        "channel_monitor.create",
	"POST /channel-monitors/:id/run":              "channel_monitor.edit",
	"POST /channel-monitor-templates/:id/apply":   "channel_monitor.edit",
	"POST /prompt-audit/events/delete-preview":    "prompt_audit.view",
	"POST /prompt-audit/events/batch-delete":      "prompt_audit.delete",
	"POST /prompt-audit/events/delete-by-filter":  "prompt_audit.delete",
	"POST /audit-logs/clear":                      adminOnly,
	"POST /usage/cleanup-tasks":                   "usage.edit",
	"POST /usage/cleanup-tasks/:id/cancel":        "usage.edit",
	"POST /ops/system-logs/cleanup":               adminOnly,
	"POST /subscriptions/assign":                  "subscriptions.assign",
	"POST /subscriptions/bulk-assign":             "subscriptions.assign",
	"POST /payment/orders/:id/refund":             "orders.refund",
	"POST /payment/orders/:id/refund/query":       "orders.refund",
	"GET /payment/config":                         adminOnly,
	"PUT /payment/config":                         adminOnly,
	"GET /payment/providers":                      adminOnly,
	"POST /payment/providers":                     adminOnly,
	"PUT /payment/providers/:id":                  adminOnly,
	"DELETE /payment/providers/:id":               adminOnly,
	"POST /payment/plans":                         "orders.plans",
	"PUT /payment/plans/:id":                      "orders.plans",
	"DELETE /payment/plans/:id":                   "orders.plans",
	"GET /payment/plans":                          "orders.view",
}

// lookups 是页面下拉框等通用只读候选接口：持有任意一项权限即可调用。
var lookups = map[string]bool{
	"GET /groups/all":            true,
	"GET /proxies/all":           true,
	"GET /usage/search-users":    true,
	"GET /usage/search-api-keys": true,
}

// baseline 对所有运营开放（合规确认是进入后台的前置条件）。
var baseline = map[string]bool{
	"GET /compliance":         true,
	"POST /compliance/accept": true,
}

// Permission 是权限目录里的一项，供管理员配置界面渲染。
type Permission struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// TabCatalog 是配置界面展示的标签页及其可选操作。
type TabCatalog struct {
	Key         string       `json:"key"`
	Label       string       `json:"label"`
	Permissions []Permission `json:"permissions"`
}

// Catalog 返回完整权限目录。
func Catalog() []TabCatalog {
	out := make([]TabCatalog, 0, len(tabs))
	for _, t := range tabs {
		item := TabCatalog{Key: t.Key, Label: t.Label}
		for _, a := range t.Actions {
			item.Permissions = append(item.Permissions, Permission{ID: t.Key + "." + a.ID, Label: a.Label})
		}
		out = append(out, item)
	}
	return out
}

var validPermissions = func() map[string]struct{} {
	m := map[string]struct{}{}
	for _, t := range tabs {
		for _, a := range t.Actions {
			m[t.Key+"."+a.ID] = struct{}{}
		}
	}
	return m
}()

// Sanitize 去重并丢弃未知权限，返回稳定排序的结果；
// 授予任何写操作时自动补上该标签页的 view，保证标签页可见。
func Sanitize(perms []string) []string {
	set := map[string]struct{}{}
	for _, p := range perms {
		p = strings.TrimSpace(p)
		if _, ok := validPermissions[p]; !ok {
			continue
		}
		set[p] = struct{}{}
		tab := p[:strings.IndexByte(p, '.')]
		if _, hasView := validPermissions[tab+".view"]; hasView {
			set[tab+".view"] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	slices.Sort(out)
	return out
}

// Resolve 把请求映射为所需权限。delegable=false 表示仅管理员可调用。
func Resolve(method, fullPath string) (perm string, delegable bool) {
	rel := strings.TrimPrefix(fullPath, adminPrefix)
	if rel == fullPath || rel == "" {
		return "", false
	}
	key := method + " " + rel
	if v, ok := overrides[key]; ok {
		if v == adminOnly {
			return "", false
		}
		return v, true
	}
	for _, t := range tabs {
		for _, prefix := range t.Prefixes {
			if rel != prefix && !strings.HasPrefix(rel, prefix+"/") {
				continue
			}
			action, ok := t.ByMethod[method]
			if !ok {
				return "", false
			}
			return t.Key + "." + action, true
		}
	}
	return "", false
}

// Allowed 判断授予的权限集合是否放行该请求。
func Allowed(perms []string, method, fullPath string) bool {
	rel := strings.TrimPrefix(fullPath, adminPrefix)
	key := method + " " + rel
	if baseline[key] {
		return true
	}
	if lookups[key] {
		return len(perms) > 0
	}
	perm, ok := Resolve(method, fullPath)
	if !ok {
		return false
	}
	return slices.Contains(perms, perm)
}
