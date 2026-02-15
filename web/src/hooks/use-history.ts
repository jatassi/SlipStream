import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { historyApi } from '@/api'
import type { HistoryRetentionSettings } from '@/api/history'
import type { ListHistoryOptions } from '@/types'

export const historyKeys = {
  all: ['history'] as const,
  lists: () => [...historyKeys.all, 'list'] as const,
  list: (filters: ListHistoryOptions) => [...historyKeys.lists(), filters] as const,
  settings: () => [...historyKeys.all, 'settings'] as const,
}

export function useHistory(options?: ListHistoryOptions) {
  return useQuery({
    queryKey: historyKeys.list(options ?? {}),
    queryFn: () => historyApi.list(options),
  })
}

export function useClearHistory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => historyApi.clear(),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: historyKeys.all })
    },
  })
}

export function useHistorySettings() {
  return useQuery({
    queryKey: historyKeys.settings(),
    queryFn: () => historyApi.getSettings(),
  })
}

export function useUpdateHistorySettings() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (settings: HistoryRetentionSettings) => historyApi.updateSettings(settings),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: historyKeys.settings() })
    },
  })
}
