import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { systemApi } from '@/api'
import type { UpdateSettingsInput } from '@/types'

export const systemKeys = {
  all: ['system'] as const,
  status: () => [...systemKeys.all, 'status'] as const,
  health: () => [...systemKeys.all, 'health'] as const,
  settings: () => [...systemKeys.all, 'settings'] as const,
}

export function useStatus() {
  return useQuery({
    queryKey: systemKeys.status(),
    queryFn: () => systemApi.status(),
    refetchInterval: 30000, // Refresh every 30 seconds
  })
}

export function useHealth() {
  return useQuery({
    queryKey: systemKeys.health(),
    queryFn: () => systemApi.health(),
    refetchInterval: 60000, // Refresh every minute
  })
}

export function useSettings() {
  return useQuery({
    queryKey: systemKeys.settings(),
    queryFn: () => systemApi.getSettings(),
  })
}

export function useUpdateSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: UpdateSettingsInput) => systemApi.updateSettings(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: systemKeys.settings() })
    },
  })
}

export function useRegenerateApiKey() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => systemApi.regenerateApiKey(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: systemKeys.settings() })
    },
  })
}

export function useDeveloperMode() {
  const { data } = useStatus()
  return data?.developerMode ?? false
}

export function usePortalEnabled() {
  const { data } = useStatus()
  return data?.portalEnabled ?? true
}

export function useTMDBSearchOrdering() {
  const { data } = useStatus()
  return data?.tmdb?.disableSearchOrdering ?? false
}

export function useUpdateTMDBSearchOrdering() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (disableSearchOrdering: boolean) => systemApi.updateTMDBSearchOrdering(disableSearchOrdering),
    onSuccess: (data, variables) => {
      console.log('useUpdateTMDBSearchOrdering - Success:', { data, variables })
      queryClient.invalidateQueries({ queryKey: systemKeys.status() })
    },
    onError: (error) => {
      console.error('useUpdateTMDBSearchOrdering - Error:', error)
    },
    onSettled: () => {
      console.log('useUpdateTMDBSearchOrdering - Settled')
    }
  })
}
