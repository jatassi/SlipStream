import { apiFetch } from './client'
import type { QueueItem, QueueStats } from '@/types'

export const queueApi = {
  list: () =>
    apiFetch<QueueItem[]>('/queue'),

  pause: (clientId: number, id: string) =>
    apiFetch<{ status: string }>(`/queue/${id}/pause`, {
      method: 'POST',
      body: JSON.stringify({ clientId }),
    }),

  resume: (clientId: number, id: string) =>
    apiFetch<{ status: string }>(`/queue/${id}/resume`, {
      method: 'POST',
      body: JSON.stringify({ clientId }),
    }),

  fastForward: (clientId: number, id: string) =>
    apiFetch<{ status: string }>(`/queue/${id}/fastforward`, {
      method: 'POST',
      body: JSON.stringify({ clientId }),
    }),

  remove: (clientId: number, id: string, deleteFiles = false) =>
    apiFetch<void>(`/queue/${id}?clientId=${clientId}&deleteFiles=${deleteFiles}`, {
      method: 'DELETE',
    }),

  stats: () =>
    apiFetch<QueueStats>('/queue/stats'),
}
