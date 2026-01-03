import { apiFetch, buildQueryString } from './client'
import type {
  Indexer,
  CreateIndexerInput,
  UpdateIndexerInput,
  IndexerTestResult,
  TestConfigInput,
  DefinitionMetadata,
  Definition,
  DefinitionSetting,
  DefinitionFilters,
  IndexerStatus,
} from '@/types'

export const indexersApi = {
  // Indexer CRUD operations
  list: () =>
    apiFetch<Indexer[]>('/indexers'),

  get: (id: number) =>
    apiFetch<Indexer>(`/indexers/${id}`),

  create: (data: CreateIndexerInput) =>
    apiFetch<Indexer>('/indexers', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  update: (id: number, data: UpdateIndexerInput) =>
    apiFetch<Indexer>(`/indexers/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  delete: (id: number) =>
    apiFetch<void>(`/indexers/${id}`, { method: 'DELETE' }),

  // Test operations
  test: (id: number) =>
    apiFetch<IndexerTestResult>(`/indexers/${id}/test`, { method: 'POST' }),

  testConfig: (data: TestConfigInput) =>
    apiFetch<IndexerTestResult>('/indexers/test', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Status operations
  getStatus: (id: number) =>
    apiFetch<IndexerStatus>(`/indexers/${id}/status`),

  getAllStatuses: () =>
    apiFetch<{ indexers: IndexerStatus[]; stats?: Record<string, number> }>('/indexers/status'),

  // Definition operations
  listDefinitions: () =>
    apiFetch<DefinitionMetadata[]>('/indexers/definitions'),

  searchDefinitions: (query?: string, filters?: DefinitionFilters) => {
    const params: Record<string, string> = {}
    if (query) params.q = query
    if (filters?.protocol) params.protocol = filters.protocol
    if (filters?.privacy) params.privacy = filters.privacy
    if (filters?.language) params.language = filters.language
    const queryString = buildQueryString(params)
    return apiFetch<DefinitionMetadata[]>(`/indexers/definitions/search${queryString}`)
  },

  getDefinition: (id: string) =>
    apiFetch<Definition>(`/indexers/definitions/${encodeURIComponent(id)}`),

  getDefinitionSchema: (id: string) =>
    apiFetch<DefinitionSetting[]>(`/indexers/definitions/${encodeURIComponent(id)}/schema`),

  updateDefinitions: () =>
    apiFetch<{ message: string }>('/indexers/definitions/update', { method: 'POST' }),
}
