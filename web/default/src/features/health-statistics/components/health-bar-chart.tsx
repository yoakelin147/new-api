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

type HealthBarChartProps = {
  title: string
  data: HealthChartRow[]
  loading: boolean
  color: string
  suffix: string
  max?: number
}

export function HealthBarChart(props: HealthBarChartProps) {
  const { t } = useTranslation()
  const { resolvedTheme, themeReady } = useChartTheme()
  const spec = useMemo(
    () => ({
      type: 'bar' as const,
      direction: 'horizontal' as const,
      data: [{ id: 'health', values: props.data }],
      xField: 'value',
      yField: 'name',
      padding: { top: 8, right: 22, bottom: 24, left: 8 },
      bar: {
        style: {
          fill: props.color,
          cornerRadius: 4,
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
                `${Number(datum.value).toFixed(2)}${props.suffix}`,
            },
          ],
        },
      },
      animation: false,
      background: 'transparent',
      theme: resolvedTheme === 'dark' ? 'dark' : 'light',
    }),
    [props.color, props.data, props.max, props.suffix, resolvedTheme]
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
              <VChart spec={spec} option={VCHART_OPTION} key={resolvedTheme} />
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
