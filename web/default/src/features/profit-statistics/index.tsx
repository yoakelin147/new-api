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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  CalendarDays,
  Download,
  Eye,
  Pencil,
  RotateCcw,
  Save,
  Search,
  Settings2,
  Trash2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  formatNumber,
  formatTimestampForInput,
  formatTimestampToDate,
  parseTimestampFromInput,
} from '@/lib/format'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Empty, EmptyHeader, EmptyTitle } from '@/components/ui/empty'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { ComboboxInput } from '@/components/ui/combobox-input'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { StaticDataTable } from '@/components/data-table/static/static-data-table'
import { Main } from '@/components/layout'
import { getChannels, getGroups } from '@/features/channels/api'
import { ModelBadge } from '@/features/usage-logs/components/model-badge'
import {
  deleteChannelProfitRatio,
  getChannelProfitRatios,
  getProfitStatistics,
  saveChannelProfitRatio,
} from './api'
import type {
  ChannelProfitRatio,
  ChannelProfitRatioInput,
  ProfitCostStatus,
  ProfitStatisticsData,
  ProfitStatisticsFilters,
  ProfitStatisticsPeriod,
  ProfitStatisticsRow,
  ProfitStatisticsType,
} from './types'

const DEFAULT_PAGE_SIZE = 20

const DEFAULT_DATA: ProfitStatisticsData = {
  page: 1,
  page_size: DEFAULT_PAGE_SIZE,
  total: 0,
  items: [],
  summary: {
    request_count: 0,
    prompt_tokens: 0,
    completion_tokens: 0,
    total_tokens: 0,
    customer_amount: 0,
    original_amount: 0,
    upstream_cost: 0,
    profit: 0,
    profit_rate: 0,
    unconfigured_rows: 0,
    unconfigured_amount: 0,
    estimated_rows: 0,
  },
  period: 'today',
  start_timestamp: 0,
  end_timestamp: 0,
}

const PERIOD_OPTIONS: Array<{ value: ProfitStatisticsPeriod; label: string }> =
  [
    { value: 'today', label: 'Today' },
    { value: 'week', label: 'This Week Range' },
    { value: 'month', label: 'This Month Range' },
    { value: 'last_month', label: 'Last Month Range' },
  ]

const TYPE_OPTIONS: Array<{ value: ProfitStatisticsType; label: string }> = [
  { value: 'consume', label: 'Consumption' },
  { value: 'all', label: 'All Types' },
  { value: 'refund', label: 'Refund' },
]

const EMPTY_RATIO_FORM: ChannelProfitRatioInput = {
  channel_id: 0,
  model_name: '',
  group: '',
  upstream_ratio: 0,
  enabled: true,
  remark: '',
}

type ProfitStatisticsColumnId =
  | 'date'
  | 'username'
  | 'group'
  | 'channel'
  | 'model'
  | 'tokens'
  | 'requests'
  | 'customerRatio'
  | 'upstreamRatio'
  | 'customerAmount'
  | 'originalAmount'
  | 'upstreamCost'
  | 'profit'
  | 'profitRate'
  | 'status'

type ProfitStatisticsColumnVisibility = Record<ProfitStatisticsColumnId, boolean>

const DEFAULT_COLUMN_VISIBILITY: ProfitStatisticsColumnVisibility = {
  date: true,
  username: true,
  group: true,
  channel: true,
  model: true,
  tokens: true,
  requests: true,
  customerRatio: true,
  upstreamRatio: true,
  customerAmount: true,
  originalAmount: true,
  upstreamCost: true,
  profit: true,
  profitRate: true,
  status: true,
}

const COLUMN_TOGGLE_OPTIONS: Array<{
  id: ProfitStatisticsColumnId
  label: string
}> = [
  { id: 'date', label: 'Date' },
  { id: 'username', label: 'Username' },
  { id: 'group', label: 'Group' },
  { id: 'channel', label: 'Channel' },
  { id: 'model', label: 'Model' },
  { id: 'tokens', label: 'Tokens' },
  { id: 'requests', label: 'Requests' },
  { id: 'customerRatio', label: 'Customer Ratio' },
  { id: 'upstreamRatio', label: 'Upstream Ratio' },
  { id: 'customerAmount', label: 'Customer Consumption' },
  { id: 'originalAmount', label: 'Original Price' },
  { id: 'upstreamCost', label: 'Upstream Cost' },
  { id: 'profit', label: 'Profit' },
  { id: 'profitRate', label: 'Profit Rate' },
  { id: 'status', label: 'Cost Status' },
]

function defaultFilters(): ProfitStatisticsFilters {
  return {
    period: 'today',
    type: 'consume',
    model: '',
    group: '',
    username: '',
    channel: '',
    start_timestamp: undefined,
    end_timestamp: undefined,
  }
}

function getPeriodRange(period: ProfitStatisticsPeriod) {
  const now = new Date()
  const start = new Date(now)
  start.setHours(0, 0, 0, 0)
  let end = now

  if (period === 'week') {
    const offset = (now.getDay() + 6) % 7
    start.setDate(now.getDate() - offset)
  } else if (period === 'month') {
    start.setDate(1)
  } else if (period === 'last_month') {
    start.setDate(1)
    start.setMonth(now.getMonth() - 1)
    end = new Date(now.getFullYear(), now.getMonth(), 1, 0, 0, 0, 0)
    end = new Date(end.getTime() - 1000)
  }

  return {
    start: Math.floor(start.getTime() / 1000),
    end: Math.floor(end.getTime() / 1000),
  }
}

function getFilterRange(filters: ProfitStatisticsFilters) {
  if (
    filters.period === 'custom' &&
    filters.start_timestamp &&
    filters.end_timestamp
  ) {
    return {
      start: filters.start_timestamp,
      end: filters.end_timestamp,
    }
  }
  return getPeriodRange(filters.period)
}

function formatRangeLabel(start: number, end: number) {
  if (!start || !end) return '-'
  return `${formatTimestampToDate(start).slice(0, 16)} ~ ${formatTimestampToDate(end).slice(0, 16)}`
}

function formatMoney(value: number | null | undefined) {
  if (value == null || !Number.isFinite(value)) return '-'
  return `$${value.toFixed(4)}`
}

function formatPercent(value: number | null | undefined) {
  if (value == null || !Number.isFinite(value)) return '-'
  return `${(value * 100).toFixed(2)}%`
}

function formatRatio(value: number | null | undefined) {
  if (value == null || !Number.isFinite(value)) return '-'
  return `${value.toFixed(2)}x`
}

function costStatusLabel(t: (key: string) => string, status: ProfitCostStatus) {
  switch (status) {
    case 'configured':
      return t('Configured')
    case 'estimated':
      return t('Estimated')
    default:
      return t('Not Configured')
  }
}

function csvEscape(value: string | number | null | undefined) {
  const text = String(value ?? '')
  if (!/[",\n]/.test(text)) return text
  return `"${text.replaceAll('"', '""')}"`
}

function downloadCsv(
  filename: string,
  rows: Array<Array<string | number | null>>
) {
  const content = rows.map((row) => row.map(csvEscape).join(',')).join('\n')
  const blob = new Blob([`\uFEFF${content}`], {
    type: 'text/csv;charset=utf-8',
  })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

function buildExportFilename() {
  const date = new Date()
  const pad = (value: number) => String(value).padStart(2, '0')
  return `profit_statistics_${date.getFullYear()}${pad(date.getMonth() + 1)}${pad(date.getDate())}_${pad(date.getHours())}${pad(date.getMinutes())}${pad(date.getSeconds())}.csv`
}

function DateRangePicker(props: {
  period: ProfitStatisticsPeriod
  start: number
  end: number
  onPeriodChange: (period: ProfitStatisticsPeriod) => void
  onRangeChange: (start: number, end: number) => void
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [draftStart, setDraftStart] = useState('')
  const [draftEnd, setDraftEnd] = useState('')

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setDraftStart(formatTimestampForInput(props.start))
      setDraftEnd(formatTimestampForInput(props.end))
    }
    setOpen(nextOpen)
  }

  const applyCustomRange = () => {
    const start = parseTimestampFromInput(draftStart)
    const end = parseTimestampFromInput(draftEnd)
    if (start <= 0 || end <= 0 || end < start) {
      toast.error(t('End time must be later than start time'))
      return
    }
    props.onRangeChange(start, end)
    setOpen(false)
  }

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger
        render={
          <Button
            type='button'
            variant='outline'
            className='h-9 min-w-0 flex-1 justify-start rounded-full px-3 font-normal tabular-nums lg:min-w-[480px]'
          />
        }
      >
        <CalendarDays data-icon='inline-start' />
        <span className='truncate'>
          {formatRangeLabel(props.start, props.end)}
        </span>
      </PopoverTrigger>
      <PopoverContent align='start' className='w-[min(430px,calc(100vw-2rem))]'>
        <div className='flex flex-col gap-3'>
          <div className='grid gap-2 sm:grid-cols-2'>
            <Field>
              <FieldLabel>{t('Start Time')}</FieldLabel>
              <Input
                type='datetime-local'
                value={draftStart}
                onChange={(event) => setDraftStart(event.target.value)}
                className='h-8 font-mono text-xs tabular-nums'
              />
            </Field>
            <Field>
              <FieldLabel>{t('End Time')}</FieldLabel>
              <Input
                type='datetime-local'
                value={draftEnd}
                onChange={(event) => setDraftEnd(event.target.value)}
                className='h-8 font-mono text-xs tabular-nums'
              />
            </Field>
          </div>
          <div className='grid grid-cols-2 gap-2 sm:grid-cols-4'>
            {PERIOD_OPTIONS.map((option) => (
              <Button
                key={option.value}
                type='button'
                variant={
                  props.period === option.value ? 'default' : 'secondary'
                }
                size='sm'
                className='h-8'
                onClick={() => {
                  props.onPeriodChange(option.value)
                  setOpen(false)
                }}
              >
                {t(option.label)}
              </Button>
            ))}
          </div>
          <Button
            type='button'
            variant={props.period === 'custom' ? 'default' : 'outline'}
            size='sm'
            className='h-8'
            onClick={applyCustomRange}
          >
            {t('Custom Range')}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  )
}

function SummaryBadge(props: {
  label: string
  value: string
  loading?: boolean
}) {
  return (
    <div className='bg-muted/60 flex min-w-[150px] flex-col gap-1 rounded-lg border px-3 py-2'>
      <span className='text-muted-foreground text-xs'>{props.label}</span>
      {props.loading ? (
        <Skeleton className='h-5 w-24' />
      ) : (
        <span className='text-sm font-semibold tabular-nums'>
          {props.value}
        </span>
      )}
    </div>
  )
}

function ColumnVisibilityMenu(props: {
  visibility: ProfitStatisticsColumnVisibility
  onVisibilityChange: (visibility: ProfitStatisticsColumnVisibility) => void
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  const toggleColumn = (id: ProfitStatisticsColumnId, checked: boolean) => {
    props.onVisibilityChange({
      ...props.visibility,
      [id]: checked,
    })
  }

  return (
    <DropdownMenu modal={false} open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger
        render={
          <Button
            variant='outline'
            size='sm'
            className='rounded-full'
            onMouseEnter={() => setOpen(true)}
            onFocus={() => setOpen(true)}
          />
        }
      >
        <Eye data-icon='inline-start' />
        {t('View')}
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align='end'
        className='w-48'
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
      >
        <DropdownMenuGroup>
          <DropdownMenuLabel>{t('Toggle columns')}</DropdownMenuLabel>
          {COLUMN_TOGGLE_OPTIONS.map((column) => (
            <DropdownMenuCheckboxItem
              key={column.id}
              checked={props.visibility[column.id]}
              onCheckedChange={(value) => toggleColumn(column.id, !!value)}
            >
              {t(column.label)}
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function ProfitStatisticsTable(props: {
  data: ProfitStatisticsRow[]
  loading: boolean
  columnVisibility: ProfitStatisticsColumnVisibility
}) {
  const { t } = useTranslation()
  const columns = useMemo(
    () =>
      [
        {
          id: 'date',
          header: t('Date'),
          cell: (row: ProfitStatisticsRow) => row.date,
        },
        {
          id: 'username',
          header: t('Username'),
          cell: (row: ProfitStatisticsRow) => row.username || t('All'),
        },
        {
          id: 'group',
          header: t('Group'),
          cell: (row: ProfitStatisticsRow) => row.group || '-',
        },
        {
          id: 'channel',
          header: t('Channel'),
          cell: (row: ProfitStatisticsRow) =>
            row.channel_name
              ? `#${row.channel_id} ${row.channel_name}`
              : `#${row.channel_id}`,
        },
        {
          id: 'model',
          header: t('Model'),
          cell: (row: ProfitStatisticsRow) => (
            <ModelBadge modelName={row.model_name} />
          ),
        },
        {
          id: 'tokens',
          header: t('Tokens'),
          className: 'text-right',
          cellClassName: 'text-right tabular-nums',
          cell: (row: ProfitStatisticsRow) => formatNumber(row.total_tokens),
        },
        {
          id: 'requests',
          header: t('Requests'),
          className: 'text-right',
          cellClassName: 'text-right tabular-nums',
          cell: (row: ProfitStatisticsRow) => formatNumber(row.request_count),
        },
        {
          id: 'customerRatio',
          header: t('Customer Ratio'),
          className: 'text-right',
          cellClassName: 'text-right tabular-nums',
          cell: (row: ProfitStatisticsRow) => formatRatio(row.customer_ratio),
        },
        {
          id: 'upstreamRatio',
          header: t('Upstream Ratio'),
          className: 'text-right',
          cellClassName: 'text-right tabular-nums',
          cell: (row: ProfitStatisticsRow) => formatRatio(row.upstream_ratio),
        },
        {
          id: 'customerAmount',
          header: t('Customer Consumption'),
          className: 'text-right',
          cellClassName: 'text-right tabular-nums font-medium',
          cell: (row: ProfitStatisticsRow) => formatMoney(row.customer_amount),
        },
        {
          id: 'originalAmount',
          header: t('Original Price'),
          className: 'text-right',
          cellClassName: 'text-right tabular-nums',
          cell: (row: ProfitStatisticsRow) => formatMoney(row.original_amount),
        },
        {
          id: 'upstreamCost',
          header: t('Upstream Cost'),
          className: 'text-right',
          cellClassName: 'text-right tabular-nums',
          cell: (row: ProfitStatisticsRow) => formatMoney(row.upstream_cost),
        },
        {
          id: 'profit',
          header: t('Profit'),
          className: 'text-right',
          cellClassName: (row: ProfitStatisticsRow) =>
            cn(
              'text-right tabular-nums font-semibold',
              row.profit != null && row.profit < 0 && 'text-destructive'
            ),
          cell: (row: ProfitStatisticsRow) => formatMoney(row.profit),
        },
        {
          id: 'profitRate',
          header: t('Profit Rate'),
          className: 'text-right',
          cellClassName: 'text-right tabular-nums',
          cell: (row: ProfitStatisticsRow) => formatPercent(row.profit_rate),
        },
        {
          id: 'status',
          header: t('Cost Status'),
          cell: (row: ProfitStatisticsRow) => (
            <Badge
              variant={
                row.cost_status === 'configured' ? 'secondary' : 'outline'
              }
            >
              {costStatusLabel(t, row.cost_status)}
            </Badge>
          ),
        },
      ].filter(
        (column) =>
          props.columnVisibility[column.id as ProfitStatisticsColumnId]
      ),
    [props.columnVisibility, t]
  )

  if (props.loading) {
    return (
      <div className='flex flex-col gap-2 p-3'>
        {Array.from({ length: 8 }).map((_, index) => (
          <Skeleton key={index} className='h-10 w-full' />
        ))}
      </div>
    )
  }

  return (
    <StaticDataTable
      columns={columns}
      data={props.data}
      getRowKey={(row) =>
        `${row.date}-${row.username}-${row.group}-${row.channel_id}-${row.model_name}`
      }
      emptyContent={
        <Empty>
          <EmptyHeader>
            <EmptyTitle>{t('No Profit Statistics Found')}</EmptyTitle>
          </EmptyHeader>
        </Empty>
      }
    />
  )
}

function RatioSettingsDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [form, setForm] = useState<ChannelProfitRatioInput>(EMPTY_RATIO_FORM)
  const { data, isLoading } = useQuery({
    queryKey: ['channel-profit-ratios'],
    queryFn: getChannelProfitRatios,
    enabled: props.open,
  })
  const { data: channelsResponse } = useQuery({
    queryKey: ['profit-statistics', 'channels'],
    queryFn: () => getChannels({ p: 0, page_size: 1000, id_sort: true }),
    staleTime: 300_000,
    enabled: props.open,
  })
  const { data: groupsResponse } = useQuery({
    queryKey: ['profit-statistics', 'groups'],
    queryFn: getGroups,
    staleTime: 300_000,
    enabled: props.open,
  })
  const ratios = data?.data || []
  const channels = useMemo(() => {
    const items = channelsResponse?.success
      ? (channelsResponse.data?.items ?? [])
      : []
    return [...items].sort((left, right) => left.id - right.id)
  }, [channelsResponse])
  const channelOptions = useMemo(
    () =>
      channels.map((channel) => ({
        value: String(channel.id),
        label: `#${channel.id} ${channel.name}`,
      })),
    [channels]
  )
  const selectedChannel = channels.find(
    (channel) => channel.id === form.channel_id
  )
  const modelOptions = useMemo(() => {
    const models = new Set<string>()
    const sourceChannels = selectedChannel ? [selectedChannel] : channels
    for (const channel of sourceChannels) {
      for (const model of channel.models.split(',')) {
        const trimmed = model.trim()
        if (trimmed) models.add(trimmed)
      }
    }
    return [...models].sort().map((model) => ({
      value: model,
      label: model,
    }))
  }, [channels, selectedChannel])
  const groupOptions = useMemo(() => {
    const groups = new Set<string>()
    for (const group of groupsResponse?.data ?? []) {
      if (group.trim()) groups.add(group.trim())
    }
    const sourceChannels = selectedChannel ? [selectedChannel] : channels
    for (const channel of sourceChannels) {
      for (const group of channel.group.split(',')) {
        const trimmed = group.trim()
        if (trimmed) groups.add(trimmed)
      }
    }
    return [...groups].sort().map((group) => ({
      value: group,
      label: group,
    }))
  }, [channels, groupsResponse?.data, selectedChannel])
  const resetForm = () => setForm(EMPTY_RATIO_FORM)
  const saveMutation = useMutation({
    mutationFn: saveChannelProfitRatio,
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Save failed'))
        return
      }
      toast.success(t('Saved'))
      resetForm()
      queryClient.invalidateQueries({ queryKey: ['channel-profit-ratios'] })
      queryClient.invalidateQueries({ queryKey: ['profit-statistics'] })
    },
  })
  const deleteMutation = useMutation({
    mutationFn: deleteChannelProfitRatio,
    onSuccess: (result) => {
      if (!result.success) {
        toast.error(result.message || t('Delete failed'))
        return
      }
      toast.success(t('Deleted'))
      queryClient.invalidateQueries({ queryKey: ['channel-profit-ratios'] })
      queryClient.invalidateQueries({ queryKey: ['profit-statistics'] })
    },
  })

  const submit = () => {
    if (!form.channel_id || form.channel_id <= 0) {
      toast.error(t('Channel ID is required'))
      return
    }
    if (form.upstream_ratio < 0) {
      toast.error(t('Upstream ratio must be non-negative'))
      return
    }
    saveMutation.mutate(form)
  }

  const editRatio = (row: ChannelProfitRatio) => {
    setForm({
      id: row.id,
      channel_id: row.channel_id,
      model_name: row.model_name || '',
      group: row.group || '',
      upstream_ratio: row.upstream_ratio,
      enabled: row.enabled,
      remark: row.remark || '',
    })
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-4xl'>
        <DialogHeader>
          <DialogTitle>{t('Upstream Ratio Settings')}</DialogTitle>
          <DialogDescription>
            {t('Configure channel upstream cost ratios for profit statistics.')}
          </DialogDescription>
        </DialogHeader>

        <div className='grid gap-4 lg:grid-cols-[280px_1fr]'>
          <FieldGroup>
            <Field>
              <FieldLabel>{t('Channel ID')}</FieldLabel>
              <ComboboxInput
                options={channelOptions}
                value={form.channel_id ? String(form.channel_id) : ''}
                placeholder={t('Select or enter channel ID')}
                emptyText='No channels found'
                allowCustomValue
                onValueChange={(value) =>
                  setForm((current) => ({
                    ...current,
                    channel_id: Number(value) || 0,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel>{t('Model')}</FieldLabel>
              <ComboboxInput
                options={modelOptions}
                value={form.model_name || ''}
                placeholder={t('Optional')}
                emptyText='No model found.'
                allowCustomValue
                onValueChange={(value) =>
                  setForm((current) => ({
                    ...current,
                    model_name: value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel>{t('Group')}</FieldLabel>
              <ComboboxInput
                options={groupOptions}
                value={form.group || ''}
                placeholder={t('Optional')}
                emptyText='No group found.'
                allowCustomValue
                onValueChange={(value) =>
                  setForm((current) => ({
                    ...current,
                    group: value,
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel>{t('Upstream Ratio')}</FieldLabel>
              <Input
                type='number'
                min={0}
                step='0.01'
                value={form.upstream_ratio}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    upstream_ratio: Number(event.target.value),
                  }))
                }
              />
            </Field>
            <Field>
              <FieldLabel>{t('Remark')}</FieldLabel>
              <Input
                value={form.remark || ''}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    remark: event.target.value,
                  }))
                }
              />
            </Field>
            <Button
              type='button'
              onClick={submit}
              disabled={saveMutation.isPending}
            >
              <Save data-icon='inline-start' />
              {form.id ? t('Update') : t('Save')}
            </Button>
            {form.id ? (
              <Button type='button' variant='outline' onClick={resetForm}>
                {t('Cancel Edit')}
              </Button>
            ) : null}
          </FieldGroup>

          <div className='min-h-[320px] overflow-auto rounded-lg border'>
            <StaticDataTable
              data={ratios}
              columns={[
                {
                  id: 'channel',
                  header: t('Channel'),
                  cell: (row: ChannelProfitRatio) =>
                    row.channel_name
                      ? `#${row.channel_id} ${row.channel_name}`
                      : `#${row.channel_id}`,
                },
                {
                  id: 'model',
                  header: t('Model'),
                  cell: (row: ChannelProfitRatio) => row.model_name || t('All'),
                },
                {
                  id: 'group',
                  header: t('Group'),
                  cell: (row: ChannelProfitRatio) => row.group || t('All'),
                },
                {
                  id: 'ratio',
                  header: t('Upstream Ratio'),
                  className: 'text-right',
                  cellClassName: 'text-right tabular-nums',
                  cell: (row: ChannelProfitRatio) =>
                    formatRatio(row.upstream_ratio),
                },
                {
                  id: 'action',
                  header: t('Operation'),
                  className: 'text-right',
                  cellClassName: 'text-right',
                  cell: (row: ChannelProfitRatio) => (
                    <div className='flex justify-end gap-1'>
                      <Button
                        variant='ghost'
                        size='icon-sm'
                        onClick={() => editRatio(row)}
                        title={t('Edit')}
                      >
                        <Pencil />
                      </Button>
                      <Button
                        variant='ghost'
                        size='icon-sm'
                        onClick={() => deleteMutation.mutate(row.id)}
                        title={t('Delete')}
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  ),
                },
              ]}
              empty={isLoading ? false : ratios.length === 0}
              emptyContent={
                <Empty>
                  <EmptyHeader>
                    <EmptyTitle>{t('No upstream ratio settings')}</EmptyTitle>
                  </EmptyHeader>
                </Empty>
              }
            />
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

export function ProfitStatistics() {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const [draftFilters, setDraftFilters] = useState(defaultFilters)
  const [filters, setFilters] = useState(defaultFilters)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [columnVisibility, setColumnVisibility] =
    useState<ProfitStatisticsColumnVisibility>(DEFAULT_COLUMN_VISIBILITY)
  const draftRange = getFilterRange(draftFilters)

  const { data, isLoading, isFetching } = useQuery({
    queryKey: ['profit-statistics', page, filters],
    queryFn: async () => {
      const result = await getProfitStatistics({
        page,
        pageSize: DEFAULT_PAGE_SIZE,
        filters,
      })
      if (!result.success) {
        toast.error(result.message || t('Failed to load profit statistics'))
        return DEFAULT_DATA
      }
      return result.data || DEFAULT_DATA
    },
    placeholderData: (previousData) => previousData,
  })

  const stats = data || DEFAULT_DATA
  const loadingContent = isLoading || (isFetching && !data)

  const updateDraft = (
    key: keyof ProfitStatisticsFilters,
    value: string | ProfitStatisticsPeriod | ProfitStatisticsType
  ) => {
    setDraftFilters((current) => ({ ...current, [key]: value }))
  }

  const handlePeriodChange = (period: ProfitStatisticsPeriod) => {
    setDraftFilters((current) => ({
      ...current,
      period,
      start_timestamp: undefined,
      end_timestamp: undefined,
    }))
  }

  const handleCustomRangeChange = (start: number, end: number) => {
    setDraftFilters((current) => ({
      ...current,
      period: 'custom',
      start_timestamp: start,
      end_timestamp: end,
    }))
  }

  const applyFilters = () => {
    setPage(1)
    setFilters({ ...draftFilters })
  }

  const resetFilters = () => {
    const next = defaultFilters()
    setPage(1)
    setDraftFilters(next)
    setFilters(next)
  }

  const exportCsv = () => {
    if (stats.items.length === 0) {
      toast.warning(t('No data to export'))
      return
    }
    downloadCsv(buildExportFilename(), [
      [
        '日期',
        '用户名',
        '分组',
        '渠道',
        '模型',
        '请求数',
        '提示词 Token',
        '补全 Token',
        '总 Token',
        '客户倍率',
        '上游倍率',
        '客户消费',
        '折前原价',
        '上游成本',
        '利润',
        '利润率',
        '成本状态',
      ],
      ...stats.items.map((row) => [
        row.date,
        row.username || '全部',
        row.group,
        row.channel_name
          ? `#${row.channel_id} ${row.channel_name}`
          : `#${row.channel_id}`,
        row.model_name,
        row.request_count,
        row.prompt_tokens,
        row.completion_tokens,
        row.total_tokens,
        row.customer_ratio,
        row.upstream_ratio,
        row.customer_amount,
        row.original_amount,
        row.upstream_cost,
        row.profit,
        row.profit_rate == null ? null : row.profit_rate * 100,
        costStatusLabel(t, row.cost_status),
      ]),
    ])
    toast.success(t('Export completed'))
  }

  const activeTypeLabel =
    TYPE_OPTIONS.find((option) => option.value === draftFilters.type)?.label ||
    'Consumption'

  return (
    <Main className='bg-muted/30 p-3 sm:p-4'>
      <div className='bg-card flex min-h-0 flex-1 flex-col rounded-3xl border p-4 shadow-xs sm:p-6'>
        <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
          <h1 className='text-2xl font-bold tracking-tight'>
            {t('Profit Statistics')}
          </h1>
          <Button
            type='button'
            variant='outline'
            className='rounded-full'
            onClick={() => setSettingsOpen(true)}
          >
            <Settings2 data-icon='inline-start' />
            {t('Upstream Ratio Settings')}
          </Button>
        </div>

        <div className='mt-5 rounded-2xl border p-3'>
          <div className='flex flex-col gap-3'>
            <div className='flex flex-col gap-2 lg:flex-row'>
              <DateRangePicker
                period={draftFilters.period}
                start={draftRange.start}
                end={draftRange.end}
                onPeriodChange={handlePeriodChange}
                onRangeChange={handleCustomRangeChange}
              />
              <Select
                value={draftFilters.type}
                onValueChange={(value) =>
                  updateDraft(
                    'type',
                    (value || 'consume') as ProfitStatisticsType
                  )
                }
              >
                <SelectTrigger className='h-9 flex-1 rounded-full lg:max-w-[260px]'>
                  <SelectValue>{t(activeTypeLabel)}</SelectValue>
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {TYPE_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {t(option.label)}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>

            <div className='grid gap-2 sm:grid-cols-2 lg:grid-cols-4'>
              <Input
                placeholder={t('Model')}
                value={draftFilters.model}
                onChange={(event) => updateDraft('model', event.target.value)}
                className='h-9 rounded-full'
              />
              <Input
                placeholder={t('Group')}
                value={draftFilters.group}
                onChange={(event) => updateDraft('group', event.target.value)}
                className='h-9 rounded-full'
              />
              <Input
                placeholder={t('Username')}
                value={draftFilters.username}
                onChange={(event) =>
                  updateDraft('username', event.target.value)
                }
                className='h-9 rounded-full'
              />
              <Input
                placeholder={t('Channel ID')}
                value={draftFilters.channel}
                onChange={(event) => updateDraft('channel', event.target.value)}
                className='h-9 rounded-full'
              />
            </div>

            <div className='flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between'>
              <div className='flex flex-wrap items-center gap-2'>
                <SummaryBadge
                  label={t('Customer Consumption')}
                  value={formatMoney(stats.summary.customer_amount)}
                  loading={loadingContent}
                />
                <SummaryBadge
                  label={t('Upstream Cost')}
                  value={formatMoney(stats.summary.upstream_cost)}
                  loading={loadingContent}
                />
                <SummaryBadge
                  label={t('Profit')}
                  value={formatMoney(stats.summary.profit)}
                  loading={loadingContent}
                />
                <SummaryBadge
                  label={t('Profit Rate')}
                  value={formatPercent(stats.summary.profit_rate)}
                  loading={loadingContent}
                />
                <SummaryBadge
                  label={t('Requests')}
                  value={formatNumber(stats.summary.request_count)}
                  loading={loadingContent}
                />
              </div>
              <div className='flex flex-wrap justify-start gap-2 lg:justify-end'>
                <Button
                  variant='outline'
                  size='sm'
                  className='rounded-full'
                  onClick={exportCsv}
                >
                  <Download data-icon='inline-start' />
                  {t('Export CSV')}
                </Button>
                <Button
                  variant='outline'
                  size='sm'
                  className='rounded-full'
                  onClick={resetFilters}
                >
                  <RotateCcw data-icon='inline-start' />
                  {t('Reset')}
                </Button>
                <Button
                  size='sm'
                  className='rounded-full'
                  onClick={applyFilters}
                >
                  <Search data-icon='inline-start' />
                  {t('Search')}
                </Button>
                <ColumnVisibilityMenu
                  visibility={columnVisibility}
                  onVisibilityChange={setColumnVisibility}
                />
              </div>
            </div>
          </div>
        </div>

        <div className='mt-4 min-h-0 flex-1 overflow-auto rounded-2xl border'>
          <ProfitStatisticsTable
            data={stats.items}
            loading={loadingContent}
            columnVisibility={columnVisibility}
          />
        </div>

        <div className='mt-3 flex flex-wrap items-center justify-end gap-2 text-sm'>
          <Button
            variant='outline'
            size='sm'
            disabled={page <= 1}
            onClick={() => setPage((value) => Math.max(1, value - 1))}
          >
            {t('Previous')}
          </Button>
          <span className='text-muted-foreground tabular-nums'>
            {page} / {Math.max(1, Math.ceil(stats.total / DEFAULT_PAGE_SIZE))}
          </span>
          <Button
            variant='outline'
            size='sm'
            disabled={page >= Math.ceil(stats.total / DEFAULT_PAGE_SIZE)}
            onClick={() => setPage((value) => value + 1)}
          >
            {t('Next')}
          </Button>
        </div>
      </div>
      <RatioSettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
    </Main>
  )
}
