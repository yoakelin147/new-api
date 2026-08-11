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
export const MAX_STATISTICS_CUSTOM_RANGE_SECONDS = 183 * 24 * 60 * 60
export const CUSTOM_STATISTICS_RANGE_VALUE = 'custom'

export interface StatisticsTimeRange {
  start_timestamp: number
  end_timestamp: number
}

export type StatisticsTimeRangeValidationError =
  | 'end_before_start'
  | 'range_too_long'

export function buildStatisticsTimeRange(
  days: number,
  now: Date = new Date()
): StatisticsTimeRange {
  const end = now.getTime()
  return {
    start_timestamp: Math.floor((end - days * 24 * 60 * 60 * 1000) / 1000),
    end_timestamp: Math.floor(end / 1000),
  }
}

export function validateStatisticsTimeRange(
  range: StatisticsTimeRange
): StatisticsTimeRangeValidationError | null {
  if (range.end_timestamp < range.start_timestamp) {
    return 'end_before_start'
  }
  if (
    range.end_timestamp - range.start_timestamp >
    MAX_STATISTICS_CUSTOM_RANGE_SECONDS
  ) {
    return 'range_too_long'
  }
  return null
}

export function dateToStatisticsTimestamp(date?: Date): number | undefined {
  return date ? Math.floor(date.getTime() / 1000) : undefined
}

export function statisticsTimestampToDate(timestamp: number): Date {
  return new Date(timestamp * 1000)
}
