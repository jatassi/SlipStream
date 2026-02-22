import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { apiFetch } from '@/api/client'
import type {
  ImportSettings,
  ManualImportRequest,
  ManualImportResponse,
  ParseFilenameResponse,
  PatternPreviewRequest,
  PatternPreviewResponse,
  PendingImport,
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
