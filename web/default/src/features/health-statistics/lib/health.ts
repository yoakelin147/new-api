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
import type {
  PerfChannelModelSummary,
  PerfModelSummary,
} from '@/features/performance-metrics/types'

export type HealthStatus = 'healthy' | 'warning' | 'critical'
export type HealthStatusFilter = 'all' | HealthStatus

const STATUS_WEIGHT: Record<HealthStatus, number> = {
  healthy: 0,
  warning: 1,
  critical: 2,
}

export function successRateStatus(value: number): HealthStatus {
  if (value >= 99) return 'healthy'
  if (value >= 95) return 'warning'
  return 'critical'
}

export function latencyStatus(value: number): HealthStatus {
  if (!Number.isFinite(value) || value <= 0 || value <= 15_000) {
    return 'healthy'
  }
  if (value <= 30_000) return 'warning'
  return 'critical'
}

export function getHealthStatus(
  row: Pick<PerfModelSummary, 'success_rate' | 'avg_latency_ms'>
): HealthStatus {
  const success = successRateStatus(Number(row.success_rate))
  const latency = latencyStatus(Number(row.avg_latency_ms))
  return STATUS_WEIGHT[success] >= STATUS_WEIGHT[latency] ? success : latency
}

export function averageMetric<T>(
  rows: T[],
  read: (row: T) => number,
  isValid: (value: number) => boolean = Number.isFinite
): number {
  let count = 0
  let total = 0
  for (const row of rows) {
    const value = Number(read(row))
    if (!isValid(value)) continue
    total += value
    count++
  }
  return count > 0 ? total / count : Number.NaN
}

export function matchesStatus(
  row: PerfModelSummary | PerfChannelModelSummary,
  filter: HealthStatusFilter,
  onlyAnomalies: boolean
): boolean {
  const status = getHealthStatus(row)
  if (onlyAnomalies && status === 'healthy') return false
  return filter === 'all' || status === filter
}

export function healthStatusClass(status: HealthStatus): string {
  if (status === 'healthy') return 'text-success'
  if (status === 'warning') return 'text-warning'
  return 'text-destructive'
}

export function healthStatusBackground(status: HealthStatus): string {
  if (status === 'healthy') return 'bg-success/12 text-success'
  if (status === 'warning') return 'bg-warning/15 text-warning'
  return 'bg-destructive/12 text-destructive'
}

export function healthStatusLabel(status: HealthStatus): string {
  if (status === 'healthy') return 'Healthy'
  if (status === 'warning') return 'Warning'
  return 'Critical'
}
