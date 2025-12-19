import { apiFetch } from './client'
import type {
  Indexer,
  CreateIndexerInput,
  UpdateIndexerInput,
  IndexerTestResult,
} from '@/types'

export const indexersApi = {
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

  test: (id: number) =>
    apiFetch<IndexerTestResult>(`/indexers/${id}/test`, { method: 'POST' }),

  testNew: (data: CreateIndexerInput) =>
    apiFetch<IndexerTestResult>('/indexers/test', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
}
