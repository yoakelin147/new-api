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
import { ChevronLeft, ChevronRight, Loader2, Plus, Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useIsMobile } from '@/hooks/use-mobile'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { Dialog } from '@/components/dialog'
import { StatusBadge } from '@/components/status-badge'
import { getEnabledModels } from '@/features/channels/api'

type MissingModelPricingDialogProps = {
  open: boolean
  configuredModelNames: Set<string>
  onOpenChange: (open: boolean) => void
  onConfigure: (modelName: string) => void
}

const PAGE_SIZE = 10

export function MissingModelPricingDialog({
  open,
  configuredModelNames,
  onOpenChange,
  onConfigure,
}: MissingModelPricingDialogProps) {
  const { t } = useTranslation()
  const isMobile = useIsMobile()
  const [searchTerm, setSearchTerm] = useState('')
  const [currentPage, setCurrentPage] = useState(1)

  const { data, isLoading } = useQuery({
    queryKey: ['channels', 'models-enabled'],
    queryFn: getEnabledModels,
    enabled: open,
    staleTime: 60_000,
  })

  const enabledModels = useMemo(() => {
    const seen = new Set<string>()

    for (const modelName of data?.data ?? []) {
      const normalized = modelName.trim()
      if (normalized) seen.add(normalized)
    }

    return Array.from(seen).sort((a, b) => a.localeCompare(b))
  }, [data?.data])

  const missingModels = useMemo(
    () =>
      enabledModels.filter((modelName) => !configuredModelNames.has(modelName)),
    [configuredModelNames, enabledModels]
  )

  const filteredModels = useMemo(() => {
    const keyword = searchTerm.toLowerCase().trim()
    if (!keyword) return missingModels
    return missingModels.filter((modelName) =>
      modelName.toLowerCase().includes(keyword)
    )
  }, [missingModels, searchTerm])

  const totalItems = filteredModels.length
  const totalPages =
    totalItems === 0 ? 1 : Math.ceil(totalItems / Math.max(1, PAGE_SIZE))
  const visiblePage = Math.min(currentPage, totalPages)

  const paginatedModels = useMemo(() => {
    const startIndex = (visiblePage - 1) * PAGE_SIZE
    return filteredModels.slice(startIndex, startIndex + PAGE_SIZE)
  }, [filteredModels, visiblePage])

  const displayStart = totalItems === 0 ? 0 : (visiblePage - 1) * PAGE_SIZE + 1
  const displayEnd =
    totalItems === 0 ? 0 : Math.min(visiblePage * PAGE_SIZE, totalItems)
  const showPagination = totalItems > PAGE_SIZE

  const handleConfigure = (modelName: string) => {
    onConfigure(modelName)
    onOpenChange(false)
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Missing Model Pricing')}
      description={t(
        'Models enabled in channels but not configured in model pricing.'
      )}
      contentClassName='flex max-h-[85vh] max-w-2xl flex-col gap-3 p-4'
      headerClassName='flex-shrink-0 text-start'
      contentHeight='min(74vh, 760px)'
      bodyClassName='space-y-4'
      initialFocus={!isMobile}
    >
      {isLoading ? (
        <div className='flex items-center justify-center py-12'>
          <Loader2 className='animate-spin' />
        </div>
      ) : missingModels.length === 0 ? (
        <Empty className='border py-12'>
          <EmptyHeader>
            <EmptyMedia variant='icon'>
              <Search />
            </EmptyMedia>
            <EmptyTitle>{t('No missing model pricing found')}</EmptyTitle>
            <EmptyDescription>
              {t('All enabled models already have pricing configured.')}
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <div className='flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto'>
          <div className='flex flex-shrink-0 items-center justify-between gap-3'>
            <div className='text-muted-foreground text-sm whitespace-nowrap'>
              {t('Showing')} {displayStart}-{displayEnd} {t('of')} {totalItems}
            </div>
            <div className='relative w-48'>
              <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 -translate-y-1/2' />
              <Input
                value={searchTerm}
                onChange={(event) => {
                  setSearchTerm(event.target.value)
                  setCurrentPage(1)
                }}
                placeholder={t('Search models...')}
                className='pl-9'
                aria-label={t('Search missing model pricing')}
              />
            </div>
          </div>

          {filteredModels.length === 0 ? (
            <Empty className='border'>
              <EmptyHeader>
                <EmptyMedia variant='icon'>
                  <Search />
                </EmptyMedia>
                <EmptyTitle>{t('No matches found')}</EmptyTitle>
                <EmptyDescription>
                  {t(
                    'Try adjusting your search to locate a missing model price.'
                  )}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <div className='flex-shrink-0 rounded-lg border'>
              <div className='divide-y'>
                {paginatedModels.map((modelName) => (
                  <div
                    key={modelName}
                    className='flex items-center justify-between gap-3 p-3'
                  >
                    <div className='min-w-0 flex-1'>
                      <StatusBadge
                        label={modelName}
                        variant='neutral'
                        copyText={modelName}
                      />
                    </div>
                    <Button
                      size='sm'
                      className='flex-shrink-0'
                      onClick={() => handleConfigure(modelName)}
                    >
                      <Plus data-icon='inline-start' />
                      {t('Configure')}
                    </Button>
                  </div>
                ))}
              </div>

              <div className='bg-muted/40 flex items-center justify-between border-t px-3 py-2 text-sm'>
                <div className='text-muted-foreground text-sm'>
                  {t('Page {{current}} of {{total}}', {
                    current: visiblePage,
                    total: totalPages,
                  })}
                </div>
                {showPagination && (
                  <div className='flex items-center gap-2'>
                    <Button
                      variant='outline'
                      size='icon'
                      className='size-8'
                      onClick={() =>
                        setCurrentPage((previous) => Math.max(1, previous - 1))
                      }
                      disabled={visiblePage === 1}
                      aria-label={t('Previous page')}
                    >
                      <ChevronLeft />
                    </Button>
                    <Button
                      variant='outline'
                      size='icon'
                      className='size-8'
                      onClick={() =>
                        setCurrentPage((previous) =>
                          Math.min(totalPages, previous + 1)
                        )
                      }
                      disabled={visiblePage === totalPages}
                      aria-label={t('Next page')}
                    >
                      <ChevronRight />
                    </Button>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      )}
    </Dialog>
  )
}
