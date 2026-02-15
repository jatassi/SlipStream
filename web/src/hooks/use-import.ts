import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { apiFetch } from '@/api/client'
import type {
  ExecuteRenameRequest,
  ExecuteRenameResponse,
  ImportSettings,
  ImportStatus,
  ManualImportRequest,
  ManualImportResponse,
  ParseFilenameResponse,
  PatternPreviewRequest,
  PatternPreviewResponse,
  PatternValidateResponse,
  PendingImport,
  PreviewImportResponse,
  RenamePreviewResponse,
  ScanDirectoryResponse,
  UpdateImportSettingsRequest,
} from '@/types'

// Settings hooks
export function useImportSettings() {
  return useQuery<ImportSettings>({
    queryKey: ['importSettings'],
    queryFn: () => apiFetch<ImportSettings>('/settings/import'),
  })
}

export function useUpdateImportSettings() {
  const queryClient = useQueryClient()

  return useMutation<ImportSettings, Error, UpdateImportSettingsRequest>({
    mutationFn: (settings) =>
      apiFetch<ImportSettings>('/settings/import', {
        method: 'PUT',
        body: JSON.stringify(settings),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['importSettings'] })
    },
  })
}

// Pattern preview hooks
export function usePreviewNamingPattern() {
  return useMutation<PatternPreviewResponse, Error, PatternPreviewRequest>({
    mutationFn: (req) =>
      apiFetch<PatternPreviewResponse>('/settings/import/naming/preview', {
        method: 'POST',
        body: JSON.stringify(req),
      }),
  })
}

export function useValidateNamingPattern() {
  return useMutation<PatternValidateResponse, Error, { pattern: string }>({
    mutationFn: (req) =>
      apiFetch<PatternValidateResponse>('/settings/import/naming/validate', {
        method: 'POST',
        body: JSON.stringify(req),
      }),
  })
}

// Import status hooks
export function useImportStatus() {
  return useQuery<ImportStatus>({
    queryKey: ['importStatus'],
    queryFn: () => apiFetch<ImportStatus>('/import/status'),
    refetchInterval: 5000,
  })
}

export function usePendingImports() {
  return useQuery<PendingImport[]>({
    queryKey: ['pendingImports'],
    queryFn: () => apiFetch<PendingImport[]>('/import/pending'),
    refetchInterval: 5000,
  })
}

// Manual import hooks
export function useManualImport() {
  const queryClient = useQueryClient()

  return useMutation<ManualImportResponse, Error, ManualImportRequest>({
    mutationFn: (req) =>
      apiFetch<ManualImportResponse>('/import/manual', {
        method: 'POST',
        body: JSON.stringify(req),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['pendingImports'] })
      void queryClient.invalidateQueries({ queryKey: ['importStatus'] })
    },
  })
}

export function usePreviewManualImport() {
  return useMutation<PreviewImportResponse, Error, { path: string }>({
    mutationFn: (req) =>
      apiFetch<PreviewImportResponse>('/import/manual/preview', {
        method: 'POST',
        body: JSON.stringify(req),
      }),
  })
}

// Retry import
export function useRetryImport() {
  const queryClient = useQueryClient()

  return useMutation<{ success: boolean; message: string }, Error, number>({
    mutationFn: (id) =>
      apiFetch<{ success: boolean; message: string }>(`/import/${id}/retry`, {
        method: 'POST',
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['pendingImports'] })
      void queryClient.invalidateQueries({ queryKey: ['importStatus'] })
    },
  })
}

// Scan directory
export function useScanDirectory() {
  return useMutation<ScanDirectoryResponse, Error, { path: string }>({
    mutationFn: (req) =>
      apiFetch<ScanDirectoryResponse>('/import/scan', {
        method: 'POST',
        body: JSON.stringify(req),
      }),
  })
}

// Rename preview hooks
export function useRenamePreview(
  mediaType: 'series' | 'movie',
  mediaId?: number,
  needsRename?: boolean,
) {
  const params = new URLSearchParams({ type: mediaType })
  if (mediaId) {
    params.set('mediaId', mediaId.toString())
  }
  if (needsRename) {
    params.set('needsRename', 'true')
  }

  return useQuery<RenamePreviewResponse>({
    queryKey: ['renamePreview', mediaType, mediaId, needsRename],
    queryFn: () => apiFetch<RenamePreviewResponse>(`/import/rename/preview?${params}`),
    enabled: false,
  })
}

export function useExecuteRename() {
  const queryClient = useQueryClient()

  return useMutation<ExecuteRenameResponse, Error, ExecuteRenameRequest>({
    mutationFn: (req) =>
      apiFetch<ExecuteRenameResponse>('/import/rename/execute', {
        method: 'POST',
        body: JSON.stringify(req),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['renamePreview'] })
      void queryClient.invalidateQueries({ queryKey: ['movies'] })
      void queryClient.invalidateQueries({ queryKey: ['series'] })
    },
  })
}

// Parse filename hook
export function useParseFilename() {
  return useMutation<ParseFilenameResponse, Error, { filename: string }>({
    mutationFn: (req) =>
      apiFetch<ParseFilenameResponse>('/settings/import/naming/parse', {
        method: 'POST',
        body: JSON.stringify(req),
      }),
  })
}
