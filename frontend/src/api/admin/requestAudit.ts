/**
 * Gateway request audit API (admin only).
 *
 * Captures the full request body sent by the client and the full response body
 * returned to the client for each gateway request, linked to usage records.
 */

import { apiClient } from '../client'

export interface RequestAuditSettings {
  enabled: boolean
  retention_days: number
  max_body_bytes: number
}

export interface RequestAuditLog {
  id: number
  created_at: string
  request_id: string
  client_request_id: string
  user_id?: number
  api_key_id?: number
  account_id?: number
  method: string
  path: string
  model: string
  stream: boolean
  status_code: number
  duration_ms: number
  client_ip: string
  user_agent: string
  request_headers?: Record<string, string>
  request_content_type: string
  request_body: string
  request_body_size: number
  request_truncated: boolean
  response_content_type: string
  response_body: string
  response_body_size: number
  response_truncated: boolean
}

export async function getSettings(): Promise<RequestAuditSettings> {
  const { data } = await apiClient.get('/admin/request-audit/settings')
  return data
}

export async function updateSettings(settings: RequestAuditSettings): Promise<RequestAuditSettings> {
  const { data } = await apiClient.put('/admin/request-audit/settings', settings)
  return data
}

export async function getByUsageLog(usageLogId: number): Promise<RequestAuditLog> {
  const { data } = await apiClient.get(`/admin/request-audit/usage/${usageLogId}`)
  return data
}

export const requestAuditAPI = {
  getSettings,
  updateSettings,
  getByUsageLog
}

export default requestAuditAPI
