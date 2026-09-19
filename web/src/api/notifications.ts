import type {
  CreateNotificationInput,
  Notification,
  NotificationEventGroup,
  NotificationTestResult,
  NotifierSchema,
  UpdateNotificationInput,
} from '@/types'

import { apiFetch } from './client'

// The event catalog is the one endpoint that serialises Go field names verbatim,
// so it arrives PascalCase and is mapped to the domain type here.
type WireEvent = { ID: string; Label: string; Description: string }
type WireEventGroup = { ID: string; Label: string; Events: WireEvent[] }

function toEventGroup(group: WireEventGroup): NotificationEventGroup {
  return {
    id: group.ID,
    label: group.Label,
    events: group.Events.map((event) => ({
      id: event.ID,
      label: event.Label,
      description: event.Description,
    })),
  }
}

export const notificationsApi = {
  list: () => apiFetch<Notification[]>('/notifications'),

  get: (id: number) => apiFetch<Notification>(`/notifications/${id}`),

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

  delete: (id: number) => apiFetch<undefined>(`/notifications/${id}`, { method: 'DELETE' }),

  test: (id: number) =>
    apiFetch<NotificationTestResult>(`/notifications/${id}/test`, { method: 'POST' }),

  testNew: (data: CreateNotificationInput) =>
    apiFetch<NotificationTestResult>('/notifications/test', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  getSchemas: () => apiFetch<NotifierSchema[]>('/notifications/schema'),

  getEventCatalog: async (): Promise<NotificationEventGroup[]> => {
    const groups = await apiFetch<WireEventGroup[]>('/notifications/events')
    return groups.map((group) => toEventGroup(group))
  },
}
