import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { rssSyncApi } from '@/api'
import type { RssSyncSettings } from '@/types'

export const rssSyncKeys = {
  all: ['rsssync'] as const,
  settings: () => [...rssSyncKeys.all, 'settings'] as const,
  status: () => [...rssSyncKeys.all, 'status'] as const,
}

export function useRssSyncSettings() {
  return useQuery({
    queryKey: rssSyncKeys.settings(),
    queryFn: () => rssSyncApi.getSettings(),
  })
}

export function useUpdateRssSyncSettings() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (settings: RssSyncSettings) => rssSyncApi.updateSettings(settings),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: rssSyncKeys.settings() })
    },
  })
}

export function useRssSyncStatus() {
  return useQuery({
    queryKey: rssSyncKeys.status(),
    queryFn: () => rssSyncApi.getStatus(),
    staleTime: 5000,
  })
}

export function useTriggerRssSync() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => rssSyncApi.trigger(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: rssSyncKeys.status() })
    },
  })
}
