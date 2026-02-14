import { apiFetch } from './client'

export type DefaultEntry = {
  key: string
  entityType: string
  mediaType: string
  entityId: number
}

export type EntityType = 'root_folder' | 'quality_profile' | 'download_client' | 'indexer'
export type MediaType = 'movie' | 'tv'

export const defaultsApi = {
  getAll: () => apiFetch<DefaultEntry[]>('/defaults'),

  getByEntityType: (entityType: EntityType) => apiFetch<DefaultEntry[]>(`/defaults/${entityType}`),

  get: (entityType: EntityType, mediaType: MediaType) =>
    apiFetch<{ exists: boolean; defaultEntry?: DefaultEntry }>(
      `/defaults/${entityType}/${mediaType}`,
    ),

  set: (entityType: EntityType, mediaType: MediaType, entityId: number) =>
    apiFetch<{ message: string }>(`/defaults/${entityType}/${mediaType}`, {
      method: 'POST',
      body: JSON.stringify({ entityId }),
    }),

  clear: (entityType: EntityType, mediaType: MediaType) =>
    apiFetch<{ message: string }>(`/defaults/${entityType}/${mediaType}`, { method: 'DELETE' }),
}
