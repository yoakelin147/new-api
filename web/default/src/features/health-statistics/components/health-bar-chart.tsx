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
import { useMemo } from 'react'
import { VChart } from '@visactor/react-vchart'
import { useTranslation } from 'react-i18next'
import { useChartTheme } from '@/lib/use-chart-theme'
import { VCHART_OPTION } from '@/lib/vchart'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Empty, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'

export type HealthChartRow = {
  name: string
  value: number
}

const HEALTH_BAR_COLORS = [
  '#ef4444',
  '#f97316',
  '#eab308',
  '#84cc16',
  '#10b981',
  '#14b8a6',
  '#06b6d4',
  '#3b82f6',
  '#6366f1',
  '#8b5cf6',
  '#a855f7',
  '#d946ef',
  '#ec4899',
  '#f43f5e',
] as const

type HealthBarChartProps = {
  title: string
  data: HealthChartRow[]
  loading: boolean
  suffix: string
  max?: number
  onSelect?: (name: string) => void
}

function formatChartValue(value: number, suffix: string) {
  const digits = suffix === 'ms' ? 0 : 2
  return `${value.toLocaleString(undefined, {
    maximumFractionDigits: digits,
    minimumFractionDigits: Number.isInteger(value) ? 0 : digits,
  })}${suffix}`
}

export function HealthBarChart(props: HealthBarChartProps) {
  const { t } = useTranslation()
  const { resolvedTheme, themeReady } = useChartTheme()
  const chartData = useMemo(
    () =>
      props.data.map((row, index) => ({
        ...row,
        color: HEALTH_BAR_COLORS[index % HEALTH_BAR_COLORS.length],
      })),
    [props.data]
  )
  const spec = useMemo(
    () => ({
      type: 'bar' as const,
      direction: 'horizontal' as const,
      data: [{ id: 'health', values: chartData }],
      xField: 'value',
      yField: 'name',
      padding: { top: 8, right: 82, bottom: 24, left: 8 },
      bar: {
        style: {
          fill: (datum: HealthChartRow & { color: string }) => datum.color,
          cornerRadius: 4,
          cursor: props.onSelect ? 'pointer' : 'default',
        },
      },
      label: {
        visible: true,
        position: 'right' as const,
        offset: 6,
        overlap: false,
        formatMethod: (_value: number, datum: HealthChartRow) =>
          formatChartValue(datum.value, props.suffix),
        style: {
          fontSize: 11,
          fontWeight: 600,
        },
      },
      axes: [
        {
          orient: 'bottom' as const,
          min: 0,
          ...(props.max ? { max: props.max } : {}),
          label: { formatMethod: (value: number) => `${value}${props.suffix}` },
        },
        {
          orient: 'left' as const,
          label: { maxLineWidth: 110, ellipsis: '…' },
        },
      ],
      tooltip: {
        mark: {
          content: [
            {
              key: (datum: HealthChartRow) => datum.name,
              value: (datum: HealthChartRow) =>
                formatChartValue(datum.value, props.suffix),
            },
          ],
        },
      },
      animation: false,
      background: 'transparent',
      theme: resolvedTheme === 'dark' ? 'dark' : 'light',
    }),
    [chartData, props.max, props.onSelect, props.suffix, resolvedTheme]
  )

  return (
    <Card>
      <CardHeader className='border-b'>
        <CardTitle>{props.title}</CardTitle>
      </CardHeader>
      <CardContent>
        {props.loading ? (
          <Skeleton className='h-72 w-full' />
        ) : props.data.length === 0 ? (
          <Empty className='h-72 border-0'>
            <EmptyHeader>
              <EmptyTitle>{t('No health data in this window')}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className='h-72'>
            {themeReady && (
              <VChart
                spec={spec}
                option={VCHART_OPTION}
                key={resolvedTheme}
                onClick={(event: { datum?: { name?: unknown } }) => {
                  const name = event.datum?.name
                  if (props.onSelect && typeof name === 'string') {
                    props.onSelect(name)
                  }
                }}
              />
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
