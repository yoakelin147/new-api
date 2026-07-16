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
import type { CostStatisticsFilters, CostStatisticsResponse } from './types'

export async function getCostStatistics(params: {
  isAdmin: boolean
  page: number
  pageSize: number
  filters: CostStatisticsFilters
}): Promise<CostStatisticsResponse> {
  const endpoint = params.isAdmin
    ? '/api/cost_statistics/'
    : '/api/cost_statistics/self'
  const query = buildQueryParams({
    p: params.page,
    page_size: params.pageSize,
    period: params.filters.period,
    start_timestamp:
      params.isAdmin && params.filters.period === 'custom'
        ? params.filters.start_timestamp
        : undefined,
    end_timestamp:
      params.isAdmin && params.filters.period === 'custom'
        ? params.filters.end_timestamp
        : undefined,
    type: params.filters.type,
    model_name: params.filters.model?.trim(),
    token_name: params.filters.token?.trim(),
    group: params.filters.group?.trim(),
    username: params.isAdmin ? params.filters.username?.trim() : undefined,
    channel: params.isAdmin ? params.filters.channel?.trim() : undefined,
  })
  const res = await api.get(`${endpoint}?${query}`)
  return res.data
}
