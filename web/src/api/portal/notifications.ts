import type { CreateUserNotificationInput, NotifierSchema, UserNotification } from '@/types'

import { portalFetch } from './client'

export const portalNotificationsApi = {
  list: () => portalFetch<UserNotification[]>('/notifications'),

  get: (id: number) => portalFetch<UserNotification>(`/notifications/${id}`),

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

  delete: (id: number) => portalFetch<undefined>(`/notifications/${id}`, { method: 'DELETE' }),

  test: (id: number) => portalFetch<undefined>(`/notifications/${id}/test`, { method: 'POST' }),

  getSchema: () => portalFetch<NotifierSchema[]>('/notifications/schema'),
}
