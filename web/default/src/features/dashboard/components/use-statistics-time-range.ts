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
import { useCallback, useState } from 'react'
import type { TimeGranularity } from '@/lib/time'
import { getDefaultDays, getSavedGranularity, saveGranularity } from '../lib'
import {
  buildStatisticsTimeRange,
  validateStatisticsTimeRange,
  type StatisticsTimeRange,
} from './statistics-time-range'

export function useStatisticsTimeRange() {
  const [timeGranularity, setTimeGranularity] = useState<TimeGranularity>(() =>
    getSavedGranularity()
  )
  const [selectedRange, setSelectedRange] = useState<number | null>(() =>
    getDefaultDays(timeGranularity)
  )
  const [timeRange, setTimeRange] = useState<StatisticsTimeRange>(() =>
    buildStatisticsTimeRange(getDefaultDays(timeGranularity))
  )
  const [draftRange, setDraftRange] = useState(timeRange)
  const [filterOpen, setFilterOpen] = useState(false)

  const handleRangeChange = useCallback((days: number) => {
    setSelectedRange(days)
    setTimeRange(buildStatisticsTimeRange(days))
  }, [])

  const handleGranularityChange = useCallback(
    (granularity: TimeGranularity) => {
      setTimeGranularity(granularity)
      saveGranularity(granularity)
      if (selectedRange === null) return

      const days = getDefaultDays(granularity)
      if (days !== selectedRange) {
        handleRangeChange(days)
      }
    },
    [handleRangeChange, selectedRange]
  )

  const handleFilterOpenChange = useCallback(
    (open: boolean) => {
      if (open) {
        setDraftRange(timeRange)
      }
      setFilterOpen(open)
    },
    [timeRange]
  )

  const applyDraftRange = useCallback(() => {
    const error = validateStatisticsTimeRange(draftRange)
    if (error) return error

    setTimeRange(draftRange)
    setSelectedRange(null)
    setFilterOpen(false)
    return null
  }, [draftRange])

  const resetTimeRange = useCallback(() => {
    const days = getDefaultDays(timeGranularity)
    const nextRange = buildStatisticsTimeRange(days)
    setDraftRange(nextRange)
    setTimeRange(nextRange)
    setSelectedRange(days)
    setFilterOpen(false)
  }, [timeGranularity])

  return {
    timeGranularity,
    selectedRange,
    timeRange,
    draftRange,
    filterOpen,
    setDraftRange,
    handleRangeChange,
    handleGranularityChange,
    handleFilterOpenChange,
    applyDraftRange,
    resetTimeRange,
  }
}

export type StatisticsTimeRangeState = ReturnType<typeof useStatisticsTimeRange>
