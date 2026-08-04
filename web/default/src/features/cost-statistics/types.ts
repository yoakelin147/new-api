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

export type CostStatisticsPeriod =
  | 'today'
  | 'yesterday'
  | 'week'
  | 'month'
  | 'last_month'
  | 'custom'
export type CostStatisticsType = 'all' | 'consume' | 'refund'
export type CostStatisticsSubject = 'self' | 'managed' | `customer:${number}`

export interface CostStatisticsFilters {
  period: CostStatisticsPeriod
  type: CostStatisticsType
  model?: string
  token?: string
  group?: string
  username?: string
  channel?: string
  start_timestamp?: number
  end_timestamp?: number
}

export interface CostStatisticsRow {
  date: string
  type: number
  token_id: number
  token_name: string
  model_name: string
  group: string
  username: string
  channel_id: number
  channel_name: string
  prompt_tokens: number
  input_tokens: number
  completion_tokens: number
  total_tokens: number
  request_count: number
  quota: number
  group_ratio: number
  model_ratio: number
  completion_ratio: number
  cache_tokens: number
  cache_ratio: number
  cache_write_5m_tokens: number
  cache_write_5m_ratio: number
  cache_write_1h_tokens: number
  cache_write_1h_ratio: number
  input_unit_price: number
  output_unit_price: number
  cache_read_unit_price: number
  cache_write_unit_price: number
  cache_write_1h_unit_price: number
  model_price: number
  billing_mode: string
  matched_tier: string
  use_time_seconds: number
}

export interface CostStatisticsSummary {
  quota: number
  request_count: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
}

export interface CostStatisticsData {
  page: number
  page_size: number
  total: number
  items: CostStatisticsRow[]
  summary: CostStatisticsSummary
  period: CostStatisticsPeriod
  start_timestamp: number
  end_timestamp: number
}

export interface CostStatisticsResponse {
  success: boolean
  message?: string
  data?: CostStatisticsData
}
