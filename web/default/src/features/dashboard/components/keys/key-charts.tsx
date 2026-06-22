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
import {
  useDeferredValue,
  useEffect,
  useMemo,
  useState,
  useRef,
  useCallback,
} from 'react'
import { useQuery } from '@tanstack/react-query'
import { VChart } from '@visactor/react-vchart'
import {
  Coins,
  Check,
  ChevronsUpDown,
  KeyRound,
  Loader2,
  MousePointerClick,
  Users,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/stores/auth-store'
import { formatNumber, formatQuota } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { getRollingDateRange, type TimeGranularity } from '@/lib/time'
import { cn } from '@/lib/utils'
import { VCHART_OPTION } from '@/lib/vchart'
import { useThemeCustomization } from '@/context/theme-customization-provider'
import { useTheme } from '@/context/theme-provider'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  getUserQuotaDataByTokens,
  getUserQuotaDataByUserTokens,
} from '@/features/dashboard/api'
import {
  TIME_GRANULARITY_OPTIONS,
  TIME_RANGE_PRESETS,
} from '@/features/dashboard/constants'
import {
  getDefaultDays,
  getSavedGranularity,
  saveGranularity,
  processTokenChartData,
} from '@/features/dashboard/lib'
import type { ProcessedTokenChartData } from '@/features/dashboard/types'
import { getUsers, searchUsers } from '@/features/users/api'
import type { User } from '@/features/users/types'

let themeManagerPromise: Promise<
  (typeof import('@visactor/vchart'))['ThemeManager']
> | null = null

const TOKEN_CHARTS: {
  value: string
  labelKey: string
  specKey: keyof ProcessedTokenChartData
  icon: typeof KeyRound
}[] = [
  {
    value: 'request-rank',
    labelKey: 'API Key Request Ranking',
    specKey: 'spec_token_request_rank',
    icon: MousePointerClick,
  },
  {
    value: 'usage-rank',
    labelKey: 'API Key Token Usage Ranking',
    specKey: 'spec_token_usage_rank',
    icon: KeyRound,
  },
  {
    value: 'consumption-rank',
    labelKey: 'API Key Consumption Ranking',
    specKey: 'spec_token_rank',
    icon: Coins,
  },
  {
    value: 'consumption-trend',
    labelKey: 'API Key Consumption Trend',
    specKey: 'spec_token_trend',
    icon: KeyRound,
  },
]

const TOP_TOKEN_LIMIT_OPTIONS = [5, 10, 20, 50]

export function KeyCharts() {
  const { t } = useTranslation()
  const { resolvedTheme } = useTheme()
  const { customization } = useThemeCustomization()
  const currentUser = useAuthStore((state) => state.auth.user)
  const isAdmin = Boolean(currentUser?.role && currentUser.role >= ROLE.ADMIN)
  const [themeReady, setThemeReady] = useState(false)
  const themeManagerRef = useRef<
    (typeof import('@visactor/vchart'))['ThemeManager'] | null
  >(null)
  const [userSelectorOpen, setUserSelectorOpen] = useState(false)
  const [selectedUserId, setSelectedUserId] = useState('')
  const [userKeyword, setUserKeyword] = useState('')
  const deferredUserKeyword = useDeferredValue(userKeyword)

  const [timeGranularity, setTimeGranularity] = useState<TimeGranularity>(() =>
    getSavedGranularity()
  )
  const [selectedRange, setSelectedRange] = useState<number>(() =>
    getDefaultDays(timeGranularity)
  )
  const [topTokenLimit, setTopTokenLimit] = useState(10)
  const [timeRange, setTimeRange] = useState(() => {
    const days = getDefaultDays(timeGranularity)
    const { start, end } = getRollingDateRange(days)
    return {
      start_timestamp: Math.floor(start.getTime() / 1000),
      end_timestamp: Math.floor(end.getTime() / 1000),
    }
  })

  const handleRangeChange = useCallback((days: number) => {
    setSelectedRange(days)
    const { start, end } = getRollingDateRange(days)
    setTimeRange({
      start_timestamp: Math.floor(start.getTime() / 1000),
      end_timestamp: Math.floor(end.getTime() / 1000),
    })
  }, [])

  const handleGranularityChange = useCallback(
    (g: TimeGranularity) => {
      setTimeGranularity(g)
      saveGranularity(g)
      const days = getDefaultDays(g)
      if (days !== selectedRange) {
        handleRangeChange(days)
      }
    },
    [selectedRange, handleRangeChange]
  )

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

  const activeSelectedUserId =
    selectedUserId || (currentUser?.id ? String(currentUser.id) : '')
  const effectiveUserId = Number(activeSelectedUserId || 0)
  const trimmedUserKeyword = deferredUserKeyword.trim()

  const { data: userOptionsData, isLoading: isUsersLoading } = useQuery({
    queryKey: ['dashboard', 'token-user-options', trimmedUserKeyword],
    queryFn: () =>
      trimmedUserKeyword
        ? searchUsers({
            keyword: trimmedUserKeyword,
            p: 1,
            page_size: 100,
          })
        : getUsers({ p: 1, page_size: 100 }),
    select: (res) => (res.success ? (res.data?.items ?? []) : []),
    staleTime: 300_000,
    enabled: isAdmin,
  })

  const userOptions = useMemo(() => {
    const users = new Map<number, Pick<User, 'id' | 'username'> & { display_name?: string }>()
    if (currentUser?.id) {
      users.set(currentUser.id, {
        id: currentUser.id,
        username: currentUser.username,
        display_name: currentUser.display_name,
      })
    }
    for (const user of userOptionsData ?? []) {
      users.set(user.id, {
        id: user.id,
        username: user.username,
        display_name: user.display_name,
      })
    }
    return Array.from(users.values()).sort((a, b) => a.id - b.id)
  }, [currentUser, userOptionsData])

  const selectedUser = useMemo(() => {
    if (!effectiveUserId) return undefined
    return userOptions.find((user) => user.id === effectiveUserId)
  }, [effectiveUserId, userOptions])

  const selectedUserLabel =
    selectedUser?.display_name || selectedUser?.username || `#${effectiveUserId}`

  const handleUserSelect = useCallback((value: string) => {
    setSelectedUserId(value)
    setUserSelectorOpen(false)
    setUserKeyword('')
  }, [])

  const { data: tokenData, isLoading } = useQuery({
    queryKey: [
      'dashboard',
      'token-quota',
      timeRange,
      isAdmin ? effectiveUserId : 'self',
    ],
    queryFn: () =>
      isAdmin
        ? getUserQuotaDataByUserTokens({
            ...timeRange,
            user_id: effectiveUserId,
          })
        : getUserQuotaDataByTokens(timeRange),
    select: (res) => (res.success ? res.data : []),
    staleTime: 60_000,
    enabled: !isAdmin || effectiveUserId > 0,
  })

  const chartData = useMemo(
    () =>
      processTokenChartData(
        isLoading ? [] : (tokenData ?? []),
        timeGranularity,
        t,
        topTokenLimit,
        customization.preset
      ),
    [
      tokenData,
      isLoading,
      timeGranularity,
      t,
      topTokenLimit,
      customization.preset,
    ]
  )

  const summaryStats = useMemo(
    () =>
      (tokenData ?? []).reduce(
        (acc, item) => {
          acc.count += Number(item.count) || 0
          acc.tokenUsed += Number(item.token_used) || 0
          acc.quota += Number(item.quota) || 0
          return acc
        },
        { count: 0, tokenUsed: 0, quota: 0 }
      ),
    [tokenData]
  )

  const summaryCards = [
    {
      key: 'requests',
      title: t('Requests'),
      value: formatNumber(summaryStats.count),
      icon: MousePointerClick,
    },
    {
      key: 'tokens',
      title: t('Token Usage'),
      value: formatNumber(summaryStats.tokenUsed),
      icon: KeyRound,
    },
    {
      key: 'consumption',
      title: t('Consumption'),
      value: formatQuota(summaryStats.quota),
      icon: Coins,
    },
  ]

  return (
    <div className='space-y-3'>
      <div className='flex items-center gap-1.5 overflow-x-auto pb-1 sm:gap-2'>
        <Tabs
          value={String(selectedRange)}
          onValueChange={(value) => handleRangeChange(Number(value))}
          className='shrink-0'
        >
          <TabsList>
            {TIME_RANGE_PRESETS.map((preset) => (
              <TabsTrigger
                key={preset.days}
                value={String(preset.days)}
                className='px-2.5 text-xs'
              >
                {t(preset.label)}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

        <Tabs
          value={timeGranularity}
          onValueChange={(value) =>
            handleGranularityChange(value as TimeGranularity)
          }
          className='shrink-0'
        >
          <TabsList>
            {TIME_GRANULARITY_OPTIONS.map((opt) => (
              <TabsTrigger
                key={opt.value}
                value={opt.value}
                className='px-2.5 text-xs'
              >
                {t(opt.label)}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

        <Tabs
          value={String(topTokenLimit)}
          onValueChange={(value) => setTopTokenLimit(Number(value))}
          className='shrink-0'
        >
          <TabsList>
            <span className='text-muted-foreground px-2 text-xs font-medium whitespace-nowrap'>
              {t('Top API Keys')}
            </span>
            {TOP_TOKEN_LIMIT_OPTIONS.map((limit) => (
              <TabsTrigger
                key={limit}
                value={String(limit)}
                className='px-2.5 text-xs'
              >
                {t('Top {{count}}', { count: limit })}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

        {isAdmin && (
          <Popover open={userSelectorOpen} onOpenChange={setUserSelectorOpen}>
            <PopoverTrigger
              render={
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  role='combobox'
                  aria-expanded={userSelectorOpen}
                  className='min-w-48 max-w-64 justify-between gap-2'
                />
              }
            >
              <Users data-icon='inline-start' />
              <span className='min-w-0 flex-1 truncate text-left'>
                {selectedUserLabel || t('Select customer')}
              </span>
              <ChevronsUpDown data-icon='inline-end' />
            </PopoverTrigger>
            <PopoverContent
              className='data-closed:zoom-out-100 data-open:zoom-in-100 data-[side=bottom]:slide-in-from-top-0 data-[side=left]:slide-in-from-right-0 data-[side=right]:slide-in-from-left-0 data-[side=top]:slide-in-from-bottom-0 w-[var(--anchor-width)] overflow-hidden rounded-xl p-0 shadow-lg data-closed:duration-75 data-open:duration-100'
              align='start'
              onWheel={(event) => event.stopPropagation()}
              onTouchMove={(event) => event.stopPropagation()}
              onPointerDown={(event) => event.stopPropagation()}
            >
              <Command shouldFilter={false}>
                <CommandInput
                  placeholder={t('Search customer')}
                  value={userKeyword}
                  onValueChange={setUserKeyword}
                />
                <CommandList className='max-h-[320px]'>
                  <CommandEmpty>
                    {isUsersLoading ? t('Loading...') : t('No results found.')}
                  </CommandEmpty>
                  <CommandGroup>
                    {userOptions.map((user) => {
                      const value = String(user.id)
                      const label =
                        user.display_name ||
                        user.username ||
                        `#${String(user.id)}`
                      const isSelected = value === activeSelectedUserId
                      return (
                        <CommandItem
                          key={user.id}
                          value={`${label} ${user.username} ${user.id}`}
                          onSelect={() => handleUserSelect(value)}
                          className='data-[selected=true]:bg-muted rounded-lg px-3 py-2'
                        >
                          <Check
                            className={cn(
                              'size-4',
                              isSelected ? 'opacity-100' : 'opacity-0'
                            )}
                          />
                          <span className='min-w-0 flex-1'>
                            <span className='block truncate font-medium'>
                              {label}
                              {user.id === currentUser?.id
                                ? ` (${t('Self')})`
                                : ''}
                            </span>
                            <span className='text-muted-foreground block truncate text-xs'>
                              #{user.id}
                              {user.username ? ` · ${user.username}` : ''}
                            </span>
                          </span>
                        </CommandItem>
                      )
                    })}
                  </CommandGroup>
                </CommandList>
              </Command>
            </PopoverContent>
          </Popover>
        )}

        {(isLoading || isUsersLoading) && (
          <Loader2 className='text-muted-foreground size-4 animate-spin' />
        )}
      </div>

      {isAdmin && effectiveUserId > 0 && (
        <div className='text-muted-foreground text-xs'>
          {t('Viewing customer')}: {selectedUserLabel}
        </div>
      )}

      <div className='overflow-hidden rounded-lg border'>
        <div className='divide-border/60 grid grid-cols-1 divide-y sm:grid-cols-3 sm:divide-x sm:divide-y-0'>
          {summaryCards.map((card) => {
            const Icon = card.icon
            return (
              <div key={card.key} className='px-3 py-2.5 sm:px-5 sm:py-4'>
                <div className='flex items-center gap-2'>
                  <Icon className='text-muted-foreground/60 size-3.5 shrink-0' />
                  <div className='text-muted-foreground truncate text-xs font-medium tracking-wider uppercase'>
                    {card.title}
                  </div>
                </div>
                {isLoading ? (
                  <div className='mt-2 space-y-1.5'>
                    <Skeleton className='h-7 w-24' />
                    <Skeleton className='h-3.5 w-28' />
                  </div>
                ) : (
                  <div className='text-foreground mt-1.5 font-mono text-lg font-bold tracking-tight tabular-nums sm:mt-2 sm:text-2xl'>
                    {card.value}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>

      <div className='grid gap-3'>
        {TOKEN_CHARTS.map((chart) => {
          const spec = chartData[chart.specKey]
          const Icon = chart.icon

          return (
            <div
              key={chart.value}
              className='overflow-hidden rounded-lg border'
            >
              <div className='flex w-full items-center gap-2 border-b px-3 py-2 sm:px-5 sm:py-3'>
                <Icon className='text-muted-foreground/60 size-4' />
                <div className='text-sm font-semibold'>{t(chart.labelKey)}</div>
              </div>

              <div className='h-[300px] p-1.5 sm:h-96 sm:p-2'>
                {isLoading ? (
                  <Skeleton className='h-full w-full' />
                ) : (
                  themeReady &&
                  spec && (
                    <VChart
                      key={`token-${chart.value}-${topTokenLimit}-${resolvedTheme}-${customization.preset}-${effectiveUserId}`}
                      spec={{
                        ...spec,
                        theme: resolvedTheme === 'dark' ? 'dark' : 'light',
                        background: 'transparent',
                      }}
                      option={VCHART_OPTION}
                    />
                  )
                )}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
