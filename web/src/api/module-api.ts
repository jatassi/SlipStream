import type { ModuleApi } from '@/modules/types'

import { apiFetch, buildQueryString } from './client'

export function createModuleApi(basePath: string): ModuleApi {
  return {
    list: (options) =>
      apiFetch<unknown[]>(`${basePath}${buildQueryString(options ?? {})}`),

    get: (id) => apiFetch<unknown>(`${basePath}/${id}`),

    update: (id, data) =>
      apiFetch<unknown>(`${basePath}/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),

    delete: (id, deleteFiles) =>
      apiFetch<undefined>(
        `${basePath}/${id}${deleteFiles ? '?deleteFiles=true' : ''}`,
        { method: 'DELETE' },
      ),

    bulkDelete: (ids, deleteFiles) =>
      Promise.all(
        ids.map((id) =>
          apiFetch<undefined>(
            `${basePath}/${id}${deleteFiles ? '?deleteFiles=true' : ''}`,
            { method: 'DELETE' },
          ),
        ),
      ).then(() => undefined),

    bulkUpdate: (ids, data) =>
      Promise.all(
        ids.map((id) =>
          apiFetch<unknown>(`${basePath}/${id}`, {
            method: 'PUT',
            body: JSON.stringify(data),
          }),
        ),
      ),

    bulkMonitor: (ids, monitored) =>
      apiFetch<unknown>(`${basePath}/monitor`, {
        method: 'PUT',
        body: JSON.stringify({ ids, monitored }),
      }),

    search: (id) =>
      apiFetch<undefined>(`${basePath}/${id}/search`, { method: 'POST' }),

    refresh: (id) =>
      apiFetch<unknown>(`${basePath}/${id}/refresh`, { method: 'POST' }),

    refreshAll: () =>
      apiFetch<unknown>(`${basePath}/refresh`, { method: 'POST' }),
  }
}
