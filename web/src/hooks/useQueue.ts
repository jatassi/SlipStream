import { useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { queueApi } from '@/api'
import { useDownloadingStore } from '@/stores'

export const queueKeys = {
  all: ['queue'] as const,
  list: () => [...queueKeys.all, 'list'] as const,
  stats: () => [...queueKeys.all, 'stats'] as const,
}

export function useQueue() {
  const setQueueItems = useDownloadingStore((state) => state.setQueueItems)
  const query = useQuery({
    queryKey: queueKeys.list(),
    queryFn: () => queueApi.list(),
    // Poll faster when downloads are active, slower when idle
    // WebSocket handles immediate updates for grab/pause/resume/remove
    refetchInterval: (query) => {
      const hasActiveDownloads = (query.state.data?.length ?? 0) > 0
      return hasActiveDownloads ? 2000 : 60000 // 2s when active, 60s when idle
    },
    // Disable structural sharing to ensure store updates when items are
    // removed directly from download client (new reference on each fetch)
    structuralSharing: false,
  })

  // Sync queue items to the downloading store (including empty arrays)
  useEffect(() => {
    if (query.data !== undefined) {
      setQueueItems(query.data ?? [])
    }
  }, [query.data, setQueueItems])

  return query
}

export function useQueueStats() {
  return useQuery({
    queryKey: queueKeys.stats(),
    queryFn: () => queueApi.stats(),
    // Stats polling follows same pattern as queue list
    refetchInterval: (query) => {
      const hasActiveDownloads = (query.state.data?.totalCount ?? 0) > 0
      return hasActiveDownloads ? 2000 : 60000
    },
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
