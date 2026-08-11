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
import { Filter, RotateCcw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { DateTimePicker } from '@/components/datetime-picker'
import { Dialog } from '@/components/dialog'
import {
  dateToStatisticsTimestamp,
  statisticsTimestampToDate,
} from './statistics-time-range'
import type { StatisticsTimeRangeState } from './use-statistics-time-range'

interface StatisticsTimeFilterDialogProps {
  state: StatisticsTimeRangeState
  title: string
  description: string
}

export function StatisticsTimeFilterDialog(
  props: StatisticsTimeFilterDialogProps
) {
  const { t } = useTranslation()

  const handleApply = () => {
    const error = props.state.applyDraftRange()
    if (error === 'end_before_start') {
      toast.error(t('End time must be later than start time'))
    } else if (error === 'range_too_long') {
      toast.error(t('Time range cannot exceed 183 days'))
    }
  }

  return (
    <Dialog
      open={props.state.filterOpen}
      onOpenChange={props.state.handleFilterOpenChange}
      trigger={
        <Button variant='outline' size='sm' className='shrink-0'>
          <Filter data-icon='inline-start' />
          {t('Filter')}
        </Button>
      }
      title={props.title}
      description={props.description}
      contentClassName='sm:max-w-lg'
      contentHeight='auto'
      footerClassName='grid grid-cols-2 gap-2 sm:flex'
      footer={
        <>
          <Button
            onClick={props.state.resetTimeRange}
            variant='outline'
            type='button'
          >
            <RotateCcw data-icon='inline-start' />
            {t('Reset')}
          </Button>
          <Button onClick={handleApply} type='button'>
            <Filter data-icon='inline-start' />
            {t('Apply Filters')}
          </Button>
        </>
      }
    >
      <div className='flex flex-col gap-3 py-1'>
        <div className='grid gap-2'>
          <div className='text-sm font-medium'>{t('Start Time')}</div>
          <DateTimePicker
            value={statisticsTimestampToDate(
              props.state.draftRange.start_timestamp
            )}
            onChange={(date) => {
              const timestamp = dateToStatisticsTimestamp(date)
              if (timestamp !== undefined) {
                props.state.setDraftRange((previous) => ({
                  ...previous,
                  start_timestamp: timestamp,
                }))
              }
            }}
            placeholder={t('Select start time')}
          />
        </div>

        <div className='grid gap-2'>
          <div className='text-sm font-medium'>{t('End Time')}</div>
          <DateTimePicker
            value={statisticsTimestampToDate(
              props.state.draftRange.end_timestamp
            )}
            onChange={(date) => {
              const timestamp = dateToStatisticsTimestamp(date)
              if (timestamp !== undefined) {
                props.state.setDraftRange((previous) => ({
                  ...previous,
                  end_timestamp: timestamp,
                }))
              }
            }}
            placeholder={t('Select end time')}
          />
        </div>
      </div>
    </Dialog>
  )
}
