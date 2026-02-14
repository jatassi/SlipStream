import { portalFetch } from './client'

export type PortalNotification = {
  id: number
  requestId?: number
  type: 'approved' | 'denied' | 'available'
  title: string
  message: string
  read: boolean
  createdAt: string
}

export type PortalNotificationListResponse = {
  notifications: PortalNotification[]
  unreadCount: number
}

export type UnreadCountResponse = {
  count: number
}

export const portalInboxApi = {
  list: (limit = 50, offset = 0) =>
    portalFetch<PortalNotificationListResponse>(`/inbox?limit=${limit}&offset=${offset}`),

  unreadCount: () => portalFetch<UnreadCountResponse>('/inbox/count'),

  markRead: (id: number) => portalFetch<undefined>(`/inbox/${id}/read`, { method: 'POST' }),

  markAllRead: () => portalFetch<undefined>('/inbox/read', { method: 'POST' }),
}
