import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { apiFetch } from '@/api/client'
import { createQueryKeys } from '@/lib/query-keys'
import type {
  ModuleNamingSettings,
  NamingPreviewRequest,
  NamingPreviewResponse,
  UpdateModuleNamingRequest,
} from '@/types'

const baseKeys = createQueryKeys('settings', 'naming')
export const namingKeys = {
  ...baseKeys,
  module: (moduleId: string) => [...baseKeys.all, moduleId] as const,
}

export function useModuleNamingSettings(moduleId: string) {
  return useQuery<ModuleNamingSettings>({
    queryKey: namingKeys.module(moduleId),
    queryFn: () => apiFetch<ModuleNamingSettings>(`/settings/${moduleId}/naming`),
  })
}

export function useUpdateModuleNamingSettings(moduleId: string) {
  const queryClient = useQueryClient()

  return useMutation<ModuleNamingSettings, Error, UpdateModuleNamingRequest>({
    mutationFn: (data) =>
      apiFetch<ModuleNamingSettings>(`/settings/${moduleId}/naming`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: namingKeys.module(moduleId) })
    },
  })
}

export function useModuleNamingPreview(moduleId: string) {
  return useMutation<NamingPreviewResponse, Error, NamingPreviewRequest>({
    mutationFn: (req) =>
      apiFetch<NamingPreviewResponse>(`/settings/${moduleId}/naming/preview`, {
        method: 'POST',
        body: JSON.stringify(req),
      }),
  })
}
