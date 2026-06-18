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
import { useMemo, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Download, Loader2, RefreshCw, Upload } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { useIsMobile } from '@/hooks/use-mobile'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Textarea } from '@/components/ui/textarea'
import { Dialog } from '@/components/dialog'
import { StatusBadge } from '@/components/status-badge'
import {
  syncUpstream,
  previewUpstreamDiff,
  downloadUpstreamConfigExample,
} from '../../api'
import { getSyncLocaleOptions, getSyncSourceOptions } from '../../constants'
import { modelsQueryKeys, vendorsQueryKeys } from '../../lib'
import type { SyncLocale, SyncSource } from '../../types'
import { useModels } from '../models-provider'

type SyncWizardDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function SyncWizardDialog({
  open,
  onOpenChange,
}: SyncWizardDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const {
    setOpen,
    setUpstreamConflicts,
    setSyncWizardOptions,
    syncWizardOptions,
  } = useModels()
  const isMobile = useIsMobile()
  const SYNC_SOURCE_OPTIONS = useMemo(() => getSyncSourceOptions(t), [t])
  const SYNC_LOCALE_OPTIONS = useMemo(() => getSyncLocaleOptions(t), [t])
  const initialSource = useMemo(() => {
    const preferredSource = SYNC_SOURCE_OPTIONS.find(
      (option) => option.value === syncWizardOptions.source
    )
    return preferredSource && !preferredSource.disabled
      ? (preferredSource.value as SyncSource)
      : 'official'
  }, [SYNC_SOURCE_OPTIONS, syncWizardOptions.source])
  const [locale, setLocale] = useState<SyncLocale>(
    syncWizardOptions.locale || 'zh'
  )
  const [source, setSource] = useState<SyncSource>(initialSource)
  const [configMode, setConfigMode] = useState<'content' | 'url'>(
    syncWizardOptions.config_url ? 'url' : 'content'
  )
  const [configContent, setConfigContent] = useState(
    syncWizardOptions.config_content || ''
  )
  const [configUrl, setConfigUrl] = useState(syncWizardOptions.config_url || '')
  const [isSyncing, setIsSyncing] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const buildSyncOptions = () => ({
    locale,
    source,
    config_content:
      source === 'config' && configMode === 'content'
        ? configContent.trim()
        : undefined,
    config_url:
      source === 'config' && configMode === 'url'
        ? configUrl.trim()
        : undefined,
  })

  const validateConfigInput = () => {
    if (source !== 'config') return true
    if (configMode === 'url') {
      if (!configUrl.trim()) {
        toast.error(t('Configuration URL is required.'))
        return false
      }
      return true
    }
    if (!configContent.trim()) {
      toast.error(t('Configuration JSON content is required.'))
      return false
    }
    return true
  }

  const handleConfigFileChange = async (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    if (file.size > 2 * 1024 * 1024) {
      toast.error(t('Configuration file must be 2 MB or smaller.'))
      return
    }
    try {
      setConfigContent(await file.text())
      setConfigMode('content')
    } catch {
      toast.error(t('Failed to read configuration file.'))
    }
  }

  const handleDownloadExample = async () => {
    try {
      const blob = await downloadUpstreamConfigExample()
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'model-sync-config-example.json'
      document.body.appendChild(link)
      link.click()
      link.remove()
      URL.revokeObjectURL(url)
    } catch {
      toast.error(t('Failed to download configuration example.'))
    }
  }

  const handleSync = async () => {
    setIsSyncing(true)
    try {
      if (!validateConfigInput()) return

      const syncOptions = buildSyncOptions()
      setSyncWizardOptions(syncOptions)
      const previewRes = await previewUpstreamDiff(syncOptions)

      if (!previewRes.success) {
        throw new Error(
          previewRes.message || t('Failed to preview upstream diff')
        )
      }

      const conflicts = previewRes.data?.conflicts || []

      if (conflicts.length > 0) {
        toast.warning(
          t('Found {{count}} conflicts. Please resolve them first.', {
            count: conflicts.length,
          })
        )
        setUpstreamConflicts(conflicts)
        setOpen('upstream-conflict')
        return
      }

      // No conflicts, proceed with sync
      const response = await syncUpstream(syncOptions)

      if (response.success) {
        const { created_models, created_vendors, updated_models } =
          response.data || {}
        toast.success(
          t(
            'Sync completed! Created {{models}} models, updated {{updated}}, and added {{vendors}} vendors.',
            {
              models: created_models || 0,
              updated: updated_models || 0,
              vendors: created_vendors || 0,
            }
          )
        )
        queryClient.invalidateQueries({ queryKey: modelsQueryKeys.lists() })
        queryClient.invalidateQueries({ queryKey: vendorsQueryKeys.lists() })
        onOpenChange(false)
      } else {
        toast.error(response.message || t('Sync failed'))
      }
    } catch (error: unknown) {
      toast.error((error as Error)?.message || t('Sync failed'))
    } finally {
      setIsSyncing(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Sync Upstream Models')}
      description={t('Synchronize models and vendors from an upstream source')}
      initialFocus={!isMobile}
      contentHeight='auto'
      bodyClassName='flex flex-col gap-6'
      footer={
        <>
          <Button
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={isSyncing}
          >
            {t('Cancel')}
          </Button>
          <Button onClick={handleSync} disabled={isSyncing}>
            {isSyncing && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
            <RefreshCw className='mr-2 h-4 w-4' />
            {isSyncing ? t('Syncing...') : t('Sync Now')}
          </Button>
        </>
      }
    >
      <div className='space-y-3'>
        <div>
          <Label className='text-base'>{t('Select Sync Source')}</Label>
          <p className='text-muted-foreground text-sm'>
            {t('Choose where to fetch upstream metadata.')}
          </p>
        </div>
        <RadioGroup
          value={source}
          onValueChange={(value) => {
            const selected = SYNC_SOURCE_OPTIONS.find(
              (option) => option.value === value
            )
            if (!selected || selected.disabled) return
            setSource(selected.value)
          }}
          className='grid gap-3 md:grid-cols-2'
        >
          {SYNC_SOURCE_OPTIONS.map((option) => {
            const isActive = source === option.value
            const isDisabled = option.disabled
            return (
              <Label
                key={option.value}
                htmlFor={`sync-source-${option.value}`}
                className={cn(
                  'flex-col items-start gap-0 rounded-lg border p-4 font-normal transition-all',
                  isActive && 'border-primary ring-primary ring-1',
                  isDisabled
                    ? 'cursor-not-allowed opacity-60'
                    : 'hover:border-primary/60 cursor-pointer'
                )}
              >
                <div className='flex items-start gap-3'>
                  <RadioGroupItem
                    value={option.value}
                    id={`sync-source-${option.value}`}
                    disabled={isDisabled}
                  />
                  <div className='space-y-1'>
                    <div className='flex items-center gap-2'>
                      <span className='font-medium'>{option.label}</span>
                      {option.value === 'official' && (
                        <StatusBadge
                          label={t('Default')}
                          variant='neutral'
                          copyable={false}
                        />
                      )}
                    </div>
                    <p className='text-muted-foreground text-sm'>
                      {option.description}
                    </p>
                  </div>
                </div>
              </Label>
            )
          })}
        </RadioGroup>
      </div>

      {source === 'config' && (
        <div className='flex flex-col gap-3'>
          <div className='flex flex-wrap items-start justify-between gap-2'>
            <div>
              <Label className='text-base'>{t('Configuration Source')}</Label>
              <p className='text-muted-foreground text-sm'>
                {t(
                  'Paste JSON, upload a JSON file, or reference an HTTPS URL.'
                )}
              </p>
            </div>
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={handleDownloadExample}
            >
              <Download data-icon='inline-start' />
              {t('Download example')}
            </Button>
          </div>

          <RadioGroup
            value={configMode}
            onValueChange={(value) => setConfigMode(value as 'content' | 'url')}
            className='grid gap-3 sm:grid-cols-2'
          >
            <Label
              htmlFor='sync-config-content'
              className={cn(
                'cursor-pointer rounded-lg border p-3 font-normal',
                configMode === 'content' && 'border-primary ring-primary ring-1'
              )}
            >
              <div className='flex items-center gap-2'>
                <RadioGroupItem value='content' id='sync-config-content' />
                <span>{t('Upload or paste JSON')}</span>
              </div>
            </Label>
            <Label
              htmlFor='sync-config-url'
              className={cn(
                'cursor-pointer rounded-lg border p-3 font-normal',
                configMode === 'url' && 'border-primary ring-primary ring-1'
              )}
            >
              <div className='flex items-center gap-2'>
                <RadioGroupItem value='url' id='sync-config-url' />
                <span>{t('Reference URL')}</span>
              </div>
            </Label>
          </RadioGroup>

          {configMode === 'content' ? (
            <div className='flex flex-col gap-2'>
              <div className='flex flex-wrap items-center gap-2'>
                <input
                  ref={fileInputRef}
                  type='file'
                  accept='application/json,.json'
                  className='hidden'
                  onChange={handleConfigFileChange}
                />
                <Button
                  type='button'
                  variant='outline'
                  size='sm'
                  onClick={() => fileInputRef.current?.click()}
                >
                  <Upload data-icon='inline-start' />
                  {t('Upload JSON')}
                </Button>
                <span className='text-muted-foreground text-xs'>
                  {t('The JSON content is used only for this sync request.')}
                </span>
              </div>
              <Textarea
                value={configContent}
                onChange={(event) => setConfigContent(event.target.value)}
                placeholder={t('Paste model sync configuration JSON here...')}
                className='min-h-36 font-mono text-xs'
              />
            </div>
          ) : (
            <div className='flex flex-col gap-2'>
              <Input
                value={configUrl}
                onChange={(event) => setConfigUrl(event.target.value)}
                placeholder='https://example.com/model-sync-config.json'
              />
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Remote configuration URLs are validated with the system fetch security settings.'
                )}
              </p>
            </div>
          )}
        </div>
      )}

      <div className='space-y-2'>
        <Label className='text-base'>{t('Select Language')}</Label>
        <RadioGroup
          value={locale}
          onValueChange={(v) => setLocale(v as SyncLocale)}
          className='grid gap-3 sm:grid-cols-3'
        >
          {SYNC_LOCALE_OPTIONS.map((option) => (
            <div
              key={option.value}
              className='flex items-center space-x-2 rounded-lg border p-3'
            >
              <RadioGroupItem
                value={option.value}
                id={`locale-${option.value}`}
              />
              <Label
                htmlFor={`locale-${option.value}`}
                className='cursor-pointer font-normal'
              >
                {option.label}
              </Label>
            </div>
          ))}
        </RadioGroup>
      </div>

      <div className='bg-muted/50 rounded-lg border p-4'>
        <p className='text-muted-foreground text-sm'>
          {t(
            'The sync will fetch missing models and vendors from the selected source. Existing records are updated only when you approve conflicts.'
          )}
        </p>
      </div>
    </Dialog>
  )
}
