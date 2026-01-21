import { portalFetch } from './client'
import type { UserNotification, CreateUserNotificationInput } from '@/types'
import type { NotifierSchema } from '@/types'

export const portalNotificationsApi = {
  list: () =>
    portalFetch<UserNotification[]>('/notifications'),

  get: (id: number) =>
    portalFetch<UserNotification>(`/notifications/${id}`),

  create: (data: CreateUserNotificationInput) =>
    portalFetch<UserNotification>('/notifications', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  update: (id: number, data: CreateUserNotificationInput) =>
    portalFetch<UserNotification>(`/notifications/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  delete: (id: number) =>
    portalFetch<void>(`/notifications/${id}`, { method: 'DELETE' }),

  test: (id: number) =>
    portalFetch<void>(`/notifications/${id}/test`, { method: 'POST' }),

  getSchema: () =>
    portalFetch<NotifierSchema[]>('/notifications/schema'),
}
