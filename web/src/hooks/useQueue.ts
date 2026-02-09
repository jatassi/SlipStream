import { useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { queueApi } from '@/api'
import { useDownloadingStore, usePortalDownloadsStore } from '@/stores'

export const queueKeys = {
  all: ['queue'] as const,
  list: () => [...queueKeys.all, 'list'] as const,
  stats: () => [...queueKeys.all, 'stats'] as const,
}

const WS_TIMEOUT_MS = 10_000
const FALLBACK_POLL_MS = 2_000

export function useQueue(enabled = true) {
  const setQueueItems = useDownloadingStore((state) => state.setQueueItems)
  const setPortalQueue = usePortalDownloadsStore((state) => state.setQueue)

  const query = useQuery({
    queryKey: queueKeys.list(),
    queryFn: () => queueApi.list(),
    enabled,
    // Fallback polling: poll if we have active downloads or client errors,
    // and haven't received a WebSocket update in 10 seconds.
    // Force HTTP polling when errors exist so isFetching cycles for retry UX.
    refetchInterval: (q) => {
      if (!enabled) return false
      const data = q.state.data
      const hasErrors = (data?.errors?.length ?? 0) > 0
      const hasActiveDownloads = data?.items?.some(
        (item) => item.status === 'downloading' || item.status === 'queued'
      )
      if (!hasActiveDownloads && !hasErrors) return false
      if (hasErrors) return FALLBACK_POLL_MS
      const timeSinceUpdate = Date.now() - usePortalDownloadsStore.getState().lastUpdateTime
      return timeSinceUpdate > WS_TIMEOUT_MS ? FALLBACK_POLL_MS : false
    },
    // Disable structural sharing to ensure store updates when items are
    // removed directly from download client (new reference on each fetch)
    structuralSharing: false,
  })

  // Sync queue items to both stores (including empty arrays)
  useEffect(() => {
    if (query.data !== undefined) {
      const items = query.data?.items ?? []
      setQueueItems(items)
      setPortalQueue(items)
    }
  }, [query.data, setQueueItems, setPortalQueue])

  return query
}

export function useQueueStats() {
  return useQuery({
    queryKey: queueKeys.stats(),
    queryFn: () => queueApi.stats(),
    // Fallback polling: only poll if we have active downloads and
    // haven't received a WebSocket update in 10 seconds
    refetchInterval: (q) => {
      const hasActiveDownloads = (q.state.data?.totalCount ?? 0) > 0
      if (!hasActiveDownloads) return false
      const timeSinceUpdate = Date.now() - usePortalDownloadsStore.getState().lastUpdateTime
      return timeSinceUpdate > WS_TIMEOUT_MS ? FALLBACK_POLL_MS : false
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
      // Force immediate refetch to get updated status
      queryClient.refetchQueries({ queryKey: queueKeys.all })
    },
  })
}

export function useResumeQueueItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ clientId, id }: QueueItemParams) =>
      queueApi.resume(clientId, id),
    onSuccess: () => {
      // Force immediate refetch to get updated status
      queryClient.refetchQueries({ queryKey: queueKeys.all })
    },
  })
}

export function useFastForwardQueueItem() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ clientId, id }: QueueItemParams) =>
      queueApi.fastForward(clientId, id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queueKeys.all })
    },
  })
}
