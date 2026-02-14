import type { HistoryResponse, ListHistoryOptions } from '@/types'

import { apiFetch, buildQueryString } from './client'

export type HistoryRetentionSettings = {
  enabled: boolean
  retentionDays: number
}

export const historyApi = {
  list: (options?: ListHistoryOptions) =>
    apiFetch<HistoryResponse>(`/history${buildQueryString(options || {})}`),

  clear: () => apiFetch<void>('/history', { method: 'DELETE' }),

  getSettings: () => apiFetch<HistoryRetentionSettings>('/history/settings'),

  updateSettings: (settings: HistoryRetentionSettings) =>
    apiFetch<HistoryRetentionSettings>('/history/settings', {
      method: 'PUT',
      body: JSON.stringify(settings),
    }),
}
