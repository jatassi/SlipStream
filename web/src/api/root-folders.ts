import type { CreateRootFolderInput, RootFolder } from '@/types'

import { apiFetch } from './client'

export const rootFoldersApi = {
  list: () => apiFetch<RootFolder[]>('/rootfolders'),

  listByType: (mediaType: 'movie' | 'tv') =>
    apiFetch<RootFolder[]>(`/rootfolders?mediaType=${mediaType}`),

  get: (id: number) => apiFetch<RootFolder>(`/rootfolders/${id}`),

  create: (data: CreateRootFolderInput) =>
    apiFetch<RootFolder>('/rootfolders', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  delete: (id: number) => apiFetch<undefined>(`/rootfolders/${id}`, { method: 'DELETE' }),

  refresh: (id: number) => apiFetch<RootFolder>(`/rootfolders/${id}/refresh`, { method: 'POST' }),
}
