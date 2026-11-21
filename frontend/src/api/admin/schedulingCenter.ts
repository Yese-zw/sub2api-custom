/**
 * Admin Scheduling Center API
 * 调度中心为独立管理视图：调度池 / 账号池归类、智商检测计划与入池规则都不影响网关现有调度。
 */

import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export type SchedulingCenterPool = 'scheduling' | 'account'
/** fixed：OAuth / Setup Token 账号，调度中心只处理 Key 类型账号，这些账号固定在调度池 */
export type SchedulingCenterMemberSource = 'manual' | 'rule' | 'fallback' | 'fixed'

export interface SchedulingCenterPoolMember {
  account_id: number
  source: SchedulingCenterMemberSource
  entered_at: string
}

/** 不在调度池中的账号的调度中心状态（与账号管理中的调度状态无关） */
export type SchedulingCenterAccountStateName = 'cooling' | 'probing' | 'ready' | 'manual_out'

export interface SchedulingCenterAccountState {
  group_id: number
  account_id: number
  state: SchedulingCenterAccountStateName
  prev_source: string
  reason_code: string
  cooldown_until: string | null
  probe_status: '' | 'running' | 'passed'
  probe_started_at: string | null
  probe_passed_at: string | null
  recover_ok_count: number
  updated_at: string
}

export interface SchedulingCenterPoolView {
  members: SchedulingCenterPoolMember[]
  states: SchedulingCenterAccountState[]
  /** 分组下一次智商检测时间；未开启调度时为 null */
  next_check_at: string | null
  /** 本分组中检测计划归属于其他分组的账号（检测频率 / 模型 / 保留条数按那个分组执行） */
  plan_owners: SchedulingCenterPlanOwner[]
  /** 运维监控是否开启；未开启时没有上游错误日志，错误相关规则不会生效 */
  ops_monitoring_enabled: boolean
  /** 调度池生效在本实例上的状态与选号统计 */
  enforce: SchedulingCenterEnforceStats
}

export interface SchedulingCenterEnforceStats {
  /** 本实例当前是否按调度池选号 */
  active: boolean
  /** 网关可用的池内账号数（含恢复观察中的账号） */
  pool_size: number
  pool_hits: number
  /** 池内选号失败、回退原逻辑重选的次数 */
  fallbacks: number
  /** 候选中没有池内账号、直接走原逻辑的次数 */
  empty_fallbacks: number
  last_fallback_at?: string | null
  since: string
}

export interface SchedulingCenterPlanOwner {
  account_id: number
  group_id: number
  group_name: string
  cron_expression: string
  model_id: string
  max_results: number
}

export interface SchedulingCenterEvent {
  id: number
  group_id: number
  account_id: number
  account_name: string
  action: 'enter' | 'exit' | 'cooldown_end' | 'probe_pass' | 'probe_fail' | 'recover_pass' | 'recover_fail' | 'manual_out_end'
  source: 'manual' | 'rule' | 'system' | 'fallback' | ''
  reason: string
  detail: Record<string, unknown>
  is_bad: boolean
  created_at: string
}

export interface SchedulingCenterAccountMetrics {
  account_id: number
  /** 近 24 小时缓存率（0-100），无请求时为 null */
  cache_rate: number | null
  requests: number
  /** 首 Token（毫秒）：优先最近 10 次智商检测均值，否则近 24 小时真实请求均值；入池规则用同一个值 */
  first_token_ms: number | null
  first_token_source?: 'test' | 'traffic'
  test_first_token_ms: number | null
  traffic_first_token_ms: number | null
  /** 最近一次智商检测：passed | incorrect | error，未检测为空 */
  iq_status: string
  iq_message?: string
  iq_checked_at?: string
}

export interface SchedulingCenterGroupSettings {
  group_id: number
  enabled: boolean
  cron_expression: string
  /** 智商检测模型；空字符串表示使用各平台默认测试模型 */
  model_id: string
  rule_enabled: boolean
  min_cache_rate: number | null
  require_iq_pass: boolean
  max_first_token_ms: number | null

  /** 出池规则：满足任意一条即出池；阈值为 null 表示该条件关闭 */
  exit_enabled: boolean
  exit_cache_drop_points: number | null
  exit_cache_min_requests: number
  exit_iq_window: number
  exit_iq_max_degraded: number | null
  exit_latency_ms: number | null
  exit_latency_count: number
  exit_error_count: number | null
  /** 首 Token 出池规则观察的最近流式请求次数（exit_latency_count 为其中过慢的触发次数） */
  exit_latency_window: number
  /** 错误出池规则观察的最近真实请求次数（exit_error_count 为其中上游错误的触发次数） */
  exit_error_window: number
  /** 冷却结束、探测通过后，真实请求需连续正常的次数 */
  recover_success_count: number

  /** 首 Token 取最近 N 次智商检测的平均值 */
  first_token_test_samples: number
  /** 入池：缓存率至少需要的 24h 请求数 */
  entry_cache_min_requests: number
  /** 入池：要求最近 N 次智商检测都通过 */
  entry_iq_pass_count: number
  /** 出池：真实请求只看最近 N 小时 */
  exit_traffic_window_hours: number
  /** 退避：统计「坏」次数的窗口（分钟） */
  bad_window_minutes: number
  /** 退避：第 1、2、3… 次坏的冷却分钟数，最后一个为封顶 */
  cooldown_minutes: number[]
  /** 恢复：探测与实测的首 Token 上限，null 表示不检查 */
  recover_latency_ms: number | null

  /** 兜底：调度池账号数低于下限时，按排序从账号池补入 */
  fallback_enabled: boolean
  fallback_min_accounts: number
  /** 排序优先级：rate_multiplier 低优先、first_token 低优先、cache_rate 高优先 */
  fallback_order: string[]
  /** 兜底账号要求最近一次智商检测通过 */
  fallback_require_iq_pass: boolean
  /** 手动移出的有效期（小时），到期后入池规则 / 兜底可再次选中；0 表示永久 */
  manual_out_hours: number
  /** 每个账号保留的检测记录条数（10~1000） */
  record_retention: number
  /** 调度日志保留天数（8~365），后台定期清理更早的日志 */
  event_retention_days: number

  /** 调度池生效（灰度）：开启后网关在该分组只从调度池选账号，池内无可用账号时回退原逻辑 */
  enforce_enabled: boolean

  updated_at?: string
}

export type SchedulingCenterSettingsInput = Omit<SchedulingCenterGroupSettings, 'group_id' | 'updated_at'>

export interface SchedulingCenterTestRecord {
  id: number
  status: string
  /** 测速（常规请求 hi）的首 Token；测速失败或旧记录为空 */
  first_token_ms: number | null
  /** 测速（常规请求 hi）的总耗时 */
  probe_total_ms: number | null
  /** 智商检测耗时 */
  latency_ms: number
  iq_status: string
  iq_message: string
  error_message: string
  started_at: string
}

function normalizePoolView(data: Partial<SchedulingCenterPoolView> | null | undefined): SchedulingCenterPoolView {
  return {
    members: data?.members ?? [],
    states: data?.states ?? [],
    next_check_at: data?.next_check_at ?? null,
    plan_owners: data?.plan_owners ?? [],
    ops_monitoring_enabled: data?.ops_monitoring_enabled ?? true,
    enforce: data?.enforce ?? { active: false, pool_size: 0, pool_hits: 0, fallbacks: 0, empty_fallbacks: 0, since: '' }
  }
}

export async function getPool(groupId: number): Promise<SchedulingCenterPoolView> {
  const { data } = await apiClient.get<SchedulingCenterPoolView>(`/admin/scheduling-center/groups/${groupId}/pool`)
  return normalizePoolView(data)
}

export async function moveAccounts(
  groupId: number,
  accountIds: number[],
  pool: SchedulingCenterPool
): Promise<SchedulingCenterPoolView> {
  const { data } = await apiClient.put<SchedulingCenterPoolView>(
    `/admin/scheduling-center/groups/${groupId}/pool`,
    { account_ids: accountIds, pool }
  )
  return normalizePoolView(data)
}

export async function getMetrics(
  accountIds: number[],
  groupId?: number
): Promise<Record<string, SchedulingCenterAccountMetrics>> {
  const { data } = await apiClient.post<{ metrics: Record<string, SchedulingCenterAccountMetrics> }>(
    '/admin/scheduling-center/metrics',
    { account_ids: accountIds, group_id: groupId }
  )
  return data.metrics ?? {}
}

export async function listSettings(): Promise<SchedulingCenterGroupSettings[]> {
  const { data } = await apiClient.get<{ settings: SchedulingCenterGroupSettings[] }>(
    '/admin/scheduling-center/settings'
  )
  return data.settings ?? []
}

export async function getSettings(groupId: number): Promise<SchedulingCenterGroupSettings> {
  const { data } = await apiClient.get<SchedulingCenterGroupSettings>(
    `/admin/scheduling-center/groups/${groupId}/settings`
  )
  return data
}

export async function updateSettings(
  groupId: number,
  input: SchedulingCenterSettingsInput
): Promise<SchedulingCenterGroupSettings> {
  const { data } = await apiClient.put<SchedulingCenterGroupSettings>(
    `/admin/scheduling-center/groups/${groupId}/settings`,
    input
  )
  return data
}

export async function getAccountRecords(accountId: number, limit = 100): Promise<SchedulingCenterTestRecord[]> {
  const { data } = await apiClient.get<{ records: SchedulingCenterTestRecord[] }>(
    `/admin/scheduling-center/accounts/${accountId}/records`,
    { params: { limit } }
  )
  return data.records ?? []
}

export async function listEvents(
  groupId: number,
  params: { page: number; page_size: number; keyword?: string }
): Promise<PaginatedResponse<SchedulingCenterEvent>> {
  const { data } = await apiClient.get<PaginatedResponse<SchedulingCenterEvent>>(
    `/admin/scheduling-center/groups/${groupId}/events`,
    { params }
  )
  return data
}

export interface SchedulingCenterSideMetrics {
  accounts: number
  requests: number
  errors: number
  error_rate: number | null
  input_tokens: number
  cache_creation_tokens: number
  cache_read_tokens: number
  cache_rate: number | null
  first_token_samples: number
  avg_first_token_ms: number | null
  slow_requests: number
  slow_rate: number | null
  iq_checks: number
  iq_passed: number
  iq_incorrect: number
  iq_errored: number
  iq_pass_rate: number | null
  iq_avg_first_token_ms: number | null
}

export interface SchedulingCenterSeriesPoint {
  time: string
  pool: SchedulingCenterSideMetrics
  account_pool: SchedulingCenterSideMetrics
}

export interface SchedulingCenterComparison {
  group_id: number
  from: string
  to: string
  hours: number
  bucket_minutes: number
  slow_threshold_ms: number
  pool: SchedulingCenterSideMetrics
  account_pool: SchedulingCenterSideMetrics
  series: SchedulingCenterSeriesPoint[]
  snapshot: {
    group_accounts: number
    pool_accounts: number
    account_pool_accounts: number
    states: Record<string, number>
  }
}

/** 对比：按请求发生时账号所在的池，统计调度池与账号池的真实数据 */
export async function getComparison(groupId: number, hours: number): Promise<SchedulingCenterComparison> {
  const { data } = await apiClient.get<SchedulingCenterComparison>(
    `/admin/scheduling-center/groups/${groupId}/comparison`,
    { params: { hours } }
  )
  return data
}

/** 导出：分组最近 hours 小时的完整调度中心数据（JSON，不含凭据） */
export async function exportData(groupId: number, hours: number): Promise<Record<string, unknown>> {
  const { data } = await apiClient.get<Record<string, unknown>>(
    `/admin/scheduling-center/groups/${groupId}/export`,
    { params: { hours }, timeout: 120_000 }
  )
  return data
}

export interface SchedulingCenterResetOptions {
  clear_pool: boolean
  clear_states: boolean
  clear_events: boolean
  clear_records: boolean
  reset_settings: boolean
}

export interface SchedulingCenterResetResult {
  pool_members: number
  states: number
  events: number
  records: number
  settings_reset: boolean
}

/** 初始化分组调度：按选项清空调度池、状态、调度日志、检测记录，或把规则参数恢复默认 */
export async function resetGroup(groupId: number, options: SchedulingCenterResetOptions): Promise<SchedulingCenterResetResult> {
  const { data } = await apiClient.post<SchedulingCenterResetResult>(
    `/admin/scheduling-center/groups/${groupId}/reset`,
    options
  )
  return data
}

export default {
  resetGroup,
  getComparison,
  exportData,
  listEvents,
  getPool,
  moveAccounts,
  getMetrics,
  listSettings,
  getSettings,
  updateSettings,
  getAccountRecords
}
