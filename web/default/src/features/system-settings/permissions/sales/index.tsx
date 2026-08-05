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
import { useDebounce } from '@/hooks'
import { UserMinus, UserPlus, UsersRound, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Spinner } from '@/components/ui/spinner'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { StaticDataTable } from '@/components/data-table'
import { SectionPageLayout } from '@/components/layout'
import { StatusBadge } from '@/components/status-badge'
import {
  assignSalesCustomers,
  getAvailableSalesCustomers,
  getSalesCustomers,
  getSalesUsers,
  removeSalesCustomers,
  removeSalesCustomer,
} from './api'
import type { ManagedCustomer, SalesUser } from './types'

const SALES_PAGE_SIZE = 10
const CUSTOMER_PAGE_SIZE = 5

function CompactPagination(props: {
  page: number
  total: number
  pageSize: number
  onPageChange: (page: number) => void
  className?: string
}) {
  const { t } = useTranslation()
  const totalPages = Math.max(1, Math.ceil(props.total / props.pageSize))
  if (props.total <= props.pageSize) return null

  return (
    <div
      className={cn(
        'flex flex-col items-center justify-between gap-2 sm:flex-row',
        props.className
      )}
    >
      <span className='text-muted-foreground text-xs'>
        {t('Page {{current}} of {{total}}', {
          current: props.page,
          total: totalPages,
        })}
      </span>
      <div className='flex items-center gap-2'>
        <Button
          type='button'
          variant='outline'
          size='sm'
          disabled={props.page <= 1}
          onClick={() => props.onPageChange(Math.max(1, props.page - 1))}
        >
          {t('Previous page')}
        </Button>
        <Button
          type='button'
          variant='outline'
          size='sm'
          disabled={props.page >= totalPages}
          onClick={() =>
            props.onPageChange(Math.min(totalPages, props.page + 1))
          }
        >
          {t('Next page')}
        </Button>
      </div>
    </div>
  )
}

export function SalesManagement() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [keyword, setKeyword] = useState('')
  const [customerKeyword, setCustomerKeyword] = useState('')
  const [salesPage, setSalesPage] = useState(1)
  const [assignedPage, setAssignedPage] = useState(1)
  const [availablePage, setAvailablePage] = useState(1)
  const [selectedSales, setSelectedSales] = useState<SalesUser | null>(null)
  const [selectedCustomerIds, setSelectedCustomerIds] = useState<Set<number>>(
    new Set()
  )
  const [selectedAssignedCustomerIds, setSelectedAssignedCustomerIds] =
    useState<Set<number>>(new Set())
  const [removeTarget, setRemoveTarget] = useState<ManagedCustomer | null>(null)
  const [bulkRemoveOpen, setBulkRemoveOpen] = useState(false)
  const debouncedKeyword = useDebounce(keyword, 300)
  const debouncedCustomerKeyword = useDebounce(customerKeyword, 300)

  const salesQuery = useQuery({
    queryKey: ['sales-users', debouncedKeyword, salesPage],
    queryFn: () => getSalesUsers(debouncedKeyword, salesPage, SALES_PAGE_SIZE),
  })
  const assignedQuery = useQuery({
    queryKey: ['sales-customers', selectedSales?.id, assignedPage],
    queryFn: () =>
      getSalesCustomers(selectedSales!.id, assignedPage, CUSTOMER_PAGE_SIZE),
    enabled: selectedSales !== null,
  })
  const availableQuery = useQuery({
    queryKey: [
      'available-sales-customers',
      selectedSales?.id,
      debouncedCustomerKeyword,
      availablePage,
    ],
    queryFn: () =>
      getAvailableSalesCustomers(
        selectedSales!.id,
        debouncedCustomerKeyword,
        availablePage,
        CUSTOMER_PAGE_SIZE
      ),
    enabled: selectedSales !== null,
  })

  const refreshAssignments = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['sales-users'] }),
      queryClient.invalidateQueries({ queryKey: ['sales-customers'] }),
      queryClient.invalidateQueries({
        queryKey: ['available-sales-customers'],
      }),
    ])
  }

  const assignMutation = useMutation({
    mutationFn: () =>
      assignSalesCustomers(selectedSales!.id, Array.from(selectedCustomerIds)),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to assign customers'))
        return
      }
      toast.success(t('Customers assigned successfully'))
      setSelectedCustomerIds(new Set())
      setSelectedAssignedCustomerIds(new Set())
      setAssignedPage(1)
      setAvailablePage(1)
      await refreshAssignments()
    },
    onError: () => toast.error(t('Failed to assign customers')),
  })

  const removeMutation = useMutation({
    mutationFn: (customer: ManagedCustomer) =>
      removeSalesCustomer(selectedSales!.id, customer.id),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to remove customer'))
        return
      }
      toast.success(t('Customer removed successfully'))
      setRemoveTarget(null)
      setSelectedAssignedCustomerIds((current) => {
        const next = new Set(current)
        if (removeTarget) next.delete(removeTarget.id)
        return next
      })
      if (
        assignedPage > 1 &&
        (assignedQuery.data?.data?.items.length ?? 0) === 1
      ) {
        setAssignedPage((page) => page - 1)
      }
      await refreshAssignments()
    },
    onError: () => toast.error(t('Failed to remove customer')),
  })

  const bulkRemoveMutation = useMutation({
    mutationFn: () =>
      removeSalesCustomers(
        selectedSales!.id,
        Array.from(selectedAssignedCustomerIds)
      ),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to remove customers'))
        return
      }
      toast.success(t('Customers removed successfully'))
      setBulkRemoveOpen(false)
      setSelectedAssignedCustomerIds(new Set())
      const remainingAssignedTotal = Math.max(
        0,
        assignedTotal - selectedAssignedCustomerIds.size
      )
      const remainingAssignedPages = Math.max(
        1,
        Math.ceil(remainingAssignedTotal / CUSTOMER_PAGE_SIZE)
      )
      if (assignedPage > remainingAssignedPages) {
        setAssignedPage(remainingAssignedPages)
      }
      await refreshAssignments()
    },
    onError: () => toast.error(t('Failed to remove customers')),
  })

  const salesUsers = salesQuery.data?.data?.items ?? []
  const salesTotal = salesQuery.data?.data?.total ?? 0
  const assignedCustomers = assignedQuery.data?.data?.items ?? []
  const assignedTotal = assignedQuery.data?.data?.total ?? 0
  const availableCustomers = useMemo(
    () => availableQuery.data?.data?.items ?? [],
    [availableQuery.data]
  )
  const availableTotal = availableQuery.data?.data?.total ?? 0
  const allAssignedPageSelected =
    assignedCustomers.length > 0 &&
    assignedCustomers.every((customer) =>
      selectedAssignedCustomerIds.has(customer.id)
    )
  const someAssignedPageSelected =
    !allAssignedPageSelected &&
    assignedCustomers.some((customer) =>
      selectedAssignedCustomerIds.has(customer.id)
    )
  const allAvailablePageSelected =
    availableCustomers.length > 0 &&
    availableCustomers.every((customer) => selectedCustomerIds.has(customer.id))
  const someAvailablePageSelected =
    !allAvailablePageSelected &&
    availableCustomers.some((customer) => selectedCustomerIds.has(customer.id))
  const candidateColumns = useMemo(
    () => [
      {
        id: 'select',
        header: (
          <div className='flex items-center gap-2'>
            <Checkbox
              aria-label={t('Select all')}
              checked={allAvailablePageSelected}
              indeterminate={someAvailablePageSelected}
              onCheckedChange={(checked) => {
                setSelectedCustomerIds((current) => {
                  const next = new Set(current)
                  for (const customer of availableCustomers) {
                    if (checked) next.add(customer.id)
                    else next.delete(customer.id)
                  }
                  return next
                })
              }}
            />
            <span>{t('Select')}</span>
          </div>
        ),
        cell: (customer: ManagedCustomer) => (
          <Checkbox
            aria-label={t('Select customer')}
            checked={selectedCustomerIds.has(customer.id)}
            onCheckedChange={(checked) => {
              setSelectedCustomerIds((current) => {
                const next = new Set(current)
                if (checked) next.add(customer.id)
                else next.delete(customer.id)
                return next
              })
            }}
          />
        ),
      },
      {
        id: 'customer',
        header: t('Customer'),
        cell: (customer: ManagedCustomer) => (
          <div className='flex min-w-0 flex-col gap-0.5'>
            <span className='truncate font-medium'>{customer.username}</span>
            {customer.display_name &&
              customer.display_name !== customer.username && (
                <span className='text-muted-foreground truncate text-xs'>
                  {customer.display_name}
                </span>
              )}
          </div>
        ),
      },
      {
        id: 'status',
        header: t('Status'),
        cell: (customer: ManagedCustomer) => (
          <StatusBadge
            label={customer.status === 1 ? t('Enabled') : t('Disabled')}
            variant={customer.status === 1 ? 'success' : 'neutral'}
            copyable={false}
          />
        ),
      },
    ],
    [
      allAvailablePageSelected,
      availableCustomers,
      selectedCustomerIds,
      someAvailablePageSelected,
      t,
    ]
  )

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>
          {t('Sales Management')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <Card>
            <CardHeader>
              <CardTitle>{t('Sales customer assignments')}</CardTitle>
              <CardDescription>
                {t(
                  'Assign regular users to sales accounts. Sales can only view usage logs for their assigned customers.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className='flex flex-col gap-4'>
              <Input
                value={keyword}
                onChange={(event) => {
                  setKeyword(event.target.value)
                  setSalesPage(1)
                }}
                placeholder={t('Search sales by username, name or email')}
                className='max-w-md'
              />
              {salesQuery.isLoading ? (
                <div className='text-muted-foreground flex min-h-32 items-center justify-center gap-2 text-sm'>
                  <Spinner />
                  {t('Loading sales accounts...')}
                </div>
              ) : (
                <StaticDataTable
                  data={salesUsers}
                  getRowKey={(sales) => sales.id}
                  emptyContent={t('No sales accounts found.')}
                  columns={[
                    {
                      id: 'sales',
                      header: t('Sales account'),
                      cell: (sales) => (
                        <div className='flex min-w-0 flex-col gap-0.5'>
                          <span className='truncate font-medium'>
                            {sales.username}
                          </span>
                          {sales.display_name &&
                            sales.display_name !== sales.username && (
                              <span className='text-muted-foreground truncate text-xs'>
                                {sales.display_name}
                              </span>
                            )}
                        </div>
                      ),
                    },
                    {
                      id: 'status',
                      header: t('Status'),
                      cell: (sales) => (
                        <StatusBadge
                          label={
                            sales.status === 1 ? t('Enabled') : t('Disabled')
                          }
                          variant={sales.status === 1 ? 'success' : 'neutral'}
                          copyable={false}
                        />
                      ),
                    },
                    {
                      id: 'customers',
                      header: t('Assigned customers'),
                      cell: (sales) => sales.customer_count,
                    },
                    {
                      id: 'actions',
                      header: t('Actions'),
                      className: 'text-right',
                      cellClassName: 'text-right',
                      cell: (sales) => (
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => {
                            setSelectedSales(sales)
                            setSelectedCustomerIds(new Set())
                            setSelectedAssignedCustomerIds(new Set())
                            setCustomerKeyword('')
                            setAssignedPage(1)
                            setAvailablePage(1)
                          }}
                        >
                          <UsersRound data-icon='inline-start' />
                          {t('Manage customers')}
                        </Button>
                      ),
                    },
                  ]}
                />
              )}
              <CompactPagination
                page={salesPage}
                total={salesTotal}
                pageSize={SALES_PAGE_SIZE}
                onPageChange={setSalesPage}
              />
            </CardContent>
          </Card>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <Sheet
        open={selectedSales !== null}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedSales(null)
            setSelectedCustomerIds(new Set())
            setSelectedAssignedCustomerIds(new Set())
          }
        }}
      >
        <SheetContent className='flex w-full flex-col sm:max-w-2xl'>
          <SheetHeader>
            <SheetTitle>{t('Manage assigned customers')}</SheetTitle>
            <SheetDescription>
              {t('Sales account')}: {selectedSales?.username ?? '-'}
            </SheetDescription>
          </SheetHeader>
          <div className='flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto px-4 pb-1'>
            <Card size='sm' className='shrink-0'>
              <CardHeader>
                <CardTitle>{t('Assigned customers')}</CardTitle>
                <CardDescription>
                  {t('Removing a customer revokes sales access immediately.')}
                </CardDescription>
                <CardAction>
                  <div className='flex items-center gap-2'>
                    {selectedAssignedCustomerIds.size > 0 && (
                      <Button
                        type='button'
                        variant='destructive'
                        size='sm'
                        onClick={() => setBulkRemoveOpen(true)}
                      >
                        <UserMinus data-icon='inline-start' />
                        {t('Remove selected')} (
                        {selectedAssignedCustomerIds.size})
                      </Button>
                    )}
                    <CompactPagination
                      page={assignedPage}
                      total={assignedTotal}
                      pageSize={CUSTOMER_PAGE_SIZE}
                      onPageChange={setAssignedPage}
                    />
                  </div>
                </CardAction>
              </CardHeader>
              <CardContent className='pb-3'>
                {assignedQuery.isLoading ? (
                  <div className='text-muted-foreground flex min-h-24 items-center justify-center gap-2 text-sm'>
                    <Spinner />
                    {t('Loading assigned customers...')}
                  </div>
                ) : (
                  <StaticDataTable
                    data={assignedCustomers}
                    getRowKey={(customer) => customer.id}
                    emptyContent={t('No customers assigned yet.')}
                    columns={[
                      {
                        id: 'customer',
                        header: (
                          <div className='flex items-center gap-2'>
                            <Checkbox
                              aria-label={t('Select all')}
                              checked={allAssignedPageSelected}
                              indeterminate={someAssignedPageSelected}
                              onCheckedChange={(checked) => {
                                setSelectedAssignedCustomerIds((current) => {
                                  const next = new Set(current)
                                  for (const customer of assignedCustomers) {
                                    if (checked) next.add(customer.id)
                                    else next.delete(customer.id)
                                  }
                                  return next
                                })
                              }}
                            />
                            <span>{t('Customer')}</span>
                          </div>
                        ),
                        cell: (customer) => (
                          <div className='flex min-w-0 items-center gap-2'>
                            <Checkbox
                              aria-label={t('Select customer')}
                              checked={selectedAssignedCustomerIds.has(
                                customer.id
                              )}
                              onCheckedChange={(checked) => {
                                setSelectedAssignedCustomerIds((current) => {
                                  const next = new Set(current)
                                  if (checked) next.add(customer.id)
                                  else next.delete(customer.id)
                                  return next
                                })
                              }}
                            />
                            <span className='truncate'>
                              {customer.username}
                            </span>
                          </div>
                        ),
                      },
                      {
                        id: 'status',
                        header: t('Status'),
                        cell: (customer) => (
                          <StatusBadge
                            label={
                              customer.status === 1
                                ? t('Enabled')
                                : t('Disabled')
                            }
                            variant={
                              customer.status === 1 ? 'success' : 'neutral'
                            }
                            copyable={false}
                          />
                        ),
                      },
                      {
                        id: 'actions',
                        header: t('Actions'),
                        className: 'text-right',
                        cellClassName: 'text-right',
                        cell: (customer) => (
                          <Button
                            variant='ghost'
                            size='icon-sm'
                            aria-label={t('Remove customer')}
                            onClick={() => setRemoveTarget(customer)}
                          >
                            <X />
                          </Button>
                        ),
                      },
                    ]}
                  />
                )}
              </CardContent>
            </Card>

            <Card size='sm' className='shrink-0'>
              <CardHeader>
                <CardTitle>{t('Add customers')}</CardTitle>
                <CardDescription>
                  {t(
                    'Only regular users not yet assigned to this sales account are listed.'
                  )}
                </CardDescription>
                <CardAction>
                  <CompactPagination
                    page={availablePage}
                    total={availableTotal}
                    pageSize={CUSTOMER_PAGE_SIZE}
                    onPageChange={setAvailablePage}
                  />
                </CardAction>
              </CardHeader>
              <CardContent className='flex flex-col gap-3 pb-3'>
                <Input
                  value={customerKeyword}
                  onChange={(event) => {
                    setCustomerKeyword(event.target.value)
                    setAvailablePage(1)
                    setSelectedCustomerIds(new Set())
                  }}
                  placeholder={t('Search customers by username, name or email')}
                />
                {availableQuery.isLoading ? (
                  <div className='text-muted-foreground flex min-h-24 items-center justify-center gap-2 text-sm'>
                    <Spinner />
                    {t('Loading available customers...')}
                  </div>
                ) : (
                  <StaticDataTable
                    data={availableCustomers}
                    getRowKey={(customer) => customer.id}
                    emptyContent={t('No available customers found.')}
                    columns={candidateColumns}
                  />
                )}
              </CardContent>
            </Card>
          </div>
          <SheetFooter>
            <Button
              onClick={() => assignMutation.mutate()}
              disabled={
                selectedCustomerIds.size === 0 || assignMutation.isPending
              }
            >
              {assignMutation.isPending ? (
                <Spinner data-icon='inline-start' />
              ) : (
                <UserPlus data-icon='inline-start' />
              )}
              {t('Assign selected customers')}
              {selectedCustomerIds.size > 0 && ` (${selectedCustomerIds.size})`}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={removeTarget !== null}
        onOpenChange={(open) => !open && setRemoveTarget(null)}
        title={t('Remove customer assignment')}
        desc={t(
          'Remove customer "{{name}}" from this sales account? Sales access to the customer logs will be revoked immediately.',
          { name: removeTarget?.username ?? '' }
        )}
        confirmText={t('Remove')}
        destructive
        handleConfirm={() =>
          removeTarget && removeMutation.mutate(removeTarget)
        }
        isLoading={removeMutation.isPending}
      />

      <ConfirmDialog
        open={bulkRemoveOpen}
        onOpenChange={(open) => !open && setBulkRemoveOpen(false)}
        title={t('Remove selected customers')}
        desc={t(
          'Remove {{count}} selected customers from this sales account? Sales access to their logs will be revoked immediately.',
          { count: selectedAssignedCustomerIds.size }
        )}
        confirmText={t('Remove')}
        destructive
        handleConfirm={() => bulkRemoveMutation.mutate()}
        isLoading={bulkRemoveMutation.isPending}
      />
    </>
  )
}
