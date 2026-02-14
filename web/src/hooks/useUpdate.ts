import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { updateApi } from '@/api'

export const updateKeys = {
  all: ['update'] as const,
  status: () => [...updateKeys.all, 'status'] as const,
  settings: () => [...updateKeys.all, 'settings'] as const,
}

export function useUpdateStatus() {
  return useQuery({
    queryKey: updateKeys.status(),
    queryFn: () => updateApi.getStatus(),
    refetchInterval: (query) => {
      const state = query.state.data?.state
      if (state === 'downloading' || state === 'installing') {
        return 1000
      }
      return 60_000
    },
  })
}

export function useAutoInstallSettings() {
  return useQuery({
    queryKey: updateKeys.settings(),
    queryFn: () => updateApi.getSettings(),
  })
}

export function useCheckForUpdate() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => updateApi.checkForUpdate(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: updateKeys.status() })
    },
  })
}

export function useInstallUpdate() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => updateApi.install(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: updateKeys.status() })
    },
  })
}

export function useCancelUpdate() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => updateApi.cancel(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: updateKeys.status() })
    },
  })
}

export function useUpdateAutoInstall() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (autoInstall: boolean) => updateApi.updateSettings({ autoInstall }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: updateKeys.settings() })
    },
  })
}
