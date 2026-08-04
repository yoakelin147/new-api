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
import type { ApiResponse, ManagedCustomer, PageData, SalesUser } from './types'

export async function getSalesUsers(keyword = '', page = 1, pageSize = 10) {
  const params = new URLSearchParams({
    keyword,
    p: String(page),
    page_size: String(pageSize),
  })
  const response = await api.get<ApiResponse<PageData<SalesUser>>>(
    `/api/sales/?${params}`
  )
  return response.data
}

export async function getSalesCustomers(
  salesUserId: number,
  page = 1,
  pageSize = 5
) {
  const params = new URLSearchParams({
    p: String(page),
    page_size: String(pageSize),
  })
  const response = await api.get<ApiResponse<PageData<ManagedCustomer>>>(
    `/api/sales/${salesUserId}/customers?${params}`
  )
  return response.data
}

export async function getAvailableSalesCustomers(
  keyword = '',
  page = 1,
  pageSize = 5
) {
  const params = new URLSearchParams({
    keyword,
    p: String(page),
    page_size: String(pageSize),
  })
  const response = await api.get<ApiResponse<PageData<ManagedCustomer>>>(
    `/api/sales/available-customers?${params}`
  )
  return response.data
}

export async function assignSalesCustomers(
  salesUserId: number,
  customerUserIds: number[]
) {
  const response = await api.post<ApiResponse>(
    `/api/sales/${salesUserId}/customers`,
    { customer_user_ids: customerUserIds }
  )
  return response.data
}

export async function removeSalesCustomer(
  salesUserId: number,
  customerUserId: number
) {
  const response = await api.delete<ApiResponse>(
    `/api/sales/${salesUserId}/customers/${customerUserId}`
  )
  return response.data
}

export async function removeSalesCustomers(
  salesUserId: number,
  customerUserIds: number[]
) {
  const response = await api.post<ApiResponse>(
    `/api/sales/${salesUserId}/customers/remove`,
    { customer_user_ids: customerUserIds }
  )
  return response.data
}
