import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { prowlarrApi } from '@/api'
import type {
  ProwlarrConfigInput,
  ProwlarrIndexerSettingsInput,
  ProwlarrTestInput,
  SetModeInput,
} from '@/types'

export const prowlarrKeys = {
  all: ['prowlarr'] as const,
  config: () => [...prowlarrKeys.all, 'config'] as const,
  indexers: () => [...prowlarrKeys.all, 'indexers'] as const,
  indexersWithSettings: () => [...prowlarrKeys.all, 'indexers', 'settings'] as const,
  indexerSettings: (id: number) => [...prowlarrKeys.all, 'indexers', id, 'settings'] as const,
  capabilities: () => [...prowlarrKeys.all, 'capabilities'] as const,
  status: () => [...prowlarrKeys.all, 'status'] as const,
  mode: () => [...prowlarrKeys.all, 'mode'] as const,
}

// Configuration hooks
export function useProwlarrConfig() {
  return useQuery({
    queryKey: prowlarrKeys.config(),
    queryFn: () => prowlarrApi.getConfig(),
  })
}

export function useUpdateProwlarrConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: ProwlarrConfigInput) => prowlarrApi.updateConfig(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: prowlarrKeys.all })
    },
  })
}

// Connection testing
export function useTestProwlarrConnection() {
  return useMutation({
    mutationFn: (data: ProwlarrTestInput) => prowlarrApi.testConnection(data),
  })
}

// Indexer hooks (read-only from Prowlarr)
export function useProwlarrIndexers() {
  return useQuery({
    queryKey: prowlarrKeys.indexers(),
    queryFn: () => prowlarrApi.getIndexers(),
  })
}

// Capabilities
export function useProwlarrCapabilities() {
  return useQuery({
    queryKey: prowlarrKeys.capabilities(),
    queryFn: () => prowlarrApi.getCapabilities(),
  })
}

// Connection status
export function useProwlarrStatus() {
  return useQuery({
    queryKey: prowlarrKeys.status(),
    queryFn: () => prowlarrApi.getStatus(),
    refetchInterval: 60_000, // Refresh every minute
  })
}

// Refresh cached data
export function useRefreshProwlarr() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => prowlarrApi.refresh(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: prowlarrKeys.indexers() })
      queryClient.invalidateQueries({ queryKey: prowlarrKeys.capabilities() })
      queryClient.invalidateQueries({ queryKey: prowlarrKeys.status() })
    },
  })
}

// Mode hooks
export function useIndexerMode() {
  return useQuery({
    queryKey: prowlarrKeys.mode(),
    queryFn: () => prowlarrApi.getMode(),
  })
}

export function useSetIndexerMode() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: SetModeInput) => prowlarrApi.setMode(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: prowlarrKeys.mode() })
      // Also invalidate indexer-related queries as mode affects what's available
      queryClient.invalidateQueries({ queryKey: ['indexers'] })
    },
  })
}

// Per-indexer settings hooks
export function useProwlarrIndexersWithSettings() {
  return useQuery({
    queryKey: prowlarrKeys.indexersWithSettings(),
    queryFn: () => prowlarrApi.getIndexersWithSettings(),
  })
}

export function useProwlarrIndexerSettings(indexerId: number) {
  return useQuery({
    queryKey: prowlarrKeys.indexerSettings(indexerId),
    queryFn: () => prowlarrApi.getIndexerSettings(indexerId),
    enabled: indexerId > 0,
  })
}

export function useUpdateProwlarrIndexerSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ indexerId, data }: { indexerId: number; data: ProwlarrIndexerSettingsInput }) =>
      prowlarrApi.updateIndexerSettings(indexerId, data),
    onSuccess: (_, { indexerId }) => {
      queryClient.invalidateQueries({ queryKey: prowlarrKeys.indexersWithSettings() })
      queryClient.invalidateQueries({ queryKey: prowlarrKeys.indexerSettings(indexerId) })
    },
  })
}

export function useDeleteProwlarrIndexerSettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (indexerId: number) => prowlarrApi.deleteIndexerSettings(indexerId),
    onSuccess: (_, indexerId) => {
      queryClient.invalidateQueries({ queryKey: prowlarrKeys.indexersWithSettings() })
      queryClient.invalidateQueries({ queryKey: prowlarrKeys.indexerSettings(indexerId) })
    },
  })
}

export function useResetProwlarrIndexerStats() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (indexerId: number) => prowlarrApi.resetIndexerStats(indexerId),
    onSuccess: (_, indexerId) => {
      queryClient.invalidateQueries({ queryKey: prowlarrKeys.indexersWithSettings() })
      queryClient.invalidateQueries({ queryKey: prowlarrKeys.indexerSettings(indexerId) })
    },
  })
}
