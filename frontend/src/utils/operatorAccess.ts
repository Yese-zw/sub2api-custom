/**
 * 运营身份的后台访问控制：路由 -> 所需查看权限。
 * 未登记的 /admin 路由（系统设置、插件等）仅管理员可进入。
 * 须与后端 internal/pkg/operatoraccess 的标签页保持一致。
 */
const ROUTE_PERMISSIONS: Array<[string, string]> = [
  ['/admin/dashboard', 'dashboard.view'],
  ['/admin/profit', 'profit.view'],
  ['/admin/ops', 'ops.view'],
  ['/admin/users', 'users.view'],
  ['/admin/groups', 'groups.view'],
  ['/admin/channels/monitor', 'channel_monitor.view'],
  ['/admin/channels', 'channels.view'],
  ['/admin/subscriptions', 'subscriptions.view'],
  ['/admin/accounts', 'accounts.view'],
  ['/admin/announcements', 'announcements.view'],
  ['/admin/proxies', 'proxies.view'],
  ['/admin/risk-control', 'risk_control.view'],
  ['/admin/prompt-audit', 'prompt_audit.view'],
  ['/admin/redeem', 'redeem.view'],
  ['/admin/promo-codes', 'promo.view'],
  ['/admin/affiliates', 'affiliates.view'],
  ['/admin/orders', 'orders.view'],
  ['/admin/usage', 'usage.view'],
  ['/admin/audit-logs', 'audit_logs.view']
]

export function requiredPermissionForPath(path: string): string | null {
  for (const [prefix, perm] of ROUTE_PERMISSIONS) {
    if (path === prefix || path.startsWith(prefix + '/')) return perm
  }
  return null
}

export function canOperatorAccessPath(path: string, perms: string[]): boolean {
  const perm = requiredPermissionForPath(path)
  return perm !== null && perms.includes(perm)
}

export function firstAllowedAdminPath(perms: string[]): string | null {
  const hit = ROUTE_PERMISSIONS.find(([, perm]) => perms.includes(perm))
  return hit ? hit[0] : null
}
