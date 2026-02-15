import type { CreateRequestInput, PortalDownload, Request, RequestListFilters } from '@/types'

import { buildQueryString, portalFetch } from './client'

export const portalRequestsApi = {
  list: (filters?: RequestListFilters) => portalFetch<Request[]>(buildQueryString(filters ?? {})),

  get: (id: number) => portalFetch<Request>(`/${id}`),

  create: (data: CreateRequestInput) =>
    portalFetch<Request>('', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  cancel: (id: number) => portalFetch<undefined>(`/${id}`, { method: 'DELETE' }),

  watch: (id: number) => portalFetch<undefined>(`/${id}/watch`, { method: 'POST' }),

  unwatch: (id: number) => portalFetch<undefined>(`/${id}/watch`, { method: 'DELETE' }),

  getWatchers: (id: number) => portalFetch<number[]>(`/${id}/watchers`),

  downloads: () => portalFetch<PortalDownload[]>('/downloads'),
}
