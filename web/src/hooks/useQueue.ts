import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { queueApi } from '@/api'

export const queueKeys = {
  all: ['queue'] as const,
  list: () => [...queueKeys.all, 'list'] as const,
  stats: () => [...queueKeys.all, 'stats'] as const,
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

interface QueueItemParams {
  clientId: number
  id: string
}

interface RemoveParams extends QueueItemParams {
  deleteFiles?: boolean
}

export function useRemoveFromQueue() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ clientId, id, deleteFiles = false }: RemoveParams) =>
      queueApi.remove(clientId, id, deleteFiles),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queueKeys.all })
    },
  })
}

export function usePauseQueueItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ clientId, id }: QueueItemParams) =>
      queueApi.pause(clientId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queueKeys.all })
    },
  })
}

export function useResumeQueueItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ clientId, id }: QueueItemParams) =>
      queueApi.resume(clientId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queueKeys.all })
    },
  })
}
