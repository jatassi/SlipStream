import type {
  ConnectionConfig,
  DetectDBResponse,
  ImportMappings,
  ImportPreview,
  SourceQualityProfile,
  SourceRootFolder,
  SourceType,
} from '@/types/arr-import'

import { apiFetch } from './client'

export const arrImportApi = {
  detectDB: (sourceType: SourceType) =>
    apiFetch<DetectDBResponse>(`/arrimport/detect-db?sourceType=${sourceType}`),
  connect: (config: ConnectionConfig) =>
    apiFetch<undefined>('/arrimport/connect', {
      method: 'POST',
      body: JSON.stringify(config),
    }),
  getSourceRootFolders: () =>
    apiFetch<SourceRootFolder[]>('/arrimport/source/rootfolders'),
  getSourceQualityProfiles: () =>
    apiFetch<SourceQualityProfile[]>('/arrimport/source/qualityprofiles'),
  preview: (mappings: ImportMappings) =>
    apiFetch<ImportPreview>('/arrimport/preview', {
      method: 'POST',
      body: JSON.stringify(mappings),
    }),
  execute: (mappings: ImportMappings) =>
    apiFetch<undefined>('/arrimport/execute', {
      method: 'POST',
      body: JSON.stringify(mappings),
    }),
  disconnect: () =>
    apiFetch<undefined>('/arrimport/session', { method: 'DELETE' }),
}
