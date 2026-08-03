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
import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { VChart } from '@visactor/react-vchart'
import { AlertCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Empty, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getPerfMetrics } from '@/features/performance-metrics/api'
import {
  formatLatency,
  formatThroughput,
  formatUptimePct,
} from '@/features/performance-metrics/lib/format'
import type { PerformanceGroup } from '@/features/performance-metrics/types'
import {
  getHealthStatus,
  healthStatusBackground,
  healthStatusLabel,
} from '../lib/health'

type TrendMetric = 'success_rate' | 'avg_latency_ms' | 'avg_ttft_ms' | 'avg_tps'

type ModelHealthSheetProps = {
  modelName: string | null
  hours: number
  onOpenChange: (open: boolean) => void
}

const TREND_METRICS: Array<{ value: TrendMetric; label: string }> = [
  { value: 'success_rate', label: 'Success rate' },
  { value: 'avg_latency_ms', label: 'Average latency' },
  { value: 'avg_ttft_ms', label: 'TTFT' },
  { value: 'avg_tps', label: 'TPS' },
]

function formatTrendValue(metric: TrendMetric, value: number) {
  if (metric === 'success_rate') return formatUptimePct(value)
  if (metric === 'avg_tps') return formatThroughput(value)
  return formatLatency(value)
}

function timestampLabel(timestamp: number, locale: string) {
  const milliseconds =
    timestamp > 1_000_000_000_000 ? timestamp : timestamp * 1000
  return new Intl.DateTimeFormat(locale, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(milliseconds))
}

export function ModelHealthSheet(props: ModelHealthSheetProps) {
  const { t, i18n } = useTranslation()
  const [metric, setMetric] = useState<TrendMetric>('success_rate')
  const { resolvedTheme, themeReady } = useChartTheme()
  const detailQuery = useQuery({
    queryKey: [
      'health-statistics',
      'model-detail',
      props.modelName,
      props.hours,
    ],
    queryFn: () => getPerfMetrics(props.modelName!, props.hours),
    enabled: Boolean(props.modelName),
    staleTime: 60_000,
    retry: false,
  })

  const groups = useMemo(
    () => detailQuery.data?.data.groups ?? [],
    [detailQuery.data]
  )
  const chartData = useMemo(
    () =>
      groups.flatMap((group) =>
        group.series.map((point) => ({
          group: group.group,
          time: timestampLabel(point.ts, i18n.language),
          value: Number(point[metric]),
        }))
      ),
    [groups, i18n.language, metric]
  )
  const chartSpec = useMemo(
    () => ({
      type: 'line' as const,
      data: [{ id: 'groups', values: chartData }],
      xField: 'time',
      yField: 'value',
      seriesField: 'group',
      padding: { top: 16, right: 16, bottom: 48, left: 52 },
      point: { visible: chartData.length <= 100 },
      legends: { visible: true, orient: 'bottom' as const },
      axes: [
        { orient: 'bottom' as const, label: { autoRotate: true } },
        { orient: 'left' as const, min: 0 },
      ],
      tooltip: {
        dimension: {
          content: [
            {
              key: (datum: { group: string }) => datum.group,
              value: (datum: { value: number }) =>
                formatTrendValue(metric, Number(datum.value)),
            },
          ],
        },
      },
      animation: false,
      background: 'transparent',
      theme: resolvedTheme === 'dark' ? 'dark' : 'light',
    }),
    [chartData, metric, resolvedTheme]
  )

  return (
    <Sheet open={Boolean(props.modelName)} onOpenChange={props.onOpenChange}>
      <SheetContent className='w-full overflow-y-auto sm:max-w-4xl'>
        <SheetHeader className='border-b'>
          <SheetTitle>{t('Model health details')}</SheetTitle>
          <SheetDescription>
            {props.modelName} ·{' '}
            {t('Last {{hours}} hours', { hours: props.hours })}
          </SheetDescription>
        </SheetHeader>

        <div className='flex flex-col gap-5 px-4 pb-6'>
          {detailQuery.isError && (
            <Alert variant='destructive'>
              <AlertCircle />
              <AlertTitle>
                {t('Failed to load model health details')}
              </AlertTitle>
              <AlertDescription>{t('Please try again later')}</AlertDescription>
            </Alert>
          )}

          <section className='overflow-hidden rounded-xl border'>
            <div className='bg-muted/30 border-b px-4 py-3'>
              <h3 className='font-medium'>{t('Group comparison')}</h3>
              <p className='text-muted-foreground mt-0.5 text-xs'>
                {t('Only groups with calls during this window are shown')}
              </p>
            </div>
            {detailQuery.isLoading ? (
              <div className='grid gap-2 p-4'>
                {Array.from({ length: 4 }).map((_, index) => (
                  <Skeleton key={index} className='h-9 w-full' />
                ))}
              </div>
            ) : groups.length === 0 ? (
              <Empty className='min-h-36 border-0'>
                <EmptyHeader>
                  <EmptyTitle>{t('No group data in this window')}</EmptyTitle>
                </EmptyHeader>
              </Empty>
            ) : (
              <GroupComparisonTable groups={groups} />
            )}
          </section>

          <section className='overflow-hidden rounded-xl border'>
            <div className='flex flex-col gap-3 border-b px-4 py-3 lg:flex-row lg:items-center lg:justify-between'>
              <div>
                <h3 className='font-medium'>{t('Group health trend')}</h3>
                <p className='text-muted-foreground mt-0.5 text-xs'>
                  {t(
                    'Trend granularity follows the performance metrics source'
                  )}
                </p>
              </div>
              <Tabs
                value={metric}
                onValueChange={(value) => setMetric(value as TrendMetric)}
              >
                <TabsList className='max-w-full overflow-x-auto'>
                  {TREND_METRICS.map((item) => (
                    <TabsTrigger key={item.value} value={item.value}>
                      {t(item.label)}
                    </TabsTrigger>
                  ))}
                </TabsList>
              </Tabs>
            </div>
            <div className='h-80 p-2'>
              {detailQuery.isLoading ? (
                <Skeleton className='h-full w-full' />
              ) : chartData.length === 0 ? (
                <Empty className='h-full border-0'>
                  <EmptyHeader>
                    <EmptyTitle>{t('No trend data in this window')}</EmptyTitle>
                  </EmptyHeader>
                </Empty>
              ) : (
                themeReady && (
                  <VChart
                    spec={chartSpec}
                    option={VCHART_OPTION}
                    key={`${resolvedTheme}-${metric}`}
                  />
                )
              )}
            </div>
          </section>
        </div>
      </SheetContent>
    </Sheet>
  )
}

function GroupComparisonTable({ groups }: { groups: PerformanceGroup[] }) {
  const { t } = useTranslation()
  return (
    <div className='overflow-x-auto'>
      <table className='w-full min-w-[680px] text-sm'>
        <thead className='bg-muted/40 text-muted-foreground'>
          <tr>
            <th className='px-4 py-2 text-left font-medium'>{t('Group')}</th>
            <th className='px-4 py-2 text-right font-medium'>{t('Status')}</th>
            <th className='px-4 py-2 text-right font-medium'>
              {t('Success rate')}
            </th>
            <th className='px-4 py-2 text-right font-medium'>
              {t('Average latency')}
            </th>
            <th className='px-4 py-2 text-right font-medium'>{t('TTFT')}</th>
            <th className='px-4 py-2 text-right font-medium'>{t('TPS')}</th>
          </tr>
        </thead>
        <tbody className='divide-y'>
          {groups.map((group) => {
            const status = getHealthStatus(group)
            return (
              <tr key={group.group}>
                <td className='px-4 py-2 font-mono'>{group.group}</td>
                <td className='px-4 py-2 text-right'>
                  <Badge className={healthStatusBackground(status)}>
                    {t(healthStatusLabel(status))}
                  </Badge>
                </td>
                <td className='px-4 py-2 text-right font-mono tabular-nums'>
                  {formatUptimePct(group.success_rate)}
                </td>
                <td className='px-4 py-2 text-right font-mono tabular-nums'>
                  {formatLatency(group.avg_latency_ms)}
                </td>
                <td className='px-4 py-2 text-right font-mono tabular-nums'>
                  {formatLatency(group.avg_ttft_ms)}
                </td>
                <td className='px-4 py-2 text-right font-mono tabular-nums'>
                  {formatThroughput(group.avg_tps)}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
