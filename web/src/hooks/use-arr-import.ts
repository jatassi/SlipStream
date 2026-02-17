import { useMutation,useQuery  } from '@tanstack/react-query'

import { arrImportApi } from '@/api/arr-import'
import { createQueryKeys } from '@/lib/query-keys'
import { useProgressStore } from '@/stores/progress'
import type { ConnectionConfig, ImportMappings, SourceType } from '@/types/arr-import'

export type WizardStep = 'connect' | 'mapping' | 'preview' | 'importing' | 'report'

const baseKeys = createQueryKeys('arrimport')
export const arrImportKeys = {
  ...baseKeys,
  detectDB: (sourceType: SourceType) => [...baseKeys.all, 'detect-db', sourceType] as const,
  sourceRootFolders: () => [...baseKeys.all, 'source', 'rootfolders'] as const,
  sourceQualityProfiles: () => [...baseKeys.all, 'source', 'qualityprofiles'] as const,
}

export function useDetectDB(sourceType: SourceType, enabled: boolean) {
  return useQuery({
    queryKey: arrImportKeys.detectDB(sourceType),
    queryFn: () => arrImportApi.detectDB(sourceType),
    enabled,
    staleTime: 60_000,
  })
}

export function useSourceRootFolders() {
  return useQuery({
    queryKey: arrImportKeys.sourceRootFolders(),
    queryFn: () => arrImportApi.getSourceRootFolders(),
    enabled: false,
  })
}

export function useSourceQualityProfiles() {
  return useQuery({
    queryKey: arrImportKeys.sourceQualityProfiles(),
    queryFn: () => arrImportApi.getSourceQualityProfiles(),
    enabled: false,
  })
}

export function useConnect() {
  return useMutation({
    mutationFn: (config: ConnectionConfig) => arrImportApi.connect(config),
  })
}

export function usePreview() {
  return useMutation({
    mutationFn: (mappings: ImportMappings) => arrImportApi.preview(mappings),
  })
}

export function useExecuteImport() {
  return useMutation({
    mutationFn: (mappings: ImportMappings) => arrImportApi.execute(mappings),
  })
}

export function useDisconnect() {
  return useMutation({
    mutationFn: () => arrImportApi.disconnect(),
  })
}

export function useImportProgress() {
  const activities = useProgressStore((s) => s.activities)
  return activities.find((a) => a.id === 'arrimport')
}
