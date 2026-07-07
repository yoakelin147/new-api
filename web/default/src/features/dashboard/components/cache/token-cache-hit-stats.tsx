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
import { useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { VChart } from '@visactor/react-vchart'
import { BarChart3, Database } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import dayjs from '@/lib/dayjs'
import { formatNumber } from '@/lib/format'
import { VCHART_OPTION } from '@/lib/vchart'
import { useThemeCustomization } from '@/context/theme-customization-provider'
import { useTheme } from '@/context/theme-provider'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  type StaticDataTableColumn,
  StaticDataTable,
} from '@/components/data-table/static/static-data-table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  getTokenCacheHitGroupStats,
  getTokenCacheHitStats,
  getTokenCacheHitTrendStats,
} from '@/features/dashboard/api'
import type {
  TokenCacheHitGroupStatItem,
  TokenCacheHitStatItem,
  TokenCacheHitTrendItem,
} from '@/features/dashboard/types'
import type { TokenCacheDashboardFilters } from './token-cache-filters'

let themeManagerPromise: Promise<
  (typeof import('@visactor/vchart'))['ThemeManager']
> | null = null

type TokenCacheChartType = 'bar' | 'area'
type TokenCacheDetailsTab = 'channel' | 'group'

function formatRate(rate: number) {
  return Intl.NumberFormat(undefined, {
    style: 'percent',
    maximumFractionDigits: 2,
  }).format(rate)
}

function getChannelLabel(row: TokenCacheHitStatItem) {
  const name = row.channel_name?.trim()
  return name ? `#${row.channel_id} ${name}` : `#${row.channel_id}`
}

function getGroupLabel(row: TokenCacheHitGroupStatItem) {
  return row.group?.trim()
}

function getTrendChannelLabel(item: TokenCacheHitTrendItem) {
  const name = item.channel_name?.trim()
  return name ? name : `#${item.channel_id}`
}

function getTrendModelLabel(item: TokenCacheHitTrendItem) {
  const name = item.model_name?.trim()
  return name || 'unknown'
}

function buildApiParams(filters: TokenCacheDashboardFilters) {
  const startTime = Math.min(
    filters.start_timestamp.getTime(),
    filters.end_timestamp.getTime()
  )
  const endTime = Math.max(
    filters.start_timestamp.getTime(),
    filters.end_timestamp.getTime()
  )
  return {
    start_timestamp: Math.floor(startTime / 1000),
    end_timestamp: Math.floor(endTime / 1000),
    time_granularity: filters.time_granularity,
    model_name: filters.model_name?.trim() || undefined,
    channel_id: filters.channel_id,
  }
}

function formatTrendTime(timestamp: number, granularity: string) {
  if (granularity === 'day') return dayjs(timestamp * 1000).format('MM-DD')
  if (granularity === 'week') return dayjs(timestamp * 1000).format('MM-DD')
  return dayjs(timestamp * 1000).format('MM-DD HH:mm')
}

interface TokenCacheHitStatsProps {
  filters: TokenCacheDashboardFilters
}

export function TokenCacheHitStats({ filters }: TokenCacheHitStatsProps) {
  const { t } = useTranslation()
  const { resolvedTheme } = useTheme()
  const { customization } = useThemeCustomization()
  const [themeReady, setThemeReady] = useState(false)
  const [modelChartType, setModelChartType] =
    useState<TokenCacheChartType>('bar')
  const [channelChartType, setChannelChartType] =
    useState<TokenCacheChartType>('bar')
  const [detailsTab, setDetailsTab] = useState<TokenCacheDetailsTab>('channel')
  const themeManagerRef = useRef<
    (typeof import('@visactor/vchart'))['ThemeManager'] | null
  >(null)
  const apiParams = useMemo(() => buildApiParams(filters), [filters])

  const { data: stats = [], isLoading } = useQuery({
    queryKey: ['dashboard', 'token-cache-hit-stats', apiParams],
    queryFn: () => getTokenCacheHitStats(apiParams),
    select: (res) => (res.success ? res.data : []),
    staleTime: 60_000,
  })

  const { data: groupStats = [], isLoading: groupStatsLoading } = useQuery({
    queryKey: ['dashboard', 'token-cache-hit-group-stats', apiParams],
    queryFn: () => getTokenCacheHitGroupStats(apiParams),
    select: (res) => (res.success ? res.data : []),
    staleTime: 60_000,
    enabled: detailsTab === 'group',
  })

  const { data: modelTrendStats = [], isLoading: modelTrendLoading } =
    useQuery({
      queryKey: ['dashboard', 'token-cache-hit-trend', 'model', apiParams],
      queryFn: () =>
        getTokenCacheHitTrendStats({ ...apiParams, trend_group: 'model' }),
      select: (res) => (res.success ? res.data : []),
      staleTime: 60_000,
    })

  const { data: channelTrendStats = [], isLoading: channelTrendLoading } =
    useQuery({
      queryKey: ['dashboard', 'token-cache-hit-trend', 'channel', apiParams],
      queryFn: () =>
        getTokenCacheHitTrendStats({ ...apiParams, trend_group: 'channel' }),
      select: (res) => (res.success ? res.data : []),
      staleTime: 60_000,
    })

  useEffect(() => {
    const updateTheme = async () => {
      setThemeReady(false)
      if (!themeManagerPromise) {
        themeManagerPromise = import('@visactor/vchart').then(
          (m) => m.ThemeManager
        )
      }
      const ThemeManager = await themeManagerPromise
      themeManagerRef.current = ThemeManager
      ThemeManager.setCurrentTheme(resolvedTheme === 'dark' ? 'dark' : 'light')
      setThemeReady(true)
    }
    updateTheme()
  }, [resolvedTheme])

  const summary = useMemo(() => {
    const totals = stats.reduce(
      (acc, item) => {
        acc.requestCount += item.request_count
        acc.hitRequestCount += item.hit_request_count
        acc.promptTokens += item.prompt_tokens
        acc.completionTokens += item.completion_tokens
        acc.cacheTokens += item.cache_tokens
        acc.cacheWriteTokens += item.cache_write_tokens || 0
        acc.cacheInputTokens += item.cache_input_tokens || item.prompt_tokens
        return acc
      },
      {
        requestCount: 0,
        hitRequestCount: 0,
        promptTokens: 0,
        completionTokens: 0,
        cacheTokens: 0,
        cacheWriteTokens: 0,
        cacheInputTokens: 0,
      }
    )
    return {
      ...totals,
      hitRate:
        totals.cacheInputTokens > 0
          ? totals.cacheTokens / totals.cacheInputTokens
          : 0,
    }
  }, [stats])

  const modelTrendSpec = useMemo(
    () =>
      buildTrendSpec({
        data: modelTrendStats,
        dataId: 'tokenCacheModelTrend',
        granularity: filters.time_granularity,
        getSeriesLabel: getTrendModelLabel,
        rateLabel: t('Model Hit Rate'),
        chartType: modelChartType,
        t,
      }),
    [filters.time_granularity, modelChartType, modelTrendStats, t]
  )

  const channelTrendSpec = useMemo(
    () =>
      buildTrendSpec({
        data: channelTrendStats,
        dataId: 'tokenCacheChannelTrend',
        granularity: filters.time_granularity,
        getSeriesLabel: getTrendChannelLabel,
        rateLabel: t('Channel Hit Rate'),
        chartType: channelChartType,
        t,
      }),
    [channelChartType, channelTrendStats, filters.time_granularity, t]
  )

  const channelColumns = useMemo<
    StaticDataTableColumn<TokenCacheHitStatItem>[]
  >(
    () => [
      {
        id: 'model_channel',
        header: t('Model & Channel'),
        className: 'min-w-[220px]',
        cell: (row) => (
          <div className='flex min-w-0 flex-col gap-1'>
            <span className='truncate font-medium'>{row.model_name}</span>
            <span className='text-muted-foreground truncate text-xs'>
              {getChannelLabel(row)}
            </span>
          </div>
        ),
      },
      {
        id: 'hit_rate',
        header: t('Channel Hit Rate'),
        className: 'min-w-[180px]',
        cell: (row) => {
          const percent = Math.max(0, Math.min(row.hit_rate * 100, 100))
          return (
            <div className='flex min-w-0 flex-col gap-1.5'>
              <span className='font-medium'>{formatRate(row.hit_rate)}</span>
              <div className='bg-muted h-1.5 overflow-hidden rounded-full'>
                <div
                  className='bg-primary h-full rounded-full'
                  style={{ width: `${percent}%` }}
                />
              </div>
            </div>
          )
        },
      },
      {
        id: 'request_count',
        header: t('Requests'),
        cellClassName: 'text-right tabular-nums',
        className: 'text-right',
        cell: (row) => formatNumber(row.request_count),
      },
      {
        id: 'hit_request_count',
        header: t('Cache Hit Requests'),
        cellClassName: 'text-right tabular-nums',
        className: 'text-right',
        cell: (row) => formatNumber(row.hit_request_count),
      },
      {
        id: 'prompt_tokens',
        header: t('Non-Cache Input Tokens'),
        cellClassName: 'text-right tabular-nums',
        className: 'text-right',
        cell: (row) => formatNumber(row.prompt_tokens),
      },
      {
        id: 'cache_write_tokens',
        header: t('Cache Write Tokens'),
        cellClassName: 'text-right tabular-nums',
        className: 'text-right',
        cell: (row) => formatNumber(row.cache_write_tokens || 0),
      },
      {
        id: 'cache_tokens',
        header: t('Cache Read Tokens'),
        cellClassName: 'text-right tabular-nums',
        className: 'text-right',
        cell: (row) => formatNumber(row.cache_tokens),
      },
      {
        id: 'cache_input_tokens',
        header: t('Cache Input Tokens'),
        cellClassName: 'text-right tabular-nums',
        className: 'text-right',
        cell: (row) =>
          formatNumber(row.cache_input_tokens || row.prompt_tokens),
      },
      {
        id: 'completion_tokens',
        header: t('Output Tokens'),
        cellClassName: 'text-right tabular-nums',
        className: 'text-right',
        cell: (row) => formatNumber(row.completion_tokens),
      },
    ],
    [t]
  )

  const groupColumns = useMemo<
    StaticDataTableColumn<TokenCacheHitGroupStatItem>[]
  >(
    () => [
      {
        id: 'model_group',
        header: t('Model & Group'),
        className: 'min-w-[220px]',
        cell: (row) => (
          <div className='flex min-w-0 flex-col gap-1'>
            <span className='truncate font-medium'>{row.model_name}</span>
            <span className='text-muted-foreground truncate text-xs'>
              {getGroupLabel(row) || t('Unknown')}
            </span>
          </div>
        ),
      },
      {
        id: 'hit_rate',
        header: t('Group Hit Rate'),
        className: 'min-w-[180px]',
        cell: (row) => {
          const percent = Math.max(0, Math.min(row.hit_rate * 100, 100))
          return (
            <div className='flex min-w-0 flex-col gap-1.5'>
              <span className='font-medium'>{formatRate(row.hit_rate)}</span>
              <div className='bg-muted h-1.5 overflow-hidden rounded-full'>
                <div
                  className='bg-primary h-full rounded-full'
                  style={{ width: `${percent}%` }}
                />
              </div>
            </div>
          )
        },
      },
      {
        id: 'request_count',
        header: t('Requests'),
        cellClassName: 'text-right tabular-nums',
        className: 'text-right',
        cell: (row) => formatNumber(row.request_count),
      },
      {
        id: 'hit_request_count',
        header: t('Cache Hit Requests'),
        cellClassName: 'text-right tabular-nums',
        className: 'text-right',
        cell: (row) => formatNumber(row.hit_request_count),
      },
      {
        id: 'prompt_tokens',
        header: t('Non-Cache Input Tokens'),
        cellClassName: 'text-right tabular-nums',
        className: 'text-right',
        cell: (row) => formatNumber(row.prompt_tokens),
      },
      {
        id: 'cache_write_tokens',
        header: t('Cache Write Tokens'),
        cellClassName: 'text-right tabular-nums',
        className: 'text-right',
        cell: (row) => formatNumber(row.cache_write_tokens || 0),
      },
      {
        id: 'cache_tokens',
        header: t('Cache Read Tokens'),
        cellClassName: 'text-right tabular-nums',
        className: 'text-right',
        cell: (row) => formatNumber(row.cache_tokens),
      },
      {
        id: 'cache_input_tokens',
        header: t('Cache Input Tokens'),
        cellClassName: 'text-right tabular-nums',
        className: 'text-right',
        cell: (row) =>
          formatNumber(row.cache_input_tokens || row.prompt_tokens),
      },
      {
        id: 'completion_tokens',
        header: t('Output Tokens'),
        cellClassName: 'text-right tabular-nums',
        className: 'text-right',
        cell: (row) => formatNumber(row.completion_tokens),
      },
    ],
    [t]
  )

  const detailsTitle =
    detailsTab === 'channel'
      ? t('Token Cache Hit Rate by Model and Channel')
      : t('Token Cache Hit Rate by Model and Group')

  const summaryCards = [
    {
      label: t('Requests'),
      value: formatNumber(summary.requestCount),
      description: t('Total requests in selected range'),
    },
    {
      label: t('Cache Hit Requests'),
      value: formatNumber(summary.hitRequestCount),
      description: t('Requests with cache read tokens'),
    },
    {
      label: t('Cache Read Tokens'),
      value: formatNumber(summary.cacheTokens),
      description: t('Total cache read tokens'),
    },
    {
      label: t('Cache Write Tokens'),
      value: formatNumber(summary.cacheWriteTokens),
      description: t('Total cache write tokens'),
    },
    {
      label: t('Overall Cache Hit Rate'),
      value: formatRate(summary.hitRate),
      description: t('Cache read tokens / cache input tokens'),
    },
  ]

  return (
    <div className='flex flex-col gap-3'>
      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-5'>
        {summaryCards.map((card) => (
          <Card key={card.label} size='sm'>
            <CardHeader>
              <CardTitle>{card.label}</CardTitle>
              <CardDescription>{card.description}</CardDescription>
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <Skeleton className='h-7 w-24' />
              ) : (
                <div className='text-2xl font-semibold tabular-nums'>
                  {card.value}
                </div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      <TrendChartPanel
        title={t('Model Cache Hit Rate Trend')}
        loading={modelTrendLoading}
        empty={modelTrendStats.length === 0}
        spec={modelTrendSpec}
        themeReady={themeReady}
        resolvedTheme={resolvedTheme}
        themeKey={customization.preset}
        chartType={modelChartType}
        onChartTypeChange={setModelChartType}
      />

      <TrendChartPanel
        title={t('Channel Cache Hit Rate Trend')}
        loading={channelTrendLoading}
        empty={channelTrendStats.length === 0}
        spec={channelTrendSpec}
        themeReady={themeReady}
        resolvedTheme={resolvedTheme}
        themeKey={customization.preset}
        chartType={channelChartType}
        onChartTypeChange={setChannelChartType}
      />

      <div className='overflow-hidden rounded-lg border'>
        <div className='flex w-full flex-wrap items-center justify-between gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
          <div className='flex items-center gap-2'>
            <Database className='text-muted-foreground/60 size-4' />
            <div className='text-sm font-semibold'>{detailsTitle}</div>
          </div>
          <Tabs
            value={detailsTab}
            onValueChange={(value) =>
              setDetailsTab(value as TokenCacheDetailsTab)
            }
            className='shrink-0'
          >
            <TabsList>
              <TabsTrigger value='channel' className='px-2.5 text-xs'>
                {t('Model & Channel')}
              </TabsTrigger>
              <TabsTrigger value='group' className='px-2.5 text-xs'>
                {t('Model & Group')}
              </TabsTrigger>
            </TabsList>
          </Tabs>
        </div>
        {detailsTab === 'channel' && isLoading ? (
          <div className='p-3'>
            <Skeleton className='h-64 w-full' />
          </div>
        ) : detailsTab === 'channel' ? (
          <StaticDataTable
            columns={channelColumns}
            data={stats}
            getRowKey={(row) => `${row.model_name}-${row.channel_id}`}
            emptyContent={t('No token cache statistics found')}
          />
        ) : groupStatsLoading ? (
          <div className='p-3'>
            <Skeleton className='h-64 w-full' />
          </div>
        ) : (
          <StaticDataTable
            columns={groupColumns}
            data={groupStats}
            getRowKey={(row) => `${row.model_name}-${row.group}`}
            emptyContent={t('No token cache statistics found')}
          />
        )}
      </div>
    </div>
  )
}

function buildTrendSpec(options: {
  data: TokenCacheHitTrendItem[]
  dataId: string
  granularity: string
  getSeriesLabel: (item: TokenCacheHitTrendItem) => string
  rateLabel: string
  chartType: TokenCacheChartType
  t: ReturnType<typeof useTranslation>['t']
}) {
  const values = options.data.map((item) => {
    const rateValue = Number((item.hit_rate * 100).toFixed(2))
    return {
      Time: formatTrendTime(item.created_at, options.granularity),
      Series: options.getSeriesLabel(item),
      HitRate: rateValue,
      hitRateText: `${rateValue.toFixed(2)}%`,
      cacheTokensText: formatNumber(item.cache_tokens),
      cacheWriteTokensText: formatNumber(item.cache_write_tokens || 0),
      cacheInputTokensText: formatNumber(
        item.cache_input_tokens || item.prompt_tokens
      ),
      requestsText: formatNumber(item.request_count),
    }
  })

  return {
    type: options.chartType,
    data: [{ id: options.dataId, values }],
    xField: 'Time',
    yField: 'HitRate',
    seriesField: 'Series',
    stack: false,
    legends: {
      visible: true,
      orient: 'bottom',
      position: 'middle',
    },
    axes: [
      {
        orient: 'left',
        label: {
          formatMethod: (value: number) => `${Number(value).toFixed(2)}%`,
        },
      },
      { orient: 'bottom' },
    ],
    tooltip: {
      dimension: {
        content: [
          {
            key: (datum: { Series?: string }) => datum.Series ?? '',
            value: (datum: { hitRateText?: string }) =>
              datum.hitRateText ?? '-',
          },
          {
            key: options.t('Cache Read Tokens'),
            value: (datum: { cacheTokensText?: string }) =>
              datum.cacheTokensText ?? '-',
          },
          {
            key: options.t('Cache Write Tokens'),
            value: (datum: { cacheWriteTokensText?: string }) =>
              datum.cacheWriteTokensText ?? '-',
          },
          {
            key: options.t('Cache Input Tokens'),
            value: (datum: { cacheInputTokensText?: string }) =>
              datum.cacheInputTokensText ?? '-',
          },
          {
            key: options.t('Requests'),
            value: (datum: { requestsText?: string }) =>
              datum.requestsText ?? '-',
          },
        ],
      },
      mark: {
        content: [
          {
            key: options.rateLabel,
            value: (datum: { hitRateText?: string }) =>
              datum.hitRateText ?? '-',
          },
          {
            key: options.t('Cache Read Tokens'),
            value: (datum: { cacheTokensText?: string }) =>
              datum.cacheTokensText ?? '-',
          },
          {
            key: options.t('Cache Write Tokens'),
            value: (datum: { cacheWriteTokensText?: string }) =>
              datum.cacheWriteTokensText ?? '-',
          },
          {
            key: options.t('Cache Input Tokens'),
            value: (datum: { cacheInputTokensText?: string }) =>
              datum.cacheInputTokensText ?? '-',
          },
          {
            key: options.t('Requests'),
            value: (datum: { requestsText?: string }) =>
              datum.requestsText ?? '-',
          },
        ],
      },
    },
    ...(options.chartType === 'area'
      ? {
          point: { visible: true },
          area: { style: { fillOpacity: 0.35 } },
        }
      : {}),
    background: { fill: 'transparent' },
  }
}

function TrendChartPanel({
  title,
  loading,
  empty,
  spec,
  themeReady,
  resolvedTheme,
  themeKey,
  chartType,
  onChartTypeChange,
}: {
  title: string
  loading: boolean
  empty: boolean
  spec: Record<string, unknown>
  themeReady: boolean
  resolvedTheme: string
  themeKey?: string
  chartType: TokenCacheChartType
  onChartTypeChange: (value: TokenCacheChartType) => void
}) {
  const { t } = useTranslation()

  return (
    <div className='overflow-hidden rounded-lg border'>
      <div className='flex w-full flex-wrap items-center justify-between gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
        <div className='flex items-center gap-2'>
          <BarChart3 className='text-muted-foreground/60 size-4' />
          <div className='text-sm font-semibold'>{title}</div>
        </div>
        <Tabs
          value={chartType}
          onValueChange={(value) =>
            onChartTypeChange(value as TokenCacheChartType)
          }
          className='shrink-0'
        >
          <TabsList>
            <TabsTrigger value='bar' className='px-2.5 text-xs'>
              {t('Bar Chart')}
            </TabsTrigger>
            <TabsTrigger value='area' className='px-2.5 text-xs'>
              {t('Area Chart')}
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </div>
      {loading ? (
        <div className='p-3'>
          <Skeleton className='h-80 w-full' />
        </div>
      ) : empty ? (
        <div className='text-muted-foreground flex h-48 items-center justify-center px-4 text-sm'>
          {t('No chart data available')}
        </div>
      ) : (
        <div className='h-80 p-2'>
          {themeReady && (
            <VChart
              key={`${title}-${chartType}-${resolvedTheme}-${themeKey}`}
              spec={{
                ...spec,
                theme: resolvedTheme === 'dark' ? 'dark' : 'light',
                background: 'transparent',
              }}
              option={VCHART_OPTION}
            />
          )}
        </div>
      )}
    </div>
  )
}
