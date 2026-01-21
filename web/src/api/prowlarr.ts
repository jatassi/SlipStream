import { apiFetch } from './client'
import type {
  ProwlarrConfig,
  ProwlarrConfigInput,
  ProwlarrTestInput,
  ProwlarrTestResult,
  ProwlarrIndexer,
  ProwlarrCapabilities,
  ProwlarrConnectionStatus,
  ModeInfo,
  SetModeInput,
  RefreshResult,
  ProwlarrIndexerWithSettings,
  ProwlarrIndexerSettings,
  ProwlarrIndexerSettingsInput,
} from '@/types'

export const prowlarrApi = {
  // Configuration operations
  getConfig: () =>
    apiFetch<ProwlarrConfig>('/indexers/prowlarr'),

  updateConfig: (data: ProwlarrConfigInput) =>
    apiFetch<ProwlarrConfig>('/indexers/prowlarr', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  // Connection testing
  testConnection: (data: ProwlarrTestInput) =>
    apiFetch<ProwlarrTestResult>('/indexers/prowlarr/test', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Indexer operations (read-only from Prowlarr)
  getIndexers: () =>
    apiFetch<ProwlarrIndexer[]>('/indexers/prowlarr/indexers'),

  // Capabilities
  getCapabilities: () =>
    apiFetch<ProwlarrCapabilities>('/indexers/prowlarr/capabilities'),

  // Connection status
  getStatus: () =>
    apiFetch<ProwlarrConnectionStatus>('/indexers/prowlarr/status'),

  // Refresh cached data
  refresh: () =>
    apiFetch<RefreshResult>('/indexers/prowlarr/refresh', {
      method: 'POST',
    }),

  // Mode operations
  getMode: () =>
    apiFetch<ModeInfo>('/indexers/mode'),

  setMode: (data: SetModeInput) =>
    apiFetch<ModeInfo>('/indexers/mode', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  // Per-indexer settings operations
  getIndexersWithSettings: () =>
    apiFetch<ProwlarrIndexerWithSettings[]>('/indexers/prowlarr/indexers/settings'),

  getIndexerSettings: (indexerId: number) =>
    apiFetch<ProwlarrIndexerSettings>(`/indexers/prowlarr/indexers/${indexerId}/settings`),

  updateIndexerSettings: (indexerId: number, data: ProwlarrIndexerSettingsInput) =>
    apiFetch<ProwlarrIndexerSettings>(`/indexers/prowlarr/indexers/${indexerId}/settings`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  deleteIndexerSettings: (indexerId: number) =>
    apiFetch<void>(`/indexers/prowlarr/indexers/${indexerId}/settings`, {
      method: 'DELETE',
    }),

  resetIndexerStats: (indexerId: number) =>
    apiFetch<ProwlarrIndexerSettings>(`/indexers/prowlarr/indexers/${indexerId}/reset-stats`, {
      method: 'POST',
    }),
}
