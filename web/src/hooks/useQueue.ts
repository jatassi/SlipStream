import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { queueApi } from '@/api'

export const queueKeys = {
  all: ['queue'] as const,
  list: () => [...queueKeys.all, 'list'] as const,
  stats: () => [...queueKeys.all, 'stats'] as const,
  detail: (id: number) => [...queueKeys.all, 'detail', id] as const,
}

export function useQueue() {
  return useQuery({
    queryKey: queueKeys.list(),
    queryFn: () => queueApi.list(),
    refetchInterval: 5000, // Refresh every 5 seconds
  })
}

export function useQueueStats() {
  return useQuery({
    queryKey: queueKeys.stats(),
    queryFn: () => queueApi.stats(),
    refetchInterval: 5000,
  })
}

export function useQueueItem(id: number) {
  return useQuery({
    queryKey: queueKeys.detail(id),
    queryFn: () => queueApi.get(id),
    enabled: !!id,
  })
}

export function useRemoveFromQueue() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => queueApi.remove(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queueKeys.all })
    },
  })
}

export function usePauseQueueItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => queueApi.pause(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queueKeys.all })
    },
  })
}

export function useResumeQueueItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => queueApi.resume(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queueKeys.all })
    },
  })
}
