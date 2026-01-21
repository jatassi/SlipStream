import { portalFetch } from './client'

export interface PortalNotification {
  id: number
  requestId?: number
  type: 'approved' | 'denied' | 'available'
  title: string
  message: string
  read: boolean
  createdAt: string
}

export interface PortalNotificationListResponse {
  notifications: PortalNotification[]
  unreadCount: number
}

export interface UnreadCountResponse {
  count: number
}

export const portalInboxApi = {
  list: (limit = 50, offset = 0) =>
    portalFetch<PortalNotificationListResponse>(`/inbox?limit=${limit}&offset=${offset}`),

  unreadCount: () =>
    portalFetch<UnreadCountResponse>('/inbox/count'),

  markRead: (id: number) =>
    portalFetch<void>(`/inbox/${id}/read`, { method: 'POST' }),

  markAllRead: () =>
    portalFetch<void>('/inbox/read', { method: 'POST' }),
}
