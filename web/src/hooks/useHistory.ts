import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { historyApi } from '@/api'
import type { ListHistoryOptions } from '@/types'

export const historyKeys = {
  all: ['history'] as const,
  lists: () => [...historyKeys.all, 'list'] as const,
  list: (filters: ListHistoryOptions) => [...historyKeys.lists(), filters] as const,
  detail: (id: number) => [...historyKeys.all, 'detail', id] as const,
}

export function useHistory(options?: ListHistoryOptions) {
  return useQuery({
    queryKey: historyKeys.list(options || {}),
    queryFn: () => historyApi.list(options),
  })
}

export function useHistoryItem(id: number) {
  return useQuery({
    queryKey: historyKeys.detail(id),
    queryFn: () => historyApi.get(id),
    enabled: !!id,
  })
}

export function useClearHistory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => historyApi.clear(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: historyKeys.all })
    },
  })
}
