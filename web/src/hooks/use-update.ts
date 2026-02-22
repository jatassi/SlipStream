import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { updateApi } from '@/api'

const updateKeys = {
  all: ['update'] as const,
  status: () => [...updateKeys.all, 'status'] as const,
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

export function useCheckForUpdate() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => updateApi.checkForUpdate(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: updateKeys.status() })
    },
  })
}

export function useInstallUpdate() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => updateApi.install(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: updateKeys.status() })
    },
  })
}

