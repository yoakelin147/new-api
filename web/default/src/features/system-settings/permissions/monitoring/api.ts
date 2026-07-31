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
import type {
  ApiResponse,
  CreateMonitoringAccessKeyInput,
  CreatedMonitoringAccessKey,
  MonitoringAccessKey,
} from './types'

const monitoringAccessKeyPath = '/api/system-access-key/monitoring'

export async function listMonitoringAccessKeys() {
  const response = await api.get<ApiResponse<MonitoringAccessKey[]>>(
    monitoringAccessKeyPath
  )
  return response.data
}

export async function createMonitoringAccessKey(
  input: CreateMonitoringAccessKeyInput
) {
  const response = await api.post<ApiResponse<CreatedMonitoringAccessKey>>(
    monitoringAccessKeyPath,
    input
  )
  return response.data
}

export async function updateMonitoringAccessKeyStatus(
  id: number,
  enabled: boolean
) {
  const response = await api.patch<ApiResponse>(
    `${monitoringAccessKeyPath}/${id}/status`,
    { enabled }
  )
  return response.data
}

export async function deleteMonitoringAccessKey(id: number) {
  const response = await api.delete<ApiResponse>(
    `${monitoringAccessKeyPath}/${id}`
  )
  return response.data
}
