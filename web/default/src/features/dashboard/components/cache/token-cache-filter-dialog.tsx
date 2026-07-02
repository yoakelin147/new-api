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
import { useCallback, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Calendar, Filter, RotateCcw, Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { getRollingDateRange, type TimeGranularity } from '@/lib/time'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { DateTimePicker } from '@/components/datetime-picker'
import { Dialog } from '@/components/dialog'
import { getChannels } from '@/features/channels/api'
import {
  TIME_GRANULARITY_OPTIONS,
  TIME_RANGE_PRESETS,
} from '@/features/dashboard/constants'
import {
  buildDefaultTokenCacheFilters,
  type TokenCacheDashboardFilters,
} from './token-cache-filters'

const ALL_CHANNELS_VALUE = '__all__'

const SectionDivider = ({ label }: { label: string }) => (
  <div className='relative'>
    <div className='absolute inset-0 flex items-center'>
      <span className='w-full border-t' />
    </div>
    <div className='relative flex justify-center text-xs uppercase'>
      <span className='bg-background text-muted-foreground px-2'>{label}</span>
    </div>
  </div>
)

interface TokenCacheFilterDialogProps {
  filters: TokenCacheDashboardFilters
  onFilterChange: (filters: TokenCacheDashboardFilters) => void
}

export function TokenCacheFilterDialog({
  filters,
  onFilterChange,
}: TokenCacheFilterDialogProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState<TokenCacheDashboardFilters>(filters)

  const { data: channelsResponse } = useQuery({
    queryKey: ['dashboard', 'token-cache-filter', 'channels'],
    queryFn: () => getChannels({ p: 0, page_size: 1000, id_sort: true }),
    staleTime: 300_000,
    enabled: open,
  })

  const channels = useMemo(() => {
    const items = channelsResponse?.success
      ? (channelsResponse.data?.items ?? [])
      : []
    return [...items].sort((a, b) => a.id - b.id)
  }, [channelsResponse])

  const channelValue =
    draft.channel_id && draft.channel_id > 0
      ? String(draft.channel_id)
      : ALL_CHANNELS_VALUE

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (nextOpen) setDraft(filters)
      setOpen(nextOpen)
    },
    [filters]
  )

  const handleQuickRange = useCallback((days: number) => {
    const { start, end } = getRollingDateRange(days)
    setDraft((current) => ({
      ...current,
      start_timestamp: start,
      end_timestamp: end,
      range_days: days,
    }))
  }, [])

  const handleReset = useCallback(() => {
    const next = buildDefaultTokenCacheFilters()
    setDraft(next)
    onFilterChange(next)
    setOpen(false)
  }, [onFilterChange])

  const handleApply = useCallback(() => {
    const start = draft.start_timestamp
    const end = draft.end_timestamp
    const normalized =
      start.getTime() <= end.getTime()
        ? draft
        : {
            ...draft,
            start_timestamp: end,
            end_timestamp: start,
          }
    onFilterChange({
      ...normalized,
      model_name: normalized.model_name?.trim() || '',
    })
    setOpen(false)
  }, [draft, onFilterChange])

  return (
    <Dialog
      open={open}
      onOpenChange={handleOpenChange}
      trigger={
        <Button variant='outline' size='sm'>
          <Filter className='mr-2 h-4 w-4' />
          {t('Filter')}
        </Button>
      }
      title={t('Token Cache Analytics Filters')}
      description={t('Filter token cache analytics by time, model, and channel.')}
      contentClassName='max-sm:h-dvh max-sm:w-screen max-sm:max-w-none max-sm:rounded-none max-sm:p-4 sm:max-w-lg'
      contentHeight='min(56vh, 520px)'
      footerClassName='grid grid-cols-2 gap-2 sm:flex'
      footer={
        <>
          <Button onClick={handleReset} variant='outline' type='button'>
            <RotateCcw className='mr-2 h-4 w-4' />
            {t('Reset')}
          </Button>
          <Button onClick={handleApply} type='submit'>
            <Search className='mr-2 h-4 w-4' />
            {t('Apply Filters')}
          </Button>
        </>
      }
    >
      <ScrollArea className='h-full pr-3 sm:pr-4'>
        <div className='grid gap-2.5 py-2'>
          <div className='grid gap-2'>
            <Label className='flex items-center gap-2'>
              <Calendar className='h-4 w-4' />
              {t('Quick Range')}
            </Label>
            <div className='grid grid-cols-2 gap-2 sm:flex'>
              {TIME_RANGE_PRESETS.map((range) => (
                <Button
                  key={range.days}
                  type='button'
                  size='sm'
                  variant={draft.range_days === range.days ? 'default' : 'outline'}
                  onClick={() => handleQuickRange(range.days)}
                  className={cn(
                    'flex-1',
                    draft.range_days === range.days &&
                      'ring-ring ring-2 ring-offset-2'
                  )}
                >
                  {t(range.label)}
                </Button>
              ))}
            </div>
          </div>

          <SectionDivider label={t('Custom Time Range')} />

          <div className='grid gap-2.5'>
            <div className='grid gap-2'>
              <Label>{t('Start Time')}</Label>
              <DateTimePicker
                value={draft.start_timestamp}
                onChange={(date) => {
                  if (!date) return
                  setDraft((current) => ({
                    ...current,
                    start_timestamp: date,
                    range_days: null,
                  }))
                }}
                placeholder={t('Select start time')}
              />
            </div>

            <div className='grid gap-2'>
              <Label>{t('End Time')}</Label>
              <DateTimePicker
                value={draft.end_timestamp}
                onChange={(date) => {
                  if (!date) return
                  setDraft((current) => ({
                    ...current,
                    end_timestamp: date,
                    range_days: null,
                  }))
                }}
                placeholder={t('Select end time')}
              />
            </div>
          </div>

          <SectionDivider label={t('Chart Settings')} />

          <div className='grid gap-2'>
            <Label>{t('Time Granularity')}</Label>
            <Select
              value={draft.time_granularity}
              onValueChange={(value) =>
                setDraft((current) => ({
                  ...current,
                  time_granularity: (value ?? 'hour') as TimeGranularity,
                }))
              }
            >
              <SelectTrigger>
                <SelectValue placeholder={t('Select time granularity')} />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  {TIME_GRANULARITY_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {t(option.label)}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>

          <SectionDivider label={t('Filters')} />

          <div className='grid gap-2'>
            <Label>{t('Model')}</Label>
            <Input
              value={draft.model_name ?? ''}
              onChange={(event) =>
                setDraft((current) => ({
                  ...current,
                  model_name: event.target.value,
                }))
              }
              placeholder={t('Filter model')}
            />
          </div>

          <div className='grid gap-2'>
            <Label>{t('Channel')}</Label>
            <Select
              value={channelValue}
              onValueChange={(value) =>
                setDraft((current) => ({
                  ...current,
                  channel_id:
                    value && value !== ALL_CHANNELS_VALUE
                      ? Number(value)
                      : undefined,
                }))
              }
            >
              <SelectTrigger>
                <SelectValue placeholder={t('All Channels')} />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  <SelectItem value={ALL_CHANNELS_VALUE}>
                    {t('All Channels')}
                  </SelectItem>
                  {channels.map((channel) => (
                    <SelectItem key={channel.id} value={String(channel.id)}>
                      #{channel.id} {channel.name}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </div>
        </div>
      </ScrollArea>
    </Dialog>
  )
}
