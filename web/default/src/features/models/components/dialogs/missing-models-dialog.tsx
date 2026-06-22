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
import { useEffect, useMemo, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ChevronLeft,
  ChevronRight,
  Loader2,
  Plus,
  Search,
  Sparkles,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useIsMobile } from '@/hooks/use-mobile'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Dialog } from '@/components/dialog'
import { StatusBadge } from '@/components/status-badge'
import { getApiKeys } from '@/features/keys/api'
import { getGroups } from '@/features/users/api'
import { aiConfigureMissingModels, getMissingModels } from '../../api'
import { DEFAULT_PAGE_SIZE } from '../../constants'
import { modelsQueryKeys, vendorsQueryKeys } from '../../lib'
import type { Model } from '../../types'
import { useModels } from '../models-provider'

const MAX_AI_CONFIGURE_MODELS = 50

type MissingModelsDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function MissingModelsDialog({
  open,
  onOpenChange,
}: MissingModelsDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { setOpen, setCurrentRow } = useModels()
  const isMobile = useIsMobile()
  const [searchTerm, setSearchTerm] = useState('')
  const [currentPage, setCurrentPage] = useState(1)
  const [selectedModels, setSelectedModels] = useState<string[]>([])
  const [selectedTokenId, setSelectedTokenId] = useState('')
  const [selectedGroup, setSelectedGroup] = useState('default')
  const [aiModel, setAIModel] = useState('gpt-4o-mini')
  const [descriptionLanguage, setDescriptionLanguage] = useState<
    'zh' | 'en' | 'ja'
  >('zh')
  const [isAIConfiguring, setIsAIConfiguring] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: modelsQueryKeys.missing(),
    queryFn: getMissingModels,
    enabled: open,
  })

  const { data: tokenData } = useQuery({
    queryKey: ['model-ai-config-tokens'],
    queryFn: () => getApiKeys({ p: 1, size: 100 }),
    enabled: open,
  })

  const { data: groupsData } = useQuery({
    queryKey: ['model-ai-config-groups'],
    queryFn: getGroups,
    enabled: open,
  })

  const missingModels = useMemo(() => data?.data || [], [data?.data])
  const tokenOptions = useMemo(
    () => tokenData?.data?.items?.filter((token) => token.status === 1) || [],
    [tokenData?.data?.items]
  )
  const groupOptions = useMemo(() => groupsData?.data || [], [groupsData?.data])
  const pageSize = DEFAULT_PAGE_SIZE

  const handleConfigureModel = (modelName: string) => {
    setCurrentRow({ model_name: modelName } as unknown as Model)
    setOpen('create-model')
  }

  useEffect(() => {
    if (open) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setSearchTerm('')

      setCurrentPage(1)
      setSelectedModels([])
    }
  }, [open])

  useEffect(() => {
    if (!selectedTokenId && tokenOptions.length > 0) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setSelectedTokenId(String(tokenOptions[0].id))
    }
  }, [selectedTokenId, tokenOptions])

  useEffect(() => {
    if (groupOptions.length === 0) return
    if (!groupOptions.includes(selectedGroup)) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setSelectedGroup(
        groupOptions.includes('default') ? 'default' : groupOptions[0]
      )
    }
  }, [groupOptions, selectedGroup])

  const filteredModels = useMemo(() => {
    if (!searchTerm.trim()) {
      return missingModels
    }
    const keyword = searchTerm.toLowerCase().trim()
    return missingModels.filter((modelName) =>
      modelName.toLowerCase().includes(keyword)
    )
  }, [missingModels, searchTerm])

  const totalItems = filteredModels.length
  const totalPages =
    totalItems === 0 ? 1 : Math.ceil(totalItems / Math.max(1, pageSize))

  useEffect(() => {
    if (currentPage > totalPages) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setCurrentPage(Math.max(1, totalPages))
    }
  }, [currentPage, totalPages])

  const paginatedModels = useMemo(() => {
    const startIndex = (currentPage - 1) * pageSize
    const endIndex = startIndex + pageSize
    return filteredModels.slice(startIndex, endIndex)
  }, [filteredModels, currentPage, pageSize])

  const displayStart = totalItems === 0 ? 0 : (currentPage - 1) * pageSize + 1
  const displayEnd =
    totalItems === 0 ? 0 : Math.min(currentPage * pageSize, totalItems)
  const showPagination = totalItems > pageSize
  const selectedSet = useMemo(() => new Set(selectedModels), [selectedModels])
  const selectedCount = selectedModels.length
  const selectedCountExceedsLimit = selectedCount > MAX_AI_CONFIGURE_MODELS
  const pageSelectedCount = paginatedModels.filter((modelName) =>
    selectedSet.has(modelName)
  ).length
  const allCurrentPageSelected =
    paginatedModels.length > 0 && pageSelectedCount === paginatedModels.length
  const allFilteredSelected =
    filteredModels.length > 0 &&
    filteredModels.every((modelName) => selectedSet.has(modelName))
  const hasSelectedModels = selectedModels.length > 0

  const toggleModelSelection = (modelName: string, checked: boolean) => {
    setSelectedModels((prev) => {
      if (checked) {
        return prev.includes(modelName) ? prev : [...prev, modelName]
      }
      return prev.filter((item) => item !== modelName)
    })
  }

  const toggleCurrentPage = (checked: boolean) => {
    setSelectedModels((prev) => {
      const next = new Set(prev)
      for (const modelName of paginatedModels) {
        if (checked) {
          next.add(modelName)
        } else {
          next.delete(modelName)
        }
      }
      return [...next]
    })
  }

  const toggleAllFiltered = (checked: boolean) => {
    if (checked) {
      setSelectedModels(filteredModels)
      return
    }
    setSelectedModels([])
  }

  const handleAIConfigure = async () => {
    if (!hasSelectedModels) {
      toast.warning(t('Select at least one missing model.'))
      return
    }
    if (!selectedTokenId) {
      toast.warning(t('Select an API key for AI configuration.'))
      return
    }
    if (!aiModel.trim()) {
      toast.warning(t('Enter an AI model for configuration.'))
      return
    }
    if (selectedCountExceedsLimit) {
      toast.warning(
        t('AI configuration supports at most {{count}} models per request.', {
          count: MAX_AI_CONFIGURE_MODELS,
        })
      )
      return
    }

    setIsAIConfiguring(true)
    try {
      const response = await aiConfigureMissingModels({
        model_names: selectedModels,
        token_id: Number(selectedTokenId),
        group: selectedGroup,
        ai_model: aiModel.trim(),
        language: descriptionLanguage,
        apply: true,
      })
      if (!response.success) {
        toast.error(response.message || t('AI configuration failed.'))
        return
      }
      const result = response.data?.apply_result
      toast.success(
        t(
          'AI configuration applied. Created {{models}} models and {{vendors}} vendors.',
          {
            models: result?.created_models || 0,
            vendors: result?.created_vendors || 0,
          }
        )
      )
      setSelectedModels([])
      queryClient.invalidateQueries({ queryKey: modelsQueryKeys.lists() })
      queryClient.invalidateQueries({ queryKey: modelsQueryKeys.missing() })
      queryClient.invalidateQueries({ queryKey: vendorsQueryKeys.lists() })
    } catch (error: unknown) {
      toast.error((error as Error)?.message || t('AI configuration failed.'))
    } finally {
      setIsAIConfiguring(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Missing Models')}
      description={t(
        'Models that are being used but not configured in the system'
      )}
      contentClassName='flex max-h-[85vh] max-w-4xl flex-col gap-3 p-4'
      headerClassName='flex-shrink-0 text-start'
      contentHeight='min(74vh, 760px)'
      bodyClassName='space-y-4'
      initialFocus={!isMobile}
    >
      {isLoading ? (
        <div className='flex items-center justify-center py-12'>
          <Loader2 className='h-8 w-8 animate-spin' />
        </div>
      ) : missingModels.length === 0 ? (
        <div className='text-muted-foreground py-12 text-center'>
          <p>{t('No missing models found.')}</p>
          <p className='text-sm'>
            {t('All models in use are properly configured.')}
          </p>
        </div>
      ) : (
        <div className='flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto'>
          <div className='flex flex-col gap-3 rounded-lg border p-3'>
            <div className='flex flex-col gap-1'>
              <Label className='text-sm font-medium'>
                {t('AI Configure Selected Models')}
              </Label>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Use a selected platform API key to generate metadata for missing models and apply it directly.'
                )}
              </p>
            </div>
            <div className='grid gap-2 md:grid-cols-[1.2fr_1fr_0.8fr_1fr_auto]'>
              <Select
                items={tokenOptions.map((token) => ({
                  value: String(token.id),
                  label: token.name,
                }))}
                value={selectedTokenId}
                onValueChange={(value) => value && setSelectedTokenId(value)}
              >
                <SelectTrigger className='w-full'>
                  <SelectValue placeholder={t('Select API key')} />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {tokenOptions.map((token) => (
                      <SelectItem key={token.id} value={String(token.id)}>
                        {token.name}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              <Select
                items={groupOptions.map((group) => ({
                  value: group,
                  label: group,
                }))}
                value={selectedGroup}
                onValueChange={(value) => value && setSelectedGroup(value)}
              >
                <SelectTrigger className='w-full'>
                  <SelectValue placeholder={t('Select group')} />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {groupOptions.map((group) => (
                      <SelectItem key={group} value={group}>
                        {group}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              <Select
                items={[
                  { value: 'zh', label: t('Chinese') },
                  { value: 'en', label: t('English') },
                  { value: 'ja', label: t('Japanese') },
                ]}
                value={descriptionLanguage}
                onValueChange={(value) => {
                  if (value === 'zh' || value === 'en' || value === 'ja') {
                    setDescriptionLanguage(value)
                  }
                }}
              >
                <SelectTrigger className='w-full'>
                  <SelectValue placeholder={t('Description language')} />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    <SelectItem value='zh'>{t('Chinese')}</SelectItem>
                    <SelectItem value='en'>{t('English')}</SelectItem>
                    <SelectItem value='ja'>{t('Japanese')}</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
              <Input
                value={aiModel}
                onChange={(event) => setAIModel(event.target.value)}
                placeholder={t('AI model')}
              />
              <Button
                onClick={handleAIConfigure}
                disabled={
                  isAIConfiguring ||
                  !hasSelectedModels ||
                  selectedCountExceedsLimit
                }
              >
                {isAIConfiguring ? (
                  <Loader2 className='h-4 w-4 animate-spin' />
                ) : (
                  <Sparkles className='h-4 w-4' />
                )}
                {t('AI Configure')}
              </Button>
            </div>
            <div className='text-muted-foreground text-xs'>
              {t('{{count}} models selected', {
                count: selectedCount,
              })}
              {selectedCountExceedsLimit
                ? ` ${t(
                    'AI configuration supports at most {{count}} models per request.',
                    { count: MAX_AI_CONFIGURE_MODELS }
                  )}`
                : ''}
            </div>
          </div>

          <div className='flex flex-shrink-0 flex-col gap-2 md:flex-row md:items-center md:justify-between'>
            <div className='flex flex-wrap items-center gap-2'>
              <Checkbox
                checked={allCurrentPageSelected}
                indeterminate={!allCurrentPageSelected && pageSelectedCount > 0}
                onCheckedChange={(value) => toggleCurrentPage(!!value)}
                aria-label={t('Select current page models')}
              />
              <div className='text-muted-foreground text-sm whitespace-nowrap'>
                {t('Showing')} {displayStart}-{displayEnd} {t('of')}{' '}
                {totalItems}
              </div>
              <Button
                type='button'
                variant='outline'
                size='sm'
                className='h-8'
                onClick={() => toggleCurrentPage(!allCurrentPageSelected)}
                disabled={paginatedModels.length === 0}
              >
                {allCurrentPageSelected
                  ? t('Clear current page')
                  : t('Select current page')}
              </Button>
              <Button
                type='button'
                variant='outline'
                size='sm'
                className='h-8'
                onClick={() => toggleAllFiltered(!allFilteredSelected)}
                disabled={filteredModels.length === 0}
              >
                {allFilteredSelected
                  ? t('Clear all filtered')
                  : t('Select all filtered')}
              </Button>
            </div>
            <div className='relative w-48'>
              <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2' />
              <Input
                value={searchTerm}
                onChange={(event) => {
                  setSearchTerm(event.target.value)
                  setCurrentPage(1)
                }}
                placeholder={t('Search models...')}
                className='pl-9'
                aria-label={t('Search missing models')}
              />
            </div>
          </div>

          {filteredModels.length === 0 ? (
            <Empty className='border'>
              <EmptyHeader>
                <EmptyMedia variant='icon'>
                  <Search className='h-5 w-5' />
                </EmptyMedia>
                <EmptyTitle>{t('No matches found')}</EmptyTitle>
                <EmptyDescription>
                  {t('Try adjusting your search to locate a missing model.')}
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
                    <div className='flex min-w-0 flex-1 items-center gap-3'>
                      <Checkbox
                        checked={selectedSet.has(modelName)}
                        onCheckedChange={(value) =>
                          toggleModelSelection(modelName, !!value)
                        }
                        aria-label={t('Select {{model}}', {
                          model: modelName,
                        })}
                      />
                      <StatusBadge
                        label={modelName}
                        variant='neutral'
                        copyText={modelName}
                      />
                    </div>
                    <Button
                      size='sm'
                      className='flex-shrink-0 gap-1'
                      onClick={() => handleConfigureModel(modelName)}
                    >
                      <Plus className='h-4 w-4' />
                      {t('Configure')}
                    </Button>
                  </div>
                ))}
              </div>

              <div className='bg-muted/40 flex items-center justify-between border-t px-3 py-2 text-sm'>
                <div className='text-muted-foreground text-sm'>
                  {t('Page {{current}} of {{total}}', {
                    current: currentPage,
                    total: totalPages,
                  })}
                </div>
                {showPagination && (
                  <div className='flex items-center gap-2'>
                    <Button
                      variant='outline'
                      size='icon'
                      className='h-8 w-8'
                      onClick={() =>
                        setCurrentPage((prev) => Math.max(1, prev - 1))
                      }
                      disabled={currentPage === 1}
                      aria-label={t('Previous page')}
                    >
                      <ChevronLeft className='h-4 w-4' />
                    </Button>
                    <Button
                      variant='outline'
                      size='icon'
                      className='h-8 w-8'
                      onClick={() =>
                        setCurrentPage((prev) => Math.min(totalPages, prev + 1))
                      }
                      disabled={currentPage === totalPages}
                      aria-label={t('Next page')}
                    >
                      <ChevronRight className='h-4 w-4' />
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
