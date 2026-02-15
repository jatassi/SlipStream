import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import * as adminApi from '@/api/admin'
import { createQueryKeys } from '@/lib/query-keys'
import type { AdminUpdateUserInput, QuotaLimits } from '@/types'

export const adminUserKeys = createQueryKeys('admin', 'users')

export function useAdminUsers() {
  return useQuery({
    queryKey: adminUserKeys.list(),
    queryFn: () => adminApi.listUsers(),
  })
}

export function useAdminUser(id: number) {
  return useQuery({
    queryKey: adminUserKeys.detail(id),
    queryFn: () => adminApi.getUser(id),
    enabled: !!id,
  })
}

export function useUpdateAdminUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: AdminUpdateUserInput }) =>
      adminApi.updateUser(id, data),
    onSuccess: (user) => {
      void queryClient.invalidateQueries({ queryKey: adminUserKeys.all })
      queryClient.setQueryData(adminUserKeys.detail(user.id), user)
    },
  })
}

export function useEnableUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => adminApi.enableUser(id),
    onSuccess: (user) => {
      void queryClient.invalidateQueries({ queryKey: adminUserKeys.all })
      queryClient.setQueryData(adminUserKeys.detail(user.id), user)
    },
  })
}

export function useDisableUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => adminApi.disableUser(id),
    onSuccess: (user) => {
      void queryClient.invalidateQueries({ queryKey: adminUserKeys.all })
      queryClient.setQueryData(adminUserKeys.detail(user.id), user)
    },
  })
}

export function useDeleteAdminUser() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => adminApi.deleteUser(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminUserKeys.all })
    },
  })
}

export function useSetUserQuota() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, quota }: { id: number; quota: QuotaLimits }) =>
      adminApi.setUserQuota(id, quota),
    onSuccess: (user) => {
      void queryClient.invalidateQueries({ queryKey: adminUserKeys.all })
      queryClient.setQueryData(adminUserKeys.detail(user.id), user)
    },
  })
}
