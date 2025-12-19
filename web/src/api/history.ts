import { apiFetch, buildQueryString } from './client'
import type { HistoryResponse, ListHistoryOptions } from '@/types'

export const historyApi = {
  list: (options?: ListHistoryOptions) =>
    apiFetch<HistoryResponse>(`/history${buildQueryString(options || {})}`),

  get: (id: number) =>
    apiFetch<HistoryResponse>(`/history/${id}`),

  clear: () =>
    apiFetch<void>('/history', { method: 'DELETE' }),
}
