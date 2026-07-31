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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  createMonitoringAccessKey,
  deleteMonitoringAccessKey,
  listMonitoringAccessKeys,
  updateMonitoringAccessKeyStatus,
} from './api'

const monitoringAccessKeysQueryKey = ['monitoring-access-keys'] as const

export function useMonitoringAccessKeys() {
  return useQuery({
    queryKey: monitoringAccessKeysQueryKey,
    queryFn: async () => {
      const response = await listMonitoringAccessKeys()
      return response.data ?? []
    },
  })
}

export function useCreateMonitoringAccessKey() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: createMonitoringAccessKey,
    onSuccess: (response) => {
      if (!response.success) return
      toast.success(t('Monitoring key created successfully'))
      void queryClient.invalidateQueries({
        queryKey: monitoringAccessKeysQueryKey,
      })
    },
  })
}

export function useUpdateMonitoringAccessKeyStatus() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, enabled }: { id: number; enabled: boolean }) =>
      updateMonitoringAccessKeyStatus(id, enabled),
    onSuccess: (response) => {
      if (!response.success) return
      toast.success(t('Monitoring key status updated'))
      void queryClient.invalidateQueries({
        queryKey: monitoringAccessKeysQueryKey,
      })
    },
  })
}

export function useDeleteMonitoringAccessKey() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deleteMonitoringAccessKey,
    onSuccess: (response) => {
      if (!response.success) return
      toast.success(t('Monitoring key deleted successfully'))
      void queryClient.invalidateQueries({
        queryKey: monitoringAccessKeysQueryKey,
      })
    },
  })
}
