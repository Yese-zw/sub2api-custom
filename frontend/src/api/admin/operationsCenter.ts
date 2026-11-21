/**
 * Admin Operations Center API
 * 运营中心：按 base_url 去重的上游站点额度总览（数据来自账号缓存的上游余额快照）。
 */

import { apiClient } from '../client'

export interface OperationsSiteAccount {
  id: number
  name: string
  platform: string
  type: string
  status: string
}

export interface OperationsSite {
  key: string
  base_url: string
  site_url: string
  host: string
  account_count: number
  accounts: OperationsSiteAccount[]
  source_account?: OperationsSiteAccount
  mode?: string
  unit?: string
  plan_name?: string
  total_quota?: number
  used_quota?: number
  remaining_quota?: number
  unlimited: boolean
  today_cost?: number
  updated_at?: string
  has_balance: boolean
}

export async function listSites(): Promise<{ items: OperationsSite[]; total: number }> {
  const { data } = await apiClient.get<{ items: OperationsSite[]; total: number }>('/admin/operations-center/sites')
  return data
}

export interface AccountRequestRankingItem {
  account_id: number
  name: string
  platform: string
  type: string
  status: string
  deleted: boolean
  base_url?: string
  requests: number
  input_tokens: number
  output_tokens: number
  cache_tokens: number
  tokens: number
  account_cost: number
  user_cost: number
  avg_duration_ms: number
  avg_first_token_ms: number
  users: number
  models: number
  last_used_at?: string
  errors: number
  rate_limited: number
}

export interface SiteRequestRankingItem {
  key: string
  base_url: string
  site_url: string
  host: string
  account_count: number
  accounts: OperationsSiteAccount[]
  requests: number
  input_tokens: number
  output_tokens: number
  cache_tokens: number
  tokens: number
  account_cost: number
  user_cost: number
  avg_duration_ms: number
  avg_first_token_ms: number
  last_used_at?: string
  errors: number
  rate_limited: number
}

export interface AccountRequestRankingResponse {
  group_by: 'site' | 'account'
  items: AccountRequestRankingItem[] | null
  sites: SiteRequestRankingItem[] | null
  total_sites?: number
  total_requests: number
  total_tokens: number
  total_account_cost: number
  total_user_cost: number
  total_accounts: number
  total_errors: number
  start_time: string
  end_time: string
}

export interface AccountRequestRankingParams {
  /** RFC3339，精确时间范围（优先） */
  start_time?: string
  end_time?: string
  /** YYYY-MM-DD，按天范围 */
  start_date?: string
  end_date?: string
  timezone?: string
  limit?: number
  /** site：按 base_url 去重合并（默认）；account：按账号 */
  group_by?: 'site' | 'account'
}

export async function getAccountRanking(params: AccountRequestRankingParams): Promise<AccountRequestRankingResponse> {
  const { data } = await apiClient.get<AccountRequestRankingResponse>('/admin/operations-center/account-ranking', { params })
  return data
}

export interface AccountConsumption {
  account_id: number
  requests_24h: number
  cost_24h: number
  requests_7d: number
  cost_7d: number
  first_used_at: string
}

/** 各账号近 24 小时 / 7 天的账号口径消耗 */
export async function getConsumption(): Promise<AccountConsumption[]> {
  const { data } = await apiClient.get<{ items: AccountConsumption[] }>('/admin/operations-center/consumption')
  return data.items ?? []
}

export interface SiteTrendSeries {
  key: string
  label: string
  site_url?: string
  account_count: number
  requests: number[]
  account_cost: number[]
  tokens: number[]
  errors: number[]
  total_requests: number
  total_account_cost: number
  total_errors: number
}

export interface SiteTrendResponse {
  granularity: 'hour' | 'day'
  buckets: string[]
  series: SiteTrendSeries[]
  total: SiteTrendSeries
}

export interface SiteTrendParams extends Omit<AccountRequestRankingParams, 'limit' | 'group_by'> {
  granularity?: 'hour' | 'day'
  top?: number
}

/** 按 base_url 合并的站点趋势（前 N 个站点 + 其他） */
export async function getSiteTrend(params: SiteTrendParams): Promise<SiteTrendResponse> {
  const { data } = await apiClient.get<SiteTrendResponse>('/admin/operations-center/site-trend', { params })
  return data
}

export interface HeatmapCell {
  /** 1=周一 … 7=周日 */
  weekday: number
  hour: number
  requests: number
  account_cost: number
  errors: number
}

/** 星期 × 小时请求热力图 */
export async function getHeatmap(params: Omit<SiteTrendParams, 'granularity' | 'top'>): Promise<HeatmapCell[]> {
  const { data } = await apiClient.get<{ cells: HeatmapCell[] }>('/admin/operations-center/heatmap', { params })
  return data.cells ?? []
}

export type HealthIssueKind =
  | 'error'
  | 'expired'
  | 'quota_exceeded'
  | 'rate_limited'
  | 'overloaded'
  | 'temp_unschedulable'
  | 'expiring_soon'
  | 'paused'

export interface HealthAccount {
  id: number
  name: string
  platform: string
  type: string
  status: string
  base_url?: string
  last_used_at?: string
  created_at: string
}

export interface HealthIssue {
  account: HealthAccount
  kind: HealthIssueKind
  detail?: string
  until?: string
}

export interface HealthPlatformStat {
  platform: string
  total: number
  schedulable: number
  issues: number
  idle: number
}

export interface AccountHealthSummary {
  total: number
  active: number
  disabled: number
  error: number
  schedulable: number
  rate_limited: number
  overloaded: number
  temp_unschedulable: number
  quota_exceeded: number
  expired: number
  expiring_soon: number
  paused: number
  idle: number
}

export interface AccountHealthResponse {
  summary: AccountHealthSummary
  platforms: HealthPlatformStat[]
  issues: HealthIssue[]
  idle: HealthAccount[]
  idle_threshold_hours: number
  generated_at: string
}

/** 账号健康：状态汇总、问题账号与闲置账号 */
export async function getAccountHealth(): Promise<AccountHealthResponse> {
  const { data } = await apiClient.get<AccountHealthResponse>('/admin/operations-center/account-health')
  return data
}

export default { listSites, getAccountRanking, getConsumption, getSiteTrend, getHeatmap, getAccountHealth }
