/**
 * User-facing Channel Monitor API endpoints
 * Read-only views for end users to inspect channel availability/status.
 */

import { apiClient } from './client'
import type { MonitorQuotaSnapshot, Provider, MonitorStatus } from './admin/channelMonitor'

export type { Provider, MonitorStatus } from './admin/channelMonitor'

export interface UserMonitorExtraModel {
  model: string
  status: MonitorStatus
  latency_ms: number | null
}

export interface MonitorTimelinePoint {
  status: MonitorStatus
  latency_ms: number | null
  ping_latency_ms: number | null
  checked_at: string
}

export interface UserMonitorView {
  id: number
  name: string
  provider: Provider
  group_name: string
  primary_model: string
  primary_status: MonitorStatus
  primary_latency_ms: number | null
  primary_ping_latency_ms: number | null
  availability_7d: number
  extra_models: UserMonitorExtraModel[]
  timeline: MonitorTimelinePoint[]
  /**
   * 主模型最近配额快照。仅当系统开启 channel_monitor_show_quota 时
   * 服务端才会下发（关闭时服务端已剥离，前端 flag 仅作纵深防御）。
   */
  latest_quota?: MonitorQuotaSnapshot | null
  /** 该监控是否开启了智商检测（卡片显示「智商状态」按钮） */
  iq_enabled?: boolean
  /** 最近若干条智商检测是否通过（旧→新，最多 10 条），用于卡片分段环 */
  iq_recent?: boolean[]
}

export interface UserMonitorListResponse {
  items: UserMonitorView[]
}

export interface UserMonitorModelDetail {
  model: string
  latest_status: MonitorStatus
  latest_latency_ms: number | null
  availability_7d: number
  availability_15d: number
  availability_30d: number
  avg_latency_7d_ms: number | null
}

export interface UserMonitorDetail {
  id: number
  name: string
  provider: Provider
  group_name: string
  models: UserMonitorModelDetail[]
}

/**
 * List all monitor views available to the current user.
 */
export async function list(options?: { signal?: AbortSignal }): Promise<UserMonitorListResponse> {
  const { data } = await apiClient.get<UserMonitorListResponse>('/channel-monitors', {
    signal: options?.signal,
  })
  return data
}

/**
 * Get detailed status (multi-window availability + latency) for a single monitor.
 */
export async function status(id: number): Promise<UserMonitorDetail> {
  const { data } = await apiClient.get<UserMonitorDetail>(`/channel-monitors/${id}/status`)
  return data
}

export type MonitorIQBenchmark = 'candy' | 'pelican' | 'creative'

export interface MonitorIQResult {
  id: number
  monitor_id: number
  model: string
  benchmark: MonitorIQBenchmark
  /** candy: passed / incorrect */
  verdict?: string
  /** pelican / creative: normal / degraded / unknown / suspicious */
  quality?: string
  label?: string
  reason?: string
  answer?: string
  has_html: boolean
  /** 仅详情接口返回 */
  html?: string
  duration_ms: number
  started_at: string
  finished_at: string
}

export interface MonitorIQFailure {
  model: string
  benchmark: MonitorIQBenchmark
  message: string
  at: string
}

export interface MonitorIQResultsResponse {
  /** 各 (模型, 检测类型) 最近一次失败原因（成功后清除，服务重启后丢失） */
  failures?: MonitorIQFailure[]
  creative_enabled: boolean
  interval_minutes: number
  retention_hours: number
  max_per_model: number
  items: MonitorIQResult[]
}

/** 最近 2 小时的智商检测成功结果（每个模型每类最多 10 条） */
export async function iqResults(id: number): Promise<MonitorIQResultsResponse> {
  const { data } = await apiClient.get<MonitorIQResultsResponse>(`/channel-monitors/${id}/iq-results`)
  return data
}

/** 单条结果详情（含 html） */
export async function iqResult(id: number, resultId: number): Promise<MonitorIQResult> {
  const { data } = await apiClient.get<MonitorIQResult>(`/channel-monitors/${id}/iq-results/${resultId}`)
  return data
}

export const channelMonitorUserAPI = {
  list,
  status,
  iqResults,
  iqResult,
}

export default channelMonitorUserAPI
