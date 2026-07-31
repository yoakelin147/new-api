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
import { Copy, KeyRound, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { StaticDataTable } from '@/components/data-table'
import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import {
  useCreateMonitoringAccessKey,
  useDeleteMonitoringAccessKey,
  useMonitoringAccessKeys,
  useUpdateMonitoringAccessKeyStatus,
} from './hooks'
import type { MonitoringAccessKey } from './types'

type ExpiryChoice = 'day' | 'week' | 'custom' | 'permanent'

const monitoringEndpoint =
  '/api/perf-metrics/channel-summary?minutes=5&limit=20'

function formatTime(timestamp: number, locale: string) {
  return new Intl.DateTimeFormat(locale, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(timestamp * 1000))
}

function isExpired(key: MonitoringAccessKey) {
  return key.expires_at !== null && key.expires_at <= Date.now() / 1000
}

export function MonitoringAccessKeyManagement() {
  const { t, i18n } = useTranslation()
  const { copyToClipboard } = useCopyToClipboard()
  const { data: keys = [], isLoading } = useMonitoringAccessKeys()
  const createKey = useCreateMonitoringAccessKey()
  const updateStatus = useUpdateMonitoringAccessKeyStatus()
  const deleteKey = useDeleteMonitoringAccessKey()
  const [name, setName] = useState('Zabbix')
  const [expiryChoice, setExpiryChoice] = useState<ExpiryChoice>('permanent')
  const [customExpiry, setCustomExpiry] = useState('')
  const [generatedKey, setGeneratedKey] = useState<string | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<MonitoringAccessKey | null>(
    null
  )

  const curlExample = useMemo(
    () =>
      `curl -H "Authorization: Bearer <monitoring-key>" "${typeof window === 'undefined' ? '' : window.location.origin}${monitoringEndpoint}"`,
    []
  )

  const handleCreate = async () => {
    const trimmedName = name.trim()
    if (!trimmedName) {
      toast.error(t('Please enter a key name'))
      return
    }

    let expiresAt: number | null = null
    if (expiryChoice === 'day')
      expiresAt = Math.floor(Date.now() / 1000) + 86400
    if (expiryChoice === 'week')
      expiresAt = Math.floor(Date.now() / 1000) + 7 * 86400
    if (expiryChoice === 'custom') {
      expiresAt = Math.floor(new Date(customExpiry).getTime() / 1000)
      if (
        !customExpiry ||
        !Number.isFinite(expiresAt) ||
        expiresAt <= Date.now() / 1000
      ) {
        toast.error(t('Please select a future expiration time'))
        return
      }
    }

    const response = await createKey.mutateAsync({
      name: trimmedName,
      expires_at: expiresAt,
    })
    if (response.success && response.data?.key) {
      setGeneratedKey(response.data.key)
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    await deleteKey.mutateAsync(deleteTarget.id)
    setDeleteTarget(null)
  }

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Monitoring Management')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <div className='flex w-full flex-col gap-4'>
            <Card>
              <CardHeader>
                <CardTitle>{t('Generate monitoring key')}</CardTitle>
                <CardDescription>
                  {t(
                    'Create a dedicated access key for Zabbix or another monitoring collector.'
                  )}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <FieldGroup>
                  <Field>
                    <FieldLabel htmlFor='monitoring-key-name'>
                      {t('Key name')}
                    </FieldLabel>
                    <Input
                      id='monitoring-key-name'
                      value={name}
                      maxLength={128}
                      onChange={(event) => setName(event.target.value)}
                      placeholder={t('For example: Production Zabbix')}
                    />
                  </Field>
                  <Field>
                    <FieldLabel>{t('Validity period')}</FieldLabel>
                    <ToggleGroup
                      value={[expiryChoice]}
                      onValueChange={(value) => {
                        const nextValue = value.at(-1) as
                          | ExpiryChoice
                          | undefined
                        if (nextValue) setExpiryChoice(nextValue)
                      }}
                      variant='outline'
                      className='flex-wrap'
                    >
                      <ToggleGroupItem value='day'>
                        {t('1 day')}
                      </ToggleGroupItem>
                      <ToggleGroupItem value='week'>
                        {t('7 days')}
                      </ToggleGroupItem>
                      <ToggleGroupItem value='custom'>
                        {t('Custom')}
                      </ToggleGroupItem>
                      <ToggleGroupItem value='permanent'>
                        {t('Permanent')}
                      </ToggleGroupItem>
                    </ToggleGroup>
                    <FieldDescription>
                      {t('Expired or disabled keys are rejected immediately.')}
                    </FieldDescription>
                  </Field>
                  {expiryChoice === 'custom' && (
                    <Field>
                      <FieldLabel htmlFor='monitoring-key-expiry'>
                        {t('Expiration time')}
                      </FieldLabel>
                      <Input
                        id='monitoring-key-expiry'
                        type='datetime-local'
                        value={customExpiry}
                        onChange={(event) =>
                          setCustomExpiry(event.target.value)
                        }
                      />
                    </Field>
                  )}
                  <div>
                    <Button
                      onClick={handleCreate}
                      disabled={createKey.isPending}
                    >
                      {createKey.isPending ? (
                        <Spinner data-icon='inline-start' />
                      ) : (
                        <KeyRound data-icon='inline-start' />
                      )}
                      {t('Generate key')}
                    </Button>
                  </div>
                </FieldGroup>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>{t('Monitoring keys')}</CardTitle>
                <CardDescription>
                  {t(
                    'Only the prefix is retained for identification. The full key cannot be viewed again.'
                  )}
                </CardDescription>
              </CardHeader>
              <CardContent>
                {isLoading ? (
                  <div className='text-muted-foreground flex min-h-24 items-center justify-center gap-2 text-sm'>
                    <Spinner />
                    {t('Loading monitoring keys...')}
                  </div>
                ) : (
                  <StaticDataTable
                    data={keys}
                    getRowKey={(key) => key.id}
                    emptyContent={t('No monitoring keys yet.')}
                    columns={[
                      {
                        id: 'name',
                        header: t('Name'),
                        cell: (key) => key.name,
                      },
                      {
                        id: 'prefix',
                        header: t('Key prefix'),
                        cellClassName: 'font-mono',
                        cell: (key) => `${key.key_prefix}••••••••`,
                      },
                      {
                        id: 'status',
                        header: t('Status'),
                        cell: (key) => {
                          const expired = isExpired(key)
                          return (
                            <StatusBadge
                              label={
                                expired
                                  ? t('Expired')
                                  : key.status === 1
                                    ? t('Enabled')
                                    : t('Disabled')
                              }
                              variant={
                                expired
                                  ? 'warning'
                                  : key.status === 1
                                    ? 'success'
                                    : 'neutral'
                              }
                              copyable={false}
                            />
                          )
                        },
                      },
                      {
                        id: 'created',
                        header: t('Created at'),
                        cell: (key) =>
                          formatTime(key.created_time, i18n.language),
                      },
                      {
                        id: 'expires',
                        header: t('Expires at'),
                        cell: (key) =>
                          key.expires_at
                            ? formatTime(key.expires_at, i18n.language)
                            : t('Never'),
                      },
                      {
                        id: 'last-used',
                        header: t('Last used'),
                        cell: (key) =>
                          key.last_used_time > 0
                            ? formatTime(key.last_used_time, i18n.language)
                            : t('Never'),
                      },
                      {
                        id: 'actions',
                        header: t('Actions'),
                        className: 'text-right',
                        cellClassName: 'text-right',
                        cell: (key) => (
                          <div className='flex items-center justify-end gap-2'>
                            <Switch
                              aria-label={t('Enable monitoring key')}
                              checked={key.status === 1}
                              disabled={
                                isExpired(key) || updateStatus.isPending
                              }
                              onCheckedChange={(checked) =>
                                updateStatus.mutate({
                                  id: key.id,
                                  enabled: checked,
                                })
                              }
                            />
                            <Button
                              variant='ghost'
                              size='icon-sm'
                              aria-label={t('Delete monitoring key')}
                              onClick={() => setDeleteTarget(key)}
                            >
                              <Trash2 className='text-destructive' />
                            </Button>
                          </div>
                        ),
                      },
                    ]}
                  />
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>{t('Monitoring API')}</CardTitle>
                <CardDescription>
                  {t(
                    'This endpoint returns model and channel availability for recently active combinations.'
                  )}
                </CardDescription>
              </CardHeader>
              <CardContent className='flex flex-col gap-3'>
                <div className='bg-muted rounded-lg p-3 font-mono text-sm break-all'>
                  GET {monitoringEndpoint}
                </div>
                <div className='bg-muted relative rounded-lg p-3 pr-12 font-mono text-sm break-all'>
                  {curlExample}
                  <Button
                    variant='ghost'
                    size='icon-sm'
                    className='absolute top-2 right-2'
                    aria-label={t('Copy curl command')}
                    onClick={() => copyToClipboard(curlExample)}
                  >
                    <Copy />
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <Dialog
        open={generatedKey !== null}
        onOpenChange={(open) => {
          if (!open) setGeneratedKey(null)
        }}
      >
        <DialogContent className='sm:max-w-lg'>
          <DialogHeader>
            <DialogTitle>{t('Monitoring key created')}</DialogTitle>
            <DialogDescription>
              {t(
                'Copy this key now. For security, it will only be shown once.'
              )}
            </DialogDescription>
          </DialogHeader>
          <div className='bg-muted rounded-lg p-3 font-mono text-sm break-all'>
            {generatedKey}
          </div>
          <DialogFooter>
            <Button
              onClick={() => generatedKey && copyToClipboard(generatedKey)}
            >
              <Copy data-icon='inline-start' />
              {t('Copy key')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t('Delete monitoring key')}
        desc={t(
          'Delete monitoring key "{{name}}"? Any collector using it will immediately lose access.',
          { name: deleteTarget?.name ?? '' }
        )}
        confirmText={t('Delete')}
        destructive
        handleConfirm={handleDelete}
        isLoading={deleteKey.isPending}
      />
    </>
  )
}
