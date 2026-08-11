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
import {
  Activity,
  AlertCircle,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  Gauge,
  HeartPulse,
  RefreshCw,
  Search,
  Settings2,
  Timer,
  TriangleAlert,
  XCircle,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import {
  sideDrawerContentClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
  SideDrawerSection,
  SideDrawerSectionHeader,
} from '@/components/drawer-layout'
import { Main } from '@/components/layout'
import {
  getAdminPerfChannelSummary,
  getPerfMetricsSummary,
} from '@/features/performance-metrics/api'
import {
  formatLatency,
  formatThroughput,
  formatUptimePct,
} from '@/features/performance-metrics/lib/format'
import type { PerfChannelModelSummary } from '@/features/performance-metrics/types'
import { HealthBarChart } from './components/health-bar-chart'
import { ModelHealthSheet } from './components/model-health-sheet'
import {
  averageMetric,
  getHealthStatus,
  healthStatusBackground,
  healthStatusClass,
  healthStatusLabel,
  matchesStatus,
  type HealthStatus,
  type HealthStatusFilter,
} from './lib/health'

const MODEL_WINDOWS = [1, 6, 24]
const CHANNEL_WINDOWS = [5, 10, 15, 30, 60]
const TOP_LIMITS = [10, 20, 50, 100]
const ANOMALY_PAGE_SIZES = [10, 20, 50, 100]
const MATRIX_CHANNELS_PER_PAGE = 6
const AUTO_REFRESH_OPTIONS = [
  { value: 0, label: 'Off' },
  { value: 30_000, label: '30 seconds' },
  { value: 60_000, label: '60 seconds' },
  { value: 300_000, label: '5 minutes' },
]

type AnomalyRow = {
  key: string
  modelName: string
  channelName: string
  successRate: number
  latency: number
  tps: number
  status: HealthStatus
}

export function HealthStatistics() {
  const { t, i18n } = useTranslation()
  const [modelHours, setModelHours] = useState(1)
  const [channelMinutes, setChannelMinutes] = useState(60)
  const [topLimit, setTopLimit] = useState(10)
  const [autoRefreshMs, setAutoRefreshMs] = useState(60_000)
  const [modelSearch, setModelSearch] = useState('')
  const [channelSearch, setChannelSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<HealthStatusFilter>('all')
  const [onlyAnomalies, setOnlyAnomalies] = useState(false)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [selectedModel, setSelectedModel] = useState<string | null>(null)

  const modelQuery = useQuery({
    queryKey: ['health-statistics', 'models', modelHours],
    queryFn: () => getPerfMetricsSummary(modelHours),
    refetchInterval: autoRefreshMs || false,
    staleTime: 30_000,
    retry: false,
  })
  const channelQuery = useQuery({
    queryKey: ['health-statistics', 'channels', channelMinutes, topLimit],
    queryFn: () => getAdminPerfChannelSummary(channelMinutes, topLimit),
    refetchInterval: autoRefreshMs || false,
    staleTime: 30_000,
    retry: false,
  })

  const models = useMemo(
    () => modelQuery.data?.data.models ?? [],
    [modelQuery.data]
  )
  const channelModels = useMemo(
    () => channelQuery.data?.data.models ?? [],
    [channelQuery.data]
  )

  const modelSearchValue = modelSearch.trim().toLowerCase()
  const channelSearchValue = channelSearch.trim().toLowerCase()
  const filteredModels = useMemo(
    () =>
      models.filter(
        (row) =>
          (!modelSearchValue ||
            row.model_name.toLowerCase().includes(modelSearchValue)) &&
          matchesStatus(row, statusFilter, onlyAnomalies)
      ),
    [models, modelSearchValue, onlyAnomalies, statusFilter]
  )
  const filteredChannelModels = useMemo(
    () =>
      channelModels.filter(
        (row) =>
          (!modelSearchValue ||
            row.model_name.toLowerCase().includes(modelSearchValue)) &&
          (!channelSearchValue ||
            row.channel_name.toLowerCase().includes(channelSearchValue) ||
            String(row.channel_id).includes(channelSearchValue)) &&
          matchesStatus(row, statusFilter, onlyAnomalies)
      ),
    [
      channelModels,
      channelSearchValue,
      modelSearchValue,
      onlyAnomalies,
      statusFilter,
    ]
  )

  const modelStatusCounts = useMemo(() => {
    const counts: Record<HealthStatus, number> = {
      healthy: 0,
      warning: 0,
      critical: 0,
    }
    for (const row of models) counts[getHealthStatus(row)]++
    return counts
  }, [models])
  const modelAverages = useMemo(
    () => ({
      successRate: averageMetric(models, (row) => row.success_rate),
      latency: averageMetric(
        models,
        (row) => row.avg_latency_ms,
        (value) => Number.isFinite(value) && value > 0
      ),
      tps: averageMetric(
        models,
        (row) => row.avg_tps,
        (value) => Number.isFinite(value) && value >= 0
      ),
    }),
    [models]
  )

  const anomalies = useMemo<AnomalyRow[]>(() => {
    return filteredChannelModels
      .filter((row) => getHealthStatus(row) !== 'healthy')
      .map((row) => ({
        key: `channel-${row.model_name}-${row.channel_id}`,
        modelName: row.model_name,
        channelName: row.channel_name || `#${row.channel_id}`,
        successRate: row.success_rate,
        latency: row.avg_latency_ms,
        tps: row.avg_tps,
        status: getHealthStatus(row),
      }))
      .sort((a, b) => {
        if (a.status !== b.status) return a.status === 'critical' ? -1 : 1
        return a.successRate - b.successRate
      })
  }, [filteredChannelModels])

  const failureRanking = useMemo(
    () =>
      [...filteredModels]
        .sort((a, b) => a.success_rate - b.success_rate)
        .slice(0, topLimit)
        .map((row) => ({
          name: row.model_name,
          value: Math.max(0, 100 - row.success_rate),
        })),
    [filteredModels, topLimit]
  )
  const latencyRanking = useMemo(
    () =>
      [...filteredModels]
        .filter((row) => row.avg_latency_ms > 0)
        .sort((a, b) => b.avg_latency_ms - a.avg_latency_ms)
        .slice(0, topLimit)
        .map((row) => ({ name: row.model_name, value: row.avg_latency_ms })),
    [filteredModels, topLimit]
  )
  const tpsRanking = useMemo(
    () =>
      [...filteredModels]
        .sort((a, b) => b.avg_tps - a.avg_tps)
        .slice(0, topLimit)
        .map((row) => ({ name: row.model_name, value: row.avg_tps })),
    [filteredModels, topLimit]
  )

  const latestUpdatedAt = Math.max(
    modelQuery.dataUpdatedAt,
    channelQuery.dataUpdatedAt
  )
  const refreshing = modelQuery.isFetching || channelQuery.isFetching

  const refresh = () => {
    void Promise.all([modelQuery.refetch(), channelQuery.refetch()])
  }

  return (
    <Main className='bg-muted/30 overflow-y-auto p-3 sm:p-4'>
      <div className='mx-auto flex w-full max-w-[1800px] flex-col gap-4'>
        <div className='flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between'>
          <div>
            <div className='flex items-center gap-2'>
              <HeartPulse className='text-primary size-6' aria-hidden='true' />
              <h1 className='text-2xl font-semibold tracking-tight'>
                {t('Health Monitoring')}
              </h1>
            </div>
            <p className='text-muted-foreground mt-1 text-sm'>
              {t('Inspect model, group, and channel health from one dashboard')}
            </p>
          </div>
          <div className='flex flex-wrap items-center gap-2'>
            <span className='text-muted-foreground text-xs'>
              {latestUpdatedAt > 0
                ? t('Last updated: {{time}}', {
                    time: new Intl.DateTimeFormat(i18n.language, {
                      hour: '2-digit',
                      minute: '2-digit',
                      second: '2-digit',
                    }).format(new Date(latestUpdatedAt)),
                  })
                : t('Not updated yet')}
            </span>
            <Button variant='outline' onClick={() => setSettingsOpen(true)}>
              <Settings2 data-icon='inline-start' />
              {t('Monitoring settings')}
            </Button>
            <Button variant='outline' onClick={refresh} disabled={refreshing}>
              <RefreshCw
                data-icon='inline-start'
                className={cn(refreshing && 'animate-spin')}
              />
              {t('Refresh')}
            </Button>
          </div>
        </div>

        {(modelQuery.isError || channelQuery.isError) && (
          <Alert variant='destructive'>
            <AlertCircle />
            <AlertTitle>{t('Some health data failed to load')}</AlertTitle>
            <AlertDescription>
              {t(
                'Model and channel data are loaded independently. Refresh to retry the failed source.'
              )}
            </AlertDescription>
          </Alert>
        )}

        <section className='flex flex-col gap-4'>
          <div className='flex flex-wrap items-center justify-between gap-2'>
            <div className='flex flex-wrap items-center gap-2'>
              <h2 className='text-lg font-semibold tracking-tight'>
                {t('Model health')}
              </h2>
              <Badge variant='secondary'>
                {t('All enabled groups combined')}
              </Badge>
            </div>
            <Badge variant='outline'>
              {t('{{count}} hours', { count: modelHours })}
            </Badge>
          </div>

          <Alert>
            <Activity />
            <AlertTitle>{t('Model and group statistics scope')}</AlertTitle>
            <AlertDescription>
              {t(
                'Each model row combines calls from all enabled groups, including the auto group. Select a model in the charts to view its metrics by group.'
              )}
            </AlertDescription>
          </Alert>

          <ModelOverviewCards
            loading={modelQuery.isLoading}
            activeModels={models.length}
            statusCounts={modelStatusCounts}
            averages={modelAverages}
          />

          <div className='grid gap-4 xl:grid-cols-3'>
            <HealthBarChart
              title={t('Highest model failure rates')}
              data={failureRanking}
              loading={modelQuery.isLoading}
              suffix='%'
              max={100}
              onSelect={setSelectedModel}
            />
            <HealthBarChart
              title={t('Highest model latency')}
              data={latencyRanking}
              loading={modelQuery.isLoading}
              suffix='ms'
              onSelect={setSelectedModel}
            />
            <HealthBarChart
              title={t('Highest model TPS')}
              data={tpsRanking}
              loading={modelQuery.isLoading}
              suffix=''
              onSelect={setSelectedModel}
            />
          </div>
        </section>

        <section className='flex flex-col gap-4'>
          <div className='flex flex-wrap items-center justify-between gap-2'>
            <h2 className='text-lg font-semibold tracking-tight'>
              {t('Channel health')}
            </h2>
            <Badge variant='outline'>
              {t('{{count}} minutes', { count: channelMinutes })}
            </Badge>
          </div>

          <AnomaliesTable
            rows={anomalies}
            loading={channelQuery.isLoading}
            onSelectModel={setSelectedModel}
          />

          <ChannelHealthMatrix
            rows={filteredChannelModels}
            loading={channelQuery.isLoading}
            onSelectModel={setSelectedModel}
          />

          <ChannelRankings
            rows={filteredChannelModels}
            loading={channelQuery.isLoading}
            onSelectModel={setSelectedModel}
          />
        </section>
      </div>

      <HealthSettingsSheet
        open={settingsOpen}
        topLimit={topLimit}
        autoRefreshMs={autoRefreshMs}
        modelSearch={modelSearch}
        statusFilter={statusFilter}
        onlyAnomalies={onlyAnomalies}
        modelHours={modelHours}
        channelMinutes={channelMinutes}
        channelSearch={channelSearch}
        onOpenChange={setSettingsOpen}
        onTopLimitChange={setTopLimit}
        onAutoRefreshChange={setAutoRefreshMs}
        onModelSearchChange={setModelSearch}
        onStatusFilterChange={setStatusFilter}
        onOnlyAnomaliesChange={setOnlyAnomalies}
        onModelHoursChange={setModelHours}
        onChannelMinutesChange={setChannelMinutes}
        onChannelSearchChange={setChannelSearch}
      />

      <ModelHealthSheet
        modelName={selectedModel}
        hours={modelHours}
        onOpenChange={(open) => {
          if (!open) setSelectedModel(null)
        }}
      />
    </Main>
  )
}

type HealthSettingsSheetProps = {
  open: boolean
  topLimit: number
  autoRefreshMs: number
  modelSearch: string
  statusFilter: HealthStatusFilter
  onlyAnomalies: boolean
  modelHours: number
  channelMinutes: number
  channelSearch: string
  onOpenChange: (open: boolean) => void
  onTopLimitChange: (value: number) => void
  onAutoRefreshChange: (value: number) => void
  onModelSearchChange: (value: string) => void
  onStatusFilterChange: (value: HealthStatusFilter) => void
  onOnlyAnomaliesChange: (value: boolean) => void
  onModelHoursChange: (value: number) => void
  onChannelMinutesChange: (value: number) => void
  onChannelSearchChange: (value: string) => void
}

function HealthSettingsSheet(props: HealthSettingsSheetProps) {
  const { t } = useTranslation()
  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent className={sideDrawerContentClassName('sm:max-w-lg')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>{t('Monitoring settings')}</SheetTitle>
          <SheetDescription>
            {t('Configure filters, time periods, and refresh behavior')}
          </SheetDescription>
        </SheetHeader>

        <div className={sideDrawerFormClassName()}>
          <SideDrawerSection>
            <SideDrawerSectionHeader
              title={t('Global filters')}
              description={t(
                'These filters apply to both model and channel health sections'
              )}
              icon={<Settings2 className='size-4' aria-hidden='true' />}
            />
            <FieldGroup className='grid gap-4 sm:grid-cols-2'>
              <Field>
                <FieldLabel>{t('Status')}</FieldLabel>
                <Select
                  value={props.statusFilter}
                  onValueChange={(value) =>
                    value &&
                    props.onStatusFilterChange(value as HealthStatusFilter)
                  }
                >
                  <SelectTrigger className='w-full'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {(['all', 'healthy', 'warning', 'critical'] as const).map(
                        (status) => (
                          <SelectItem key={status} value={status}>
                            {status === 'all'
                              ? t('All')
                              : t(healthStatusLabel(status))}
                          </SelectItem>
                        )
                      )}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>
              <Field>
                <FieldLabel>{t('Display limit')}</FieldLabel>
                <Select
                  value={String(props.topLimit)}
                  onValueChange={(value) =>
                    value && props.onTopLimitChange(Number(value))
                  }
                >
                  <SelectTrigger className='w-full'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {TOP_LIMITS.map((limit) => (
                        <SelectItem key={limit} value={String(limit)}>
                          Top {limit}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>
              <Field>
                <FieldLabel>{t('Auto refresh')}</FieldLabel>
                <Select
                  value={String(props.autoRefreshMs)}
                  onValueChange={(value) =>
                    value != null && props.onAutoRefreshChange(Number(value))
                  }
                >
                  <SelectTrigger className='w-full'>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {AUTO_REFRESH_OPTIONS.map((option) => (
                        <SelectItem
                          key={option.value}
                          value={String(option.value)}
                        >
                          {t(option.label)}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>
              <Field>
                <FieldLabel>{t('Search models')}</FieldLabel>
                <div className='relative'>
                  <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2' />
                  <Input
                    value={props.modelSearch}
                    onChange={(event) =>
                      props.onModelSearchChange(event.target.value)
                    }
                    placeholder={t('Model name')}
                    className='pl-8'
                  />
                </div>
              </Field>
              <Field className='sm:col-span-2'>
                <FieldLabel>{t('Anomaly focus')}</FieldLabel>
                <label className='flex h-9 cursor-pointer items-center justify-between gap-3 rounded-lg border px-3'>
                  <span className='text-sm'>{t('Only anomalies')}</span>
                  <Switch
                    checked={props.onlyAnomalies}
                    onCheckedChange={props.onOnlyAnomaliesChange}
                    size='sm'
                  />
                </label>
              </Field>
            </FieldGroup>
          </SideDrawerSection>

          <SideDrawerSection>
            <SideDrawerSectionHeader
              title={t('Model health')}
              description={t(
                'Overall model and group health for the selected model period'
              )}
            />
            <Field>
              <FieldLabel>{t('Model statistics period')}</FieldLabel>
              <Select
                value={String(props.modelHours)}
                onValueChange={(value) =>
                  value && props.onModelHoursChange(Number(value))
                }
              >
                <SelectTrigger className='w-full'>
                  <SelectValue>
                    {t('{{count}} hours', { count: props.modelHours })}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {MODEL_WINDOWS.map((hours) => (
                      <SelectItem key={hours} value={String(hours)}>
                        {t('{{count}} hours', { count: hours })}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
          </SideDrawerSection>

          <SideDrawerSection>
            <SideDrawerSectionHeader
              title={t('Channel health')}
              description={t(
                'Current model-channel health for the selected channel period'
              )}
            />
            <FieldGroup className='grid gap-4 sm:grid-cols-2'>
              <Field>
                <FieldLabel>{t('Channel statistics period')}</FieldLabel>
                <Select
                  value={String(props.channelMinutes)}
                  onValueChange={(value) =>
                    value && props.onChannelMinutesChange(Number(value))
                  }
                >
                  <SelectTrigger className='w-full'>
                    <SelectValue>
                      {t('{{count}} minutes', {
                        count: props.channelMinutes,
                      })}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {CHANNEL_WINDOWS.map((minutes) => (
                        <SelectItem key={minutes} value={String(minutes)}>
                          {t('{{count}} minutes', { count: minutes })}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>
              <Field>
                <FieldLabel>{t('Search channels')}</FieldLabel>
                <div className='relative'>
                  <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2' />
                  <Input
                    value={props.channelSearch}
                    onChange={(event) =>
                      props.onChannelSearchChange(event.target.value)
                    }
                    placeholder={t('Channel name or ID')}
                    className='pl-8'
                  />
                </div>
              </Field>
            </FieldGroup>
          </SideDrawerSection>

          <SideDrawerSection>
            <Alert>
              <Activity />
              <AlertTitle>
                {t('How these statistics are calculated')}
              </AlertTitle>
              <AlertDescription>
                <p>
                  {t(
                    'Only models and model-channel combinations called during the selected window are shown. Summary averages are equal-weight averages because the summary API does not expose request counts. No data is not treated as unhealthy.'
                  )}
                </p>
                <p>
                  {t(
                    'The same thresholds apply to overall model, model-group, and model-channel health: healthy means success rate ≥ 90%; warning means success rate 60–<90%; success rates below 60% are critical. Latency and TPS are for reference only and do not affect health status.'
                  )}
                </p>
              </AlertDescription>
            </Alert>
          </SideDrawerSection>
        </div>
      </SheetContent>
    </Sheet>
  )
}

type ModelOverviewCardsProps = {
  loading: boolean
  activeModels: number
  statusCounts: Record<HealthStatus, number>
  averages: { successRate: number; latency: number; tps: number }
}

function ModelOverviewCards(props: ModelOverviewCardsProps) {
  const { t } = useTranslation()
  const cards = [
    {
      label: 'Active models',
      value: String(props.activeModels),
      icon: Activity,
    },
    {
      label: 'Healthy models',
      value: String(props.statusCounts.healthy),
      icon: CheckCircle2,
      className: 'text-success',
    },
    {
      label: 'Warning models',
      value: String(props.statusCounts.warning),
      icon: TriangleAlert,
      className: 'text-warning',
    },
    {
      label: 'Critical models',
      value: String(props.statusCounts.critical),
      icon: XCircle,
      className: 'text-destructive',
    },
    {
      label: 'Average success rate',
      value: formatUptimePct(props.averages.successRate),
      icon: HeartPulse,
    },
    {
      label: 'Average latency',
      value: formatLatency(props.averages.latency),
      icon: Timer,
    },
    {
      label: 'Average TPS',
      value: formatThroughput(props.averages.tps),
      icon: Gauge,
    },
  ]
  return (
    <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4 xl:grid-cols-7'>
      {cards.map((card) => {
        const Icon = card.icon
        return (
          <Card key={card.label} size='sm'>
            <CardContent>
              <div className='text-muted-foreground flex items-center gap-1.5 text-xs'>
                <Icon className={cn('size-3.5', card.className)} />
                <span>{t(card.label)}</span>
              </div>
              {props.loading ? (
                <Skeleton className='mt-2 h-7 w-20' />
              ) : (
                <div
                  className={cn(
                    'mt-2 font-mono text-xl font-semibold tabular-nums',
                    card.className
                  )}
                >
                  {card.value}
                </div>
              )}
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}

function HealthBadge({ status }: { status: HealthStatus }) {
  const { t } = useTranslation()
  return (
    <Badge className={healthStatusBackground(status)}>
      {t(healthStatusLabel(status))}
    </Badge>
  )
}

function AnomaliesTable(props: {
  rows: AnomalyRow[]
  loading: boolean
  onSelectModel: (model: string) => void
}) {
  const { t } = useTranslation()
  const [pageSize, setPageSize] = useState(10)
  const [pageIndex, setPageIndex] = useState(0)
  const pageCount = Math.max(1, Math.ceil(props.rows.length / pageSize))
  const activePageIndex = Math.min(pageIndex, pageCount - 1)
  const visibleRows = props.rows.slice(
    activePageIndex * pageSize,
    (activePageIndex + 1) * pageSize
  )
  const firstVisibleRow =
    props.rows.length === 0 ? 0 : activePageIndex * pageSize + 1
  const lastVisibleRow = Math.min(
    props.rows.length,
    (activePageIndex + 1) * pageSize
  )

  return (
    <Card>
      <CardHeader className='border-b'>
        <CardTitle>{t('Current anomalies')}</CardTitle>
        <CardDescription>
          {t('Model-channel issues in the selected channel window')}
        </CardDescription>
        <CardAction className='flex items-center gap-2'>
          <span className='text-muted-foreground hidden text-xs sm:inline'>
            {t('Rows per page')}
          </span>
          <Select
            items={ANOMALY_PAGE_SIZES.map((size) => ({
              value: String(size),
              label: size,
            }))}
            value={String(pageSize)}
            onValueChange={(value) => {
              if (!value) return
              setPageSize(Number(value))
              setPageIndex(0)
            }}
          >
            <SelectTrigger size='sm' className='w-[76px]'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              <SelectGroup>
                {ANOMALY_PAGE_SIZES.map((size) => (
                  <SelectItem key={size} value={String(size)}>
                    {size}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </CardAction>
      </CardHeader>
      <CardContent className='px-0'>
        {props.loading ? (
          <div className='grid gap-2 px-4'>
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className='h-9 w-full' />
            ))}
          </div>
        ) : props.rows.length === 0 ? (
          <Empty className='min-h-40 border-0'>
            <EmptyMedia variant='icon'>
              <CheckCircle2 />
            </EmptyMedia>
            <EmptyHeader>
              <EmptyTitle>{t('No anomalies found')}</EmptyTitle>
              <EmptyDescription>
                {t('No warning or critical data matches the current filters')}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className='overflow-x-auto'>
            <table className='w-full min-w-[680px] text-sm'>
              <thead className='bg-muted/40 text-muted-foreground'>
                <tr>
                  <th className='px-4 py-2 text-left font-medium'>
                    {t('Model')}
                  </th>
                  <th className='px-4 py-2 text-left font-medium'>
                    {t('Channel')}
                  </th>
                  <th className='px-4 py-2 text-right font-medium'>
                    {t('Status')}
                  </th>
                  <th className='px-4 py-2 text-right font-medium'>
                    {t('Success rate')}
                  </th>
                  <th className='px-4 py-2 text-right font-medium'>
                    {t('Average latency')}
                  </th>
                  <th className='px-4 py-2 text-right font-medium'>
                    {t('TPS')}
                  </th>
                </tr>
              </thead>
              <tbody className='divide-y'>
                {visibleRows.map((row) => (
                  <tr key={row.key} className='hover:bg-muted/30'>
                    <td className='px-4 py-2'>
                      <button
                        type='button'
                        className='text-primary font-mono hover:underline'
                        onClick={() => props.onSelectModel(row.modelName)}
                      >
                        {row.modelName}
                      </button>
                    </td>
                    <td className='px-4 py-2 font-mono'>{row.channelName}</td>
                    <td className='px-4 py-2 text-right'>
                      <HealthBadge status={row.status} />
                    </td>
                    <td
                      className={cn(
                        'px-4 py-2 text-right font-mono tabular-nums',
                        healthStatusClass(row.status)
                      )}
                    >
                      {formatUptimePct(row.successRate)}
                    </td>
                    <td className='px-4 py-2 text-right font-mono tabular-nums'>
                      {formatLatency(row.latency)}
                    </td>
                    <td className='px-4 py-2 text-right font-mono tabular-nums'>
                      {formatThroughput(row.tps)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className='flex flex-wrap items-center justify-between gap-3 border-t px-4 py-3'>
              <span className='text-muted-foreground text-xs tabular-nums'>
                {t('Showing')} {firstVisibleRow}–{lastVisibleRow} {t('of')}{' '}
                {props.rows.length}
              </span>
              <div className='flex items-center gap-2'>
                <Button
                  variant='outline'
                  size='icon-sm'
                  onClick={() => setPageIndex(Math.max(0, activePageIndex - 1))}
                  disabled={activePageIndex === 0}
                  aria-label={t('Previous page')}
                >
                  <ChevronLeft />
                </Button>
                <span className='min-w-12 text-center font-mono text-xs tabular-nums'>
                  {activePageIndex + 1}/{pageCount}
                </span>
                <Button
                  variant='outline'
                  size='icon-sm'
                  onClick={() =>
                    setPageIndex(Math.min(pageCount - 1, activePageIndex + 1))
                  }
                  disabled={activePageIndex >= pageCount - 1}
                  aria-label={t('Next page')}
                >
                  <ChevronRight />
                </Button>
              </div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function ChannelHealthMatrix(props: {
  rows: PerfChannelModelSummary[]
  loading: boolean
  onSelectModel: (model: string) => void
}) {
  const { t } = useTranslation()
  const [channelPage, setChannelPage] = useState(0)
  const models = useMemo(
    () => [...new Set(props.rows.map((row) => row.model_name))].sort(),
    [props.rows]
  )
  const channels = useMemo(() => {
    const values = new Map<number, string>()
    for (const row of props.rows) {
      values.set(row.channel_id, row.channel_name || `#${row.channel_id}`)
    }
    return [...values.entries()].map(([id, name]) => ({ id, name }))
  }, [props.rows])
  const channelPageCount = Math.max(
    1,
    Math.ceil(channels.length / MATRIX_CHANNELS_PER_PAGE)
  )
  const activeChannelPage = Math.min(channelPage, channelPageCount - 1)
  const visibleChannels = channels.slice(
    activeChannelPage * MATRIX_CHANNELS_PER_PAGE,
    (activeChannelPage + 1) * MATRIX_CHANNELS_PER_PAGE
  )
  const firstVisibleChannel =
    channels.length === 0 ? 0 : activeChannelPage * MATRIX_CHANNELS_PER_PAGE + 1
  const lastVisibleChannel = Math.min(
    channels.length,
    (activeChannelPage + 1) * MATRIX_CHANNELS_PER_PAGE
  )
  const cells = useMemo(
    () =>
      new Map(
        props.rows.map((row) => [`${row.model_name}:${row.channel_id}`, row])
      ),
    [props.rows]
  )

  return (
    <Card>
      <CardHeader className='border-b'>
        <CardTitle>{t('Model × channel health matrix')}</CardTitle>
        <CardDescription>
          {t('Cells show success rate for the current channel snapshot window')}
        </CardDescription>
        {channels.length > 0 && (
          <CardAction className='flex items-center gap-2'>
            <span className='text-muted-foreground hidden text-xs tabular-nums sm:inline'>
              {t('Channels')} {firstVisibleChannel}–{lastVisibleChannel} /{' '}
              {channels.length}
            </span>
            <div className='flex items-center gap-1'>
              <Button
                variant='outline'
                size='icon-sm'
                onClick={() => setChannelPage((page) => Math.max(0, page - 1))}
                disabled={activeChannelPage === 0}
                aria-label={t('Previous')}
              >
                <ChevronLeft />
              </Button>
              <span className='min-w-12 text-center font-mono text-xs tabular-nums'>
                {activeChannelPage + 1}/{channelPageCount}
              </span>
              <Button
                variant='outline'
                size='icon-sm'
                onClick={() =>
                  setChannelPage((page) =>
                    Math.min(channelPageCount - 1, page + 1)
                  )
                }
                disabled={activeChannelPage >= channelPageCount - 1}
                aria-label={t('Next')}
              >
                <ChevronRight />
              </Button>
            </div>
          </CardAction>
        )}
      </CardHeader>
      <CardContent className='px-0'>
        {props.loading ? (
          <Skeleton className='mx-4 h-64 w-auto' />
        ) : models.length === 0 ? (
          <Empty className='min-h-48 border-0'>
            <EmptyHeader>
              <EmptyTitle>
                {t('No channel health data in this window')}
              </EmptyTitle>
              <EmptyDescription>
                {t('Only combinations that received calls are returned')}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className='max-h-[520px] overflow-auto'>
            <table className='w-full min-w-[976px] table-fixed border-collapse text-xs'>
              <thead className='bg-background sticky top-0 z-20 shadow-sm'>
                <tr>
                  <th className='bg-background sticky left-0 z-30 min-w-52 border-r px-3 py-2 text-left font-medium'>
                    {t('Model')}
                  </th>
                  {visibleChannels.map((channel) => (
                    <th
                      key={channel.id}
                      className='min-w-32 px-2 py-2 text-center font-medium'
                    >
                      <div className='max-w-40 truncate' title={channel.name}>
                        {channel.name}
                      </div>
                      <div className='text-muted-foreground font-mono'>
                        #{channel.id}
                      </div>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className='divide-y'>
                {models.map((model) => (
                  <tr key={model}>
                    <td className='bg-card sticky left-0 z-10 border-r px-3 py-2'>
                      <button
                        type='button'
                        className='text-primary max-w-52 truncate font-mono hover:underline'
                        onClick={() => props.onSelectModel(model)}
                        title={model}
                      >
                        {model}
                      </button>
                    </td>
                    {visibleChannels.map((channel) => {
                      const row = cells.get(`${model}:${channel.id}`)
                      if (!row) {
                        return (
                          <td
                            key={channel.id}
                            className='bg-muted/20 text-muted-foreground px-2 py-2 text-center'
                          >
                            {t('No data')}
                          </td>
                        )
                      }
                      const status = getHealthStatus(row)
                      return (
                        <td key={channel.id} className='px-2 py-2 text-center'>
                          <span
                            className={cn(
                              'inline-flex min-w-20 justify-center rounded-md px-2 py-1 font-mono font-semibold tabular-nums',
                              healthStatusBackground(status)
                            )}
                          >
                            {formatUptimePct(row.success_rate)}
                          </span>
                        </td>
                      )
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function ChannelRankings(props: {
  rows: PerfChannelModelSummary[]
  loading: boolean
  onSelectModel: (model: string) => void
}) {
  const { t } = useTranslation()
  const lowestSuccess = useMemo(
    () =>
      [...props.rows]
        .sort((a, b) => a.success_rate - b.success_rate)
        .slice(0, 10),
    [props.rows]
  )
  const highestLatency = useMemo(
    () =>
      [...props.rows]
        .filter((row) => row.avg_latency_ms > 0)
        .sort((a, b) => b.avg_latency_ms - a.avg_latency_ms)
        .slice(0, 10),
    [props.rows]
  )
  return (
    <div className='grid gap-4 xl:grid-cols-2'>
      <ChannelRankingCard
        title={t('Lowest channel success rates')}
        rows={lowestSuccess}
        loading={props.loading}
        metric={(row) => formatUptimePct(row.success_rate)}
        onSelectModel={props.onSelectModel}
      />
      <ChannelRankingCard
        title={t('Highest channel latency')}
        rows={highestLatency}
        loading={props.loading}
        metric={(row) => formatLatency(row.avg_latency_ms)}
        onSelectModel={props.onSelectModel}
      />
    </div>
  )
}

function ChannelRankingCard(props: {
  title: string
  rows: PerfChannelModelSummary[]
  loading: boolean
  metric: (row: PerfChannelModelSummary) => string
  onSelectModel: (model: string) => void
}) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader className='border-b'>
        <CardTitle>{props.title}</CardTitle>
      </CardHeader>
      <CardContent className='px-0'>
        {props.loading ? (
          <div className='grid gap-2 px-4'>
            {Array.from({ length: 5 }).map((_, index) => (
              <Skeleton key={index} className='h-8 w-full' />
            ))}
          </div>
        ) : props.rows.length === 0 ? (
          <Empty className='min-h-40 border-0'>
            <EmptyHeader>
              <EmptyTitle>{t('No health data in this window')}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className='divide-y'>
            {props.rows.map((row, index) => (
              <div
                key={`${row.model_name}-${row.channel_id}`}
                className='flex items-center gap-3 px-4 py-2'
              >
                <span className='text-muted-foreground w-5 font-mono text-xs'>
                  {index + 1}
                </span>
                <div className='min-w-0 flex-1'>
                  <button
                    type='button'
                    onClick={() => props.onSelectModel(row.model_name)}
                    className='text-primary block max-w-full truncate font-mono text-sm hover:underline'
                  >
                    {row.model_name}
                  </button>
                  <span className='text-muted-foreground block truncate text-xs'>
                    {row.channel_name || `#${row.channel_id}`} · #
                    {row.channel_id}
                  </span>
                </div>
                <div className='text-right'>
                  <div
                    className={cn(
                      'font-mono font-semibold tabular-nums',
                      healthStatusClass(getHealthStatus(row))
                    )}
                  >
                    {props.metric(row)}
                  </div>
                  <HealthBadge status={getHealthStatus(row)} />
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
