import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { notificationsApi } from '@/api'
import { createQueryKeys } from '@/lib/query-keys'
import type { CreateNotificationInput, Notification, UpdateNotificationInput } from '@/types'

const baseKeys = createQueryKeys('notifications')
const notificationKeys = {
  ...baseKeys,
  schemas: () => [...baseKeys.all, 'schemas'] as const,
}

export function useNotifications() {
  return useQuery({
    queryKey: notificationKeys.list(),
    queryFn: () => notificationsApi.list(),
  })
}

export function useNotificationSchemas() {
  return useQuery({
    queryKey: notificationKeys.schemas(),
    queryFn: () => notificationsApi.getSchemas(),
    staleTime: Infinity,
  })
}

export function useCreateNotification() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateNotificationInput) => notificationsApi.create(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all })
    },
  })
}

export function useUpdateNotification() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateNotificationInput }) =>
      notificationsApi.update(id, data),
    onSuccess: (notification: Notification) => {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all })
      queryClient.setQueryData(notificationKeys.detail(notification.id), notification)
    },
  })
}

export function useDeleteNotification() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => notificationsApi.delete(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: notificationKeys.all })
    },
  })
}

export function useTestNotification() {
  return useMutation({
    mutationFn: (id: number) => notificationsApi.test(id),
  })
}

export function useTestNewNotification() {
  return useMutation({
    mutationFn: (data: CreateNotificationInput) => notificationsApi.testNew(data),
  })
}
