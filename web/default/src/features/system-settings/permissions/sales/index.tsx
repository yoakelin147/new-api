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
import { UserPlus, UsersRound, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Card,
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
  removeSalesCustomer,
} from './api'
import type { ManagedCustomer, SalesUser } from './types'

export function SalesManagement() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [keyword, setKeyword] = useState('')
  const [customerKeyword, setCustomerKeyword] = useState('')
  const [selectedSales, setSelectedSales] = useState<SalesUser | null>(null)
  const [selectedCustomerIds, setSelectedCustomerIds] = useState<Set<number>>(
    new Set()
  )
  const [removeTarget, setRemoveTarget] = useState<ManagedCustomer | null>(null)
  const debouncedKeyword = useDebounce(keyword, 300)
  const debouncedCustomerKeyword = useDebounce(customerKeyword, 300)

  const salesQuery = useQuery({
    queryKey: ['sales-users', debouncedKeyword],
    queryFn: () => getSalesUsers(debouncedKeyword),
  })
  const assignedQuery = useQuery({
    queryKey: ['sales-customers', selectedSales?.id],
    queryFn: () => getSalesCustomers(selectedSales!.id),
    enabled: selectedSales !== null,
  })
  const availableQuery = useQuery({
    queryKey: ['available-sales-customers', debouncedCustomerKeyword],
    queryFn: () => getAvailableSalesCustomers(debouncedCustomerKeyword),
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
      await refreshAssignments()
    },
    onError: () => toast.error(t('Failed to remove customer')),
  })

  const salesUsers = salesQuery.data?.data?.items ?? []
  const assignedCustomers = assignedQuery.data?.data ?? []
  const availableCustomers = availableQuery.data?.data?.items ?? []
  const candidateColumns = useMemo(
    () => [
      {
        id: 'select',
        header: t('Select'),
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
    [selectedCustomerIds, t]
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
                onChange={(event) => setKeyword(event.target.value)}
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
                            setCustomerKeyword('')
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
            </CardContent>
          </Card>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <Sheet
        open={selectedSales !== null}
        onOpenChange={(open) => {
          if (!open) setSelectedSales(null)
        }}
      >
        <SheetContent className='flex w-full flex-col sm:max-w-2xl'>
          <SheetHeader>
            <SheetTitle>{t('Manage assigned customers')}</SheetTitle>
            <SheetDescription>
              {t('Sales account')}: {selectedSales?.username ?? '-'}
            </SheetDescription>
          </SheetHeader>
          <div className='flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-4'>
            <Card>
              <CardHeader>
                <CardTitle>{t('Assigned customers')}</CardTitle>
                <CardDescription>
                  {t('Removing a customer revokes sales access immediately.')}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <StaticDataTable
                  data={assignedCustomers}
                  getRowKey={(customer) => customer.id}
                  emptyContent={t('No customers assigned yet.')}
                  columns={[
                    {
                      id: 'customer',
                      header: t('Customer'),
                      cell: (customer) => customer.username,
                    },
                    {
                      id: 'status',
                      header: t('Status'),
                      cell: (customer) => (
                        <StatusBadge
                          label={
                            customer.status === 1 ? t('Enabled') : t('Disabled')
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
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>{t('Add customers')}</CardTitle>
                <CardDescription>
                  {t('Only unassigned regular users are listed.')}
                </CardDescription>
              </CardHeader>
              <CardContent className='flex flex-col gap-3'>
                <Input
                  value={customerKeyword}
                  onChange={(event) => setCustomerKeyword(event.target.value)}
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
    </>
  )
}
