import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { portalNotificationsApi } from '@/api'
import { createQueryKeys } from '@/lib/query-keys'
import type { CreateUserNotificationInput } from '@/types'

const baseKeys = createQueryKeys('userNotifications')
export const userNotificationKeys = {
  ...baseKeys,
  schema: () => [...baseKeys.all, 'schema'] as const,
}

export function useUserNotifications() {
  return useQuery({
    queryKey: userNotificationKeys.list(),
    queryFn: () => portalNotificationsApi.list(),
  })
}

export function useUserNotification(id: number) {
  return useQuery({
    queryKey: userNotificationKeys.detail(id),
    queryFn: () => portalNotificationsApi.get(id),
    enabled: !!id,
  })
}

export function useCreateUserNotification() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateUserNotificationInput) => portalNotificationsApi.create(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: userNotificationKeys.all })
    },
  })
}

export function useUpdateUserNotification() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: CreateUserNotificationInput }) =>
      portalNotificationsApi.update(id, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: userNotificationKeys.all })
    },
  })
}

export function useDeleteUserNotification() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => portalNotificationsApi.delete(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: userNotificationKeys.all })
    },
  })
}

export function useTestUserNotification() {
  return useMutation({
    mutationFn: (id: number) => portalNotificationsApi.test(id),
  })
}

export function useUserNotificationSchema() {
  return useQuery({
    queryKey: userNotificationKeys.schema(),
    queryFn: () => portalNotificationsApi.getSchema(),
    staleTime: 24 * 60 * 60 * 1000, // 24 hours
  })
}
