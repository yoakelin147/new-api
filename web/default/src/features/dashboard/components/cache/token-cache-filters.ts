import { getRollingDateRange, type TimeGranularity } from '@/lib/time'

export const DEFAULT_TOKEN_CACHE_RANGE_DAYS = 1

export interface TokenCacheDashboardFilters {
  start_timestamp: Date
  end_timestamp: Date
  time_granularity: TimeGranularity
  model_name?: string
  channel_id?: number
  range_days?: number | null
}

export function buildDefaultTokenCacheFilters(): TokenCacheDashboardFilters {
  const { start, end } = getRollingDateRange(DEFAULT_TOKEN_CACHE_RANGE_DAYS)
  return {
    start_timestamp: start,
    end_timestamp: end,
    time_granularity: 'hour',
    model_name: '',
    channel_id: undefined,
    range_days: DEFAULT_TOKEN_CACHE_RANGE_DAYS,
  }
}
