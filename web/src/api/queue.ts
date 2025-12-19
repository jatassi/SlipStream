import { apiFetch } from './client'
import type { QueueItem, QueueStats } from '@/types'

export const queueApi = {
  list: () =>
    apiFetch<QueueItem[]>('/queue'),

  get: (id: number) =>
    apiFetch<QueueItem>(`/queue/${id}`),

  remove: (id: number) =>
    apiFetch<void>(`/queue/${id}`, { method: 'DELETE' }),

  pause: (id: number) =>
    apiFetch<QueueItem>(`/queue/${id}/pause`, { method: 'POST' }),

  resume: (id: number) =>
    apiFetch<QueueItem>(`/queue/${id}/resume`, { method: 'POST' }),

  stats: () =>
    apiFetch<QueueStats>('/queue/stats'),
}
