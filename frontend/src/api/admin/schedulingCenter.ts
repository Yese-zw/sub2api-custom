/**
 * Admin Scheduling Center API
 * 调度中心为独立管理视图：调度池 / 账号池归类、智商检测计划与入池规则都不影响网关现有调度。
 */

import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export type SchedulingCenterPool = 'scheduling' | 'account'
export type SchedulingCenterMemberSource = 'manual' | 'rule'

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
}

export interface SchedulingCenterEvent {
  id: number
  group_id: number
  account_id: number
  account_name: string
  action: 'enter' | 'exit' | 'cooldown_end' | 'probe_pass' | 'probe_fail' | 'recover_pass' | 'recover_fail'
  source: 'manual' | 'rule' | 'system' | ''
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

  updated_at?: string
}

export type SchedulingCenterSettingsInput = Omit<SchedulingCenterGroupSettings, 'group_id' | 'updated_at'>

export interface SchedulingCenterTestRecord {
  id: number
  status: string
  first_token_ms: number | null
  latency_ms: number
  iq_status: string
  iq_message: string
  error_message: string
  started_at: string
}

function normalizePoolView(data: Partial<SchedulingCenterPoolView> | null | undefined): SchedulingCenterPoolView {
  return { members: data?.members ?? [], states: data?.states ?? [] }
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

export default {
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
