import { apiFetch } from './client'
import type {
  Notification,
  CreateNotificationInput,
  UpdateNotificationInput,
  NotificationTestResult,
  NotifierSchema,
} from '@/types'

export const notificationsApi = {
  list: () =>
    apiFetch<Notification[]>('/notifications'),

  get: (id: number) =>
    apiFetch<Notification>(`/notifications/${id}`),

  create: (data: CreateNotificationInput) =>
    apiFetch<Notification>('/notifications', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  update: (id: number, data: UpdateNotificationInput) =>
    apiFetch<Notification>(`/notifications/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  delete: (id: number) =>
    apiFetch<void>(`/notifications/${id}`, { method: 'DELETE' }),

  test: (id: number) =>
    apiFetch<NotificationTestResult>(`/notifications/${id}/test`, { method: 'POST' }),

  testNew: (data: CreateNotificationInput) =>
    apiFetch<NotificationTestResult>('/notifications/test', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  getSchemas: () =>
    apiFetch<NotifierSchema[]>('/notifications/schema'),
}
