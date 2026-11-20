/**
 * Daily check-in API endpoints
 */

import { apiClient } from './client'

export interface CheckinStreakReward {
  day: number
  bonus: number
}

export interface CheckinRecord {
  id: number
  user_id: number
  checkin_date: string
  base_reward: number
  streak_bonus: number
  reward_amount: number
  streak_days: number
  created_at: string
}

export interface CheckinDayPreview {
  day: number
  bonus: number
  claimed: boolean
  is_today: boolean
}

export type CheckinIneligibleReason = '' | 'user_inactive' | 'recharge_required' | 'account_too_new'

export interface CheckinStatus {
  enabled: boolean
  today: string
  timezone: string
  checked_in_today: boolean
  current_streak: number
  today_record?: CheckinRecord
  eligible: boolean
  ineligible_reason?: CheckinIneligibleReason
  reward_min: number
  reward_max: number
  min_total_recharged: number
  min_account_age_days: number
  streak_cycle_days: number
  streak_rewards: CheckinStreakReward[]
  next_bonus: number
  cycle: CheckinDayPreview[]
  month_dates: string[]
  totals: {
    total_days: number
    total_reward: number
  }
}

export interface CheckinResult {
  record: CheckinRecord
  new_balance: number
}

export interface CheckinHistoryPage {
  items: CheckinRecord[]
  total: number
  page: number
  page_size: number
}

export async function getStatus(): Promise<CheckinStatus> {
  const { data } = await apiClient.get<CheckinStatus>('/checkin/status')
  return data
}

export async function checkin(): Promise<CheckinResult> {
  const { data } = await apiClient.post<CheckinResult>('/checkin')
  return data
}

export async function getHistory(page = 1, pageSize = 10): Promise<CheckinHistoryPage> {
  const { data } = await apiClient.get<CheckinHistoryPage>('/checkin/history', {
    params: { page, page_size: pageSize }
  })
  return data
}

export const checkinAPI = {
  getStatus,
  checkin,
  getHistory
}

export default checkinAPI
