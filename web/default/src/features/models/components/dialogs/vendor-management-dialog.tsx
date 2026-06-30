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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Building2,
  Loader2,
  Pencil,
  Plus,
  RefreshCcw,
  Trash2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { useIsMobile } from '@/hooks/use-mobile'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { StaticDataTable } from '@/components/data-table'
import { Dialog } from '@/components/dialog'
import { ProviderBadge } from '@/components/provider-badge'
import { StatusBadge } from '@/components/status-badge'
import { TableId } from '@/components/table-id'
import { deleteVendor, getModels, getVendors } from '../../api'
import { modelsQueryKeys, vendorsQueryKeys } from '../../lib'
import type { Vendor } from '../../types'
import { useModels } from '../models-provider'

type VendorManagementDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function VendorManagementDialog({
  open,
  onOpenChange,
}: VendorManagementDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const isMobile = useIsMobile()
  const { setCurrentVendor, setOpen } = useModels()
  const [deleteState, setDeleteState] = useState<{
    open: boolean
    vendor: Vendor | null
    modelCount: number
  }>({ open: false, vendor: null, modelCount: 0 })
  const [isDeleting, setIsDeleting] = useState(false)

  const {
    data: vendorsData,
    isLoading,
    isFetching,
    error,
    refetch: refetchVendors,
  } = useQuery({
    queryKey: vendorsQueryKeys.list({ page_size: 1000 }),
    queryFn: () => getVendors({ page_size: 1000 }),
    enabled: open,
  })

  const { data: modelData } = useQuery({
    queryKey: modelsQueryKeys.list({ p: 1, page_size: 1 }),
    queryFn: () => getModels({ p: 1, page_size: 1 }),
    enabled: open,
  })

  const vendors = useMemo(
    () => vendorsData?.data?.items || [],
    [vendorsData?.data?.items]
  )

  const vendorCounts = modelData?.data?.vendor_counts || {}

  const sortedVendors = useMemo(
    () => [...vendors].sort((a, b) => a.name.localeCompare(b.name)),
    [vendors]
  )

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      setDeleteState({ open: false, vendor: null, modelCount: 0 })
      setIsDeleting(false)
    }
    onOpenChange(nextOpen)
  }

  const handleCreateVendor = () => {
    setCurrentVendor(null)
    setOpen('create-vendor')
  }

  const handleEditVendor = (vendor: Vendor) => {
    setCurrentVendor(vendor)
    setOpen('update-vendor')
  }

  const handleDeleteClick = (vendor: Vendor) => {
    setDeleteState({
      open: true,
      vendor,
      modelCount: vendorCounts[String(vendor.id)] || 0,
    })
  }

  const handleDeleteConfirm = async () => {
    if (!deleteState.vendor) return
    setIsDeleting(true)
    try {
      const response = await deleteVendor(deleteState.vendor.id)
      if (response.success) {
        toast.success(t('Vendor deleted successfully'))
        queryClient.invalidateQueries({ queryKey: vendorsQueryKeys.lists() })
        queryClient.invalidateQueries({ queryKey: modelsQueryKeys.lists() })
        setDeleteState({ open: false, vendor: null, modelCount: 0 })
      } else {
        toast.error(response.message || t('Failed to delete vendor'))
      }
    } catch (err: unknown) {
      toast.error((err as Error)?.message || t('Failed to delete vendor'))
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <>
      <Dialog
        open={open}
        onOpenChange={handleOpenChange}
        title={
          <>
            <Building2 className='text-foreground/80 h-5 w-5' />
            {t('Manage Vendors')}
          </>
        }
        description={t(
          'Manage model vendors used by metadata, filtering, and model display.'
        )}
        contentClassName={cn(
          'w-[calc(100vw-2rem)] sm:max-w-[56rem]',
          isMobile && 'max-w-none rounded-none'
        )}
        titleClassName='flex flex-wrap items-center gap-2 text-lg'
        descriptionClassName='text-sm leading-relaxed'
        contentHeight='auto'
        bodyClassName={cn(
          'space-y-3',
          isMobile && 'pb-[calc(env(safe-area-inset-bottom,0px)+1rem)]'
        )}
      >
        <div className='bg-muted/30 flex flex-wrap items-center justify-between gap-3 rounded-md border p-2 text-sm'>
          <div className='flex flex-wrap items-center gap-2'>
            <Button size='sm' onClick={handleCreateVendor}>
              <Plus className='mr-2 h-4 w-4' />
              {t('New Vendor')}
            </Button>
            <Button
              size='sm'
              variant='ghost'
              onClick={() => refetchVendors()}
              disabled={isFetching}
            >
              {isFetching ? (
                <Loader2 className='mr-2 h-4 w-4 animate-spin' />
              ) : (
                <RefreshCcw className='mr-2 h-4 w-4' />
              )}
              {t('Refresh')}
            </Button>
          </div>
          <StatusBadge
            label={t('{{count}} vendor(s)', { count: vendors.length })}
            variant='neutral'
            copyable={false}
          />
        </div>

        {error && (
          <Alert variant='destructive'>
            <AlertTitle>{t('Unable to load vendors')}</AlertTitle>
            <AlertDescription>
              {(error as Error).message || t('Please retry or refresh.')}
            </AlertDescription>
          </Alert>
        )}

        {isLoading ? (
          <div className='flex flex-col items-center justify-center gap-2 py-12 text-center'>
            <Loader2 className='text-muted-foreground h-6 w-6 animate-spin' />
            <p className='text-muted-foreground text-sm'>
              {t('Fetching vendors...')}
            </p>
          </div>
        ) : sortedVendors.length === 0 ? (
          <Empty className='border border-dashed py-10'>
            <EmptyMedia variant='icon'>
              <Building2 className='h-6 w-6' />
            </EmptyMedia>
            <EmptyHeader>
              <EmptyTitle>{t('No vendors yet')}</EmptyTitle>
              <EmptyDescription>
                {t('Create a vendor to organize model metadata.')}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : isMobile ? (
          <div className='space-y-2'>
            {sortedVendors.map((vendor) => {
              const count = vendorCounts[String(vendor.id)] || 0
              return (
                <div
                  key={vendor.id}
                  className='border-border/70 flex flex-col gap-3 rounded-md border p-3'
                >
                  <div className='flex items-start justify-between gap-3'>
                    <div className='min-w-0 space-y-2'>
                      <div className='flex flex-wrap items-center gap-2'>
                        <ProviderBadge
                          iconKey={vendor.icon}
                          label={vendor.name}
                        />
                        <TableId value={vendor.id} />
                      </div>
                      <p className='text-muted-foreground line-clamp-2 text-sm'>
                        {vendor.description || t('No description provided')}
                      </p>
                      <StatusBadge
                        label={t('{{count}} model(s)', { count })}
                        variant={count > 0 ? 'info' : 'neutral'}
                        size='sm'
                        copyable={false}
                      />
                    </div>
                    <div className='flex shrink-0 gap-2'>
                      <Button
                        size='icon'
                        variant='outline'
                        onClick={() => handleEditVendor(vendor)}
                      >
                        <Pencil className='h-4 w-4' />
                        <span className='sr-only'>{t('Edit')}</span>
                      </Button>
                      <Button
                        size='icon'
                        variant='ghost'
                        className='text-destructive hover:text-destructive'
                        onClick={() => handleDeleteClick(vendor)}
                      >
                        <Trash2 className='h-4 w-4' />
                        <span className='sr-only'>{t('Delete')}</span>
                      </Button>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        ) : (
          <StaticDataTable
            tableClassName='min-w-[760px]'
            data={sortedVendors}
            getRowKey={(vendor) => vendor.id}
            columns={[
              {
                id: 'vendor',
                header: t('Vendor'),
                cellClassName: 'align-top whitespace-normal',
                cell: (vendor) => (
                  <div className='flex flex-col gap-1'>
                    <div className='flex flex-wrap items-center gap-2'>
                      <ProviderBadge
                        iconKey={vendor.icon}
                        label={vendor.name}
                      />
                      <TableId value={vendor.id} />
                    </div>
                    {vendor.description ? (
                      <p className='text-muted-foreground text-xs'>
                        {vendor.description}
                      </p>
                    ) : (
                      <p className='text-muted-foreground text-xs italic'>
                        {t('No description provided')}
                      </p>
                    )}
                  </div>
                ),
              },
              {
                id: 'status',
                header: t('Status'),
                className: 'w-[120px]',
                cellClassName: 'align-top',
                cell: (vendor) => (
                  <StatusBadge
                    label={vendor.status === 1 ? t('Enabled') : t('Disabled')}
                    variant={vendor.status === 1 ? 'success' : 'neutral'}
                    size='sm'
                    copyable={false}
                  />
                ),
              },
              {
                id: 'models',
                header: t('Models'),
                className: 'w-[120px]',
                cellClassName: 'align-top',
                cell: (vendor) => {
                  const count = vendorCounts[String(vendor.id)] || 0
                  return (
                    <StatusBadge
                      label={String(count)}
                      variant={count > 0 ? 'info' : 'neutral'}
                      size='sm'
                      copyable={false}
                    />
                  )
                },
              },
              {
                id: 'actions',
                header: t('Actions'),
                className: 'w-[120px] text-right',
                cellClassName: 'align-top',
                cell: (vendor) => (
                  <div className='flex justify-end gap-2'>
                    <Button
                      size='icon'
                      variant='outline'
                      onClick={() => handleEditVendor(vendor)}
                    >
                      <Pencil className='h-4 w-4' />
                      <span className='sr-only'>{t('Edit')}</span>
                    </Button>
                    <Button
                      size='icon'
                      variant='ghost'
                      className='text-destructive hover:text-destructive'
                      onClick={() => handleDeleteClick(vendor)}
                    >
                      <Trash2 className='h-4 w-4' />
                      <span className='sr-only'>{t('Delete')}</span>
                    </Button>
                  </div>
                ),
              },
            ]}
          />
        )}
      </Dialog>

      <ConfirmDialog
        open={deleteState.open}
        onOpenChange={(next) =>
          setDeleteState((current) => ({
            open: next,
            vendor: next ? current.vendor : null,
            modelCount: next ? current.modelCount : 0,
          }))
        }
        title={t('Delete Vendor')}
        desc={
          <div className='space-y-2'>
            <p>
              {t('Are you sure you want to delete')}{' '}
              <span className='font-medium'>{deleteState.vendor?.name}</span>
              {t('? This action cannot be undone.')}
            </p>
            {deleteState.modelCount > 0 && (
              <p className='text-destructive text-sm'>
                {t(
                  'This vendor is still referenced by {{count}} model(s). Those models may display without vendor metadata after deletion.',
                  { count: deleteState.modelCount }
                )}
              </p>
            )}
          </div>
        }
        destructive
        confirmText={isDeleting ? t('Deleting...') : t('Delete')}
        isLoading={isDeleting}
        handleConfirm={handleDeleteConfirm}
      />
    </>
  )
}
