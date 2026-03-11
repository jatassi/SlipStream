import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { systemApi } from '@/api'
import { setModuleEnabledState } from '@/modules'
import type { UpdateSettingsInput } from '@/types'

export const systemKeys = {
  all: ['system'] as const,
  status: () => [...systemKeys.all, 'status'] as const,
  settings: () => [...systemKeys.all, 'settings'] as const,
  firewall: () => [...systemKeys.all, 'firewall'] as const,
}

export function useStatus() {
  return useQuery({
    queryKey: systemKeys.status(),
    queryFn: async () => {
      const status = await systemApi.status()
      if (status.enabledModules) {
        setModuleEnabledState(status.enabledModules)
      }
      return status
    },
    refetchInterval: 30_000, // Refresh every 30 seconds
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
      void queryClient.invalidateQueries({ queryKey: systemKeys.settings() })
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

export function useMediainfoAvailable() {
  const { data } = useStatus()
  return data?.mediainfoAvailable ?? false
}

export function useRestart() {
  return useMutation({
    mutationFn: () => systemApi.restart(),
  })
}

export function useFirewallStatus() {
  return useQuery({
    queryKey: systemKeys.firewall(),
    queryFn: () => systemApi.checkFirewall(),
    staleTime: 60_000, // Consider fresh for 1 minute
  })
}

export function useCheckFirewall() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => systemApi.checkFirewall(),
    onSuccess: (data) => {
      queryClient.setQueryData(systemKeys.firewall(), data)
    },
  })
}

export function useUpdateModuleEnabled() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: Record<string, boolean>) => systemApi.updateModuleEnabled(data),
    onSuccess: (enabledMap) => {
      setModuleEnabledState(enabledMap)
      void queryClient.invalidateQueries({ queryKey: systemKeys.status() })
    },
  })
}
