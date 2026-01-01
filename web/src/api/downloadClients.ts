import { apiFetch } from './client'
import type {
  DownloadClient,
  CreateDownloadClientInput,
  UpdateDownloadClientInput,
  DownloadClientTestResult,
} from '@/types'

export const downloadClientsApi = {
  list: () =>
    apiFetch<DownloadClient[]>('/downloadclients'),

  get: (id: number) =>
    apiFetch<DownloadClient>(`/downloadclients/${id}`),

  create: (data: CreateDownloadClientInput) =>
    apiFetch<DownloadClient>('/downloadclients', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  update: (id: number, data: UpdateDownloadClientInput) =>
    apiFetch<DownloadClient>(`/downloadclients/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  delete: (id: number) =>
    apiFetch<void>(`/downloadclients/${id}`, { method: 'DELETE' }),

  test: (id: number) =>
    apiFetch<DownloadClientTestResult>(`/downloadclients/${id}/test`, { method: 'POST' }),

  testNew: (data: CreateDownloadClientInput) =>
    apiFetch<DownloadClientTestResult>('/downloadclients/test', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  debugAddTorrent: (id: number) =>
    apiFetch<{ success: boolean; torrentId: string; message: string }>(
      `/downloadclients/${id}/debug/addtorrent`,
      { method: 'POST' }
    ),
}
