import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import type {
  ImportSettings,
  UpdateImportSettingsRequest,
  PatternPreviewRequest,
  PatternPreviewResponse,
  PatternValidateResponse,
  ImportStatus,
  PendingImport,
  ManualImportRequest,
  ManualImportResponse,
  PreviewImportResponse,
  ScanDirectoryResponse,
  RenamePreviewResponse,
  ExecuteRenameRequest,
  ExecuteRenameResponse,
  ParseFilenameResponse,
} from '@/types'

const API_BASE = '/api/v1'

// Settings hooks
export function useImportSettings() {
  return useQuery<ImportSettings>({
    queryKey: ['importSettings'],
    queryFn: async () => {
      const res = await fetch(`${API_BASE}/settings/import`)
      if (!res.ok) throw new Error('Failed to fetch import settings')
      return res.json()
    },
  })
}

export function useUpdateImportSettings() {
  const queryClient = useQueryClient()

  return useMutation<ImportSettings, Error, UpdateImportSettingsRequest>({
    mutationFn: async (settings) => {
      const res = await fetch(`${API_BASE}/settings/import`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(settings),
      })
      if (!res.ok) throw new Error('Failed to update import settings')
      return res.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['importSettings'] })
    },
  })
}

// Pattern preview hooks
export function usePreviewNamingPattern() {
  return useMutation<PatternPreviewResponse, Error, PatternPreviewRequest>({
    mutationFn: async (req) => {
      const res = await fetch(`${API_BASE}/settings/import/naming/preview`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      })
      if (!res.ok) throw new Error('Failed to preview pattern')
      return res.json()
    },
  })
}

export function useValidateNamingPattern() {
  return useMutation<PatternValidateResponse, Error, { pattern: string }>({
    mutationFn: async (req) => {
      const res = await fetch(`${API_BASE}/settings/import/naming/validate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      })
      if (!res.ok) throw new Error('Failed to validate pattern')
      return res.json()
    },
  })
}

// Import status hooks
export function useImportStatus() {
  return useQuery<ImportStatus>({
    queryKey: ['importStatus'],
    queryFn: async () => {
      const res = await fetch(`${API_BASE}/import/status`)
      if (!res.ok) throw new Error('Failed to fetch import status')
      return res.json()
    },
    refetchInterval: 5000,
  })
}

export function usePendingImports() {
  return useQuery<PendingImport[]>({
    queryKey: ['pendingImports'],
    queryFn: async () => {
      const res = await fetch(`${API_BASE}/import/pending`)
      if (!res.ok) throw new Error('Failed to fetch pending imports')
      return res.json()
    },
    refetchInterval: 5000,
  })
}

// Manual import hooks
export function useManualImport() {
  const queryClient = useQueryClient()

  return useMutation<ManualImportResponse, Error, ManualImportRequest>({
    mutationFn: async (req) => {
      const res = await fetch(`${API_BASE}/import/manual`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      })
      if (!res.ok) throw new Error('Failed to execute manual import')
      return res.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pendingImports'] })
      queryClient.invalidateQueries({ queryKey: ['importStatus'] })
    },
  })
}

export function usePreviewManualImport() {
  return useMutation<PreviewImportResponse, Error, { path: string }>({
    mutationFn: async (req) => {
      const res = await fetch(`${API_BASE}/import/manual/preview`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      })
      if (!res.ok) throw new Error('Failed to preview import')
      return res.json()
    },
  })
}

// Retry import
export function useRetryImport() {
  const queryClient = useQueryClient()

  return useMutation<{ success: boolean; message: string }, Error, number>({
    mutationFn: async (id) => {
      const res = await fetch(`${API_BASE}/import/${id}/retry`, {
        method: 'POST',
      })
      if (!res.ok) throw new Error('Failed to retry import')
      return res.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pendingImports'] })
      queryClient.invalidateQueries({ queryKey: ['importStatus'] })
    },
  })
}

// Scan directory
export function useScanDirectory() {
  return useMutation<ScanDirectoryResponse, Error, { path: string }>({
    mutationFn: async (req) => {
      const res = await fetch(`${API_BASE}/import/scan`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      })
      if (!res.ok) throw new Error('Failed to scan directory')
      return res.json()
    },
  })
}

// Rename preview hooks
export function useRenamePreview(mediaType: 'series' | 'movie', mediaId?: number, needsRename?: boolean) {
  const params = new URLSearchParams({ type: mediaType })
  if (mediaId) params.set('mediaId', mediaId.toString())
  if (needsRename) params.set('needsRename', 'true')

  return useQuery<RenamePreviewResponse>({
    queryKey: ['renamePreview', mediaType, mediaId, needsRename],
    queryFn: async () => {
      const res = await fetch(`${API_BASE}/import/rename/preview?${params}`)
      if (!res.ok) throw new Error('Failed to fetch rename preview')
      return res.json()
    },
    enabled: false,
  })
}

export function useExecuteRename() {
  const queryClient = useQueryClient()

  return useMutation<ExecuteRenameResponse, Error, ExecuteRenameRequest>({
    mutationFn: async (req) => {
      const res = await fetch(`${API_BASE}/import/rename/execute`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      })
      if (!res.ok) throw new Error('Failed to execute rename')
      return res.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['renamePreview'] })
      queryClient.invalidateQueries({ queryKey: ['movies'] })
      queryClient.invalidateQueries({ queryKey: ['series'] })
    },
  })
}

// Parse filename hook
export function useParseFilename() {
  return useMutation<ParseFilenameResponse, Error, { filename: string }>({
    mutationFn: async (req) => {
      const res = await fetch(`${API_BASE}/settings/import/naming/parse`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req),
      })
      if (!res.ok) throw new Error('Failed to parse filename')
      return res.json()
    },
  })
}
