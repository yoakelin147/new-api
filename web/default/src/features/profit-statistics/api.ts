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
import { api } from '@/lib/api'
import { buildQueryParams } from '@/features/usage-logs/lib/utils'
import type {
  ChannelProfitRatioInput,
  ChannelProfitRatioResponse,
  ChannelProfitRatiosResponse,
  ProfitStatisticsFilters,
  ProfitStatisticsResponse,
} from './types'

export async function getProfitStatistics(params: {
  page: number
  pageSize: number
  filters: ProfitStatisticsFilters
}): Promise<ProfitStatisticsResponse> {
  const query = buildQueryParams({
    p: params.page,
    page_size: params.pageSize,
    period: params.filters.period,
    start_timestamp:
      params.filters.period === 'custom'
        ? params.filters.start_timestamp
        : undefined,
    end_timestamp:
      params.filters.period === 'custom'
        ? params.filters.end_timestamp
        : undefined,
    type: params.filters.type,
    model_name: params.filters.model?.trim(),
    group: params.filters.group?.trim(),
    username: params.filters.username?.trim(),
    channel: params.filters.channel?.trim(),
  })
  const res = await api.get(`/api/profit_statistics/?${query}`)
  return res.data
}

export async function getChannelProfitRatios(): Promise<ChannelProfitRatiosResponse> {
  const res = await api.get('/api/profit_statistics/ratios')
  return res.data
}

export async function saveChannelProfitRatio(
  data: ChannelProfitRatioInput
): Promise<ChannelProfitRatioResponse> {
  const res = await api.post('/api/profit_statistics/ratios', data)
  return res.data
}

export async function deleteChannelProfitRatio(
  id: number
): Promise<{ success: boolean; message?: string }> {
  const res = await api.delete(`/api/profit_statistics/ratios/${id}`)
  return res.data
}
