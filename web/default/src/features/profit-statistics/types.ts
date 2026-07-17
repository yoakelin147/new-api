/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

export type ProfitStatisticsPeriod =
  | 'today'
  | 'week'
  | 'month'
  | 'last_month'
  | 'custom'
export type ProfitStatisticsType = 'all' | 'consume' | 'refund'
export type ProfitCostStatus = 'configured' | 'unconfigured' | 'estimated'

export interface ProfitStatisticsFilters {
  period: ProfitStatisticsPeriod
  type: ProfitStatisticsType
  model?: string
  group?: string
  username?: string
  channel?: string
  start_timestamp?: number
  end_timestamp?: number
}

export interface ProfitStatisticsRow {
  date: string
  username: string
  group: string
  channel_id: number
  channel_name: string
  model_name: string
  request_count: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  customer_ratio: number
  upstream_ratio: number | null
  customer_amount: number
  original_amount: number
  upstream_cost: number | null
  profit: number | null
  profit_rate: number | null
  cost_status: ProfitCostStatus
  missing_ratio_count: number
}

export interface ProfitStatisticsSummary {
  request_count: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  customer_amount: number
  original_amount: number
  upstream_cost: number
  profit: number
  profit_rate: number
  unconfigured_rows: number
  unconfigured_amount: number
  estimated_rows: number
}

export interface ProfitStatisticsData {
  page: number
  page_size: number
  total: number
  items: ProfitStatisticsRow[]
  summary: ProfitStatisticsSummary
  period: ProfitStatisticsPeriod
  start_timestamp: number
  end_timestamp: number
}

export interface ProfitStatisticsResponse {
  success: boolean
  message?: string
  data?: ProfitStatisticsData
}

export interface ChannelProfitRatio {
  id: number
  channel_id: number
  channel_name: string
  model_name: string
  group: string
  upstream_ratio: number
  enabled: boolean
  remark: string
  created_at: number
  updated_at: number
}

export interface ChannelProfitRatioInput {
  id?: number
  channel_id: number
  model_name?: string
  group?: string
  upstream_ratio: number
  enabled: boolean
  remark?: string
}

export interface ChannelProfitRatiosResponse {
  success: boolean
  message?: string
  data?: ChannelProfitRatio[]
}

export interface ChannelProfitRatioResponse {
  success: boolean
  message?: string
  data?: ChannelProfitRatio
}
