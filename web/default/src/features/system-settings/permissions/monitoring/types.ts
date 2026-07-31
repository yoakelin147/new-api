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
export type MonitoringAccessKey = {
  id: number
  name: string
  key_type: 'monitoring'
  key_prefix: string
  status: number
  expires_at: number | null
  created_time: number
  updated_time: number
  last_used_time: number
  created_by: number
  updated_by: number
}

export type CreateMonitoringAccessKeyInput = {
  name: string
  expires_at: number | null
}

export type CreatedMonitoringAccessKey = {
  access_key: MonitoringAccessKey
  key: string
}

export type ApiResponse<T = undefined> = {
  success: boolean
  message?: string
  data: T
}
