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
import { useEffect, useState } from 'react'
import { Clock01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface DataFreshnessIndicatorProps {
  updatedAt?: number
  className?: string
}

function getRelativeTime(
  updatedAt: number,
  now: number,
  language: string,
  justNow: string
) {
  const elapsedSeconds = Math.max(0, Math.floor((now - updatedAt) / 1000))
  if (elapsedSeconds < 60) return justNow

  const formatter = new Intl.RelativeTimeFormat(language, {
    numeric: 'always',
  })

  const minutes = Math.floor(elapsedSeconds / 60)
  if (minutes < 60) return formatter.format(-minutes, 'minute')

  const hours = Math.floor(minutes / 60)
  if (hours < 24) return formatter.format(-hours, 'hour')

  const days = Math.floor(hours / 24)
  if (days < 30) return formatter.format(-days, 'day')

  const months = Math.floor(days / 30)
  if (months < 12) return formatter.format(-months, 'month')

  const years = Math.floor(months / 12)
  return formatter.format(-years, 'year')
}

export function DataFreshnessIndicator({
  updatedAt,
  className,
}: DataFreshnessIndicatorProps) {
  const { t, i18n } = useTranslation()
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    if (!updatedAt) return

    const timer = window.setInterval(() => setNow(Date.now()), 30_000)
    return () => window.clearInterval(timer)
  }, [updatedAt])

  if (!updatedAt) return null

  const relativeTime = getRelativeTime(
    updatedAt,
    now,
    i18n.language,
    t('Just now')
  )
  const exactTime = new Intl.DateTimeFormat(i18n.language, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(updatedAt))

  return (
    <Tooltip>
      <TooltipTrigger
        className={cn(
          'text-muted-foreground inline-flex min-w-0 items-center gap-1 text-xs',
          className
        )}
      >
        <HugeiconsIcon icon={Clock01Icon} className='size-3.5 shrink-0' />
        <span className='truncate'>
          {t('Last updated: {{time}}', { time: relativeTime })}
        </span>
      </TooltipTrigger>
      <TooltipContent>{exactTime}</TooltipContent>
    </Tooltip>
  )
}
