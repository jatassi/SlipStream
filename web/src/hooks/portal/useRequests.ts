import { useEffect, useMemo } from 'react'

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { portalRequestsApi } from '@/api'
import { usePortalAuthStore, usePortalDownloadsStore } from '@/stores'
import type { CreateRequestInput, PortalDownload, Request, RequestListFilters } from '@/types'

export const requestKeys = {
  all: ['requests'] as const,
  lists: () => [...requestKeys.all, 'list'] as const,
  list: (filters?: RequestListFilters) => [...requestKeys.lists(), filters] as const,
  details: () => [...requestKeys.all, 'detail'] as const,
  detail: (id: number) => [...requestKeys.details(), id] as const,
}

export function useRequests(filters?: RequestListFilters) {
  return useQuery({
    queryKey: requestKeys.list(filters),
    queryFn: () => portalRequestsApi.list(filters),
  })
}

export function useRequest(id: number) {
  return useQuery({
    queryKey: requestKeys.detail(id),
    queryFn: () => portalRequestsApi.get(id),
    enabled: !!id,
  })
}

export function useCreateRequest() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateRequestInput) => portalRequestsApi.create(data),
    onSuccess: (request) => {
      // Immediately add to store for download matching (handles race condition where
      // queue:state arrives before useEffect syncs the query cache to the store)
      const store = usePortalDownloadsStore.getState()
      const currentRequests = store.userRequests
      if (!currentRequests.some((r) => r.id === request.id)) {
        store.setUserRequests([...currentRequests, request])
      }

      // Also update the query cache optimistically to prevent invalidation from
      // replacing userRequests with stale server data (if server hasn't persisted yet)
      queryClient.setQueryData<Request[]>(requestKeys.list(), (old) => {
        if (!old) {
          return [request]
        }
        if (old.some((r) => r.id === request.id)) {
          return old
        }
        return [...old, request]
      })

      // Invalidate to get fresh data from server
      queryClient.invalidateQueries({ queryKey: requestKeys.all })
    },
  })
}

export function useCancelRequest() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => portalRequestsApi.cancel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: requestKeys.all })
    },
  })
}

export function useWatchRequest() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => portalRequestsApi.watch(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: requestKeys.detail(id) })
      queryClient.invalidateQueries({ queryKey: requestKeys.lists() })
    },
  })
}

export function useUnwatchRequest() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => portalRequestsApi.unwatch(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: requestKeys.detail(id) })
      queryClient.invalidateQueries({ queryKey: requestKeys.lists() })
    },
  })
}

export function usePortalDownloads(): {
  data: PortalDownload[] | undefined
  requests: Request[] | undefined
  isLoading: boolean
} {
  const isAuthenticated = usePortalAuthStore((s) => s.isAuthenticated)
  const setUserRequests = usePortalDownloadsStore((s) => s.setUserRequests)

  // Subscribe to queue directly - this triggers re-render when queue updates
  // Queue data comes from WebSocket 'queue:state' broadcasts (no fallback polling
  // since the admin queue API requires session auth, not portal JWT auth)
  const queue = usePortalDownloadsStore((s) => s.queue)
  const matches = usePortalDownloadsStore((s) => s.matches)

  // Fetch user's requests to enable matching
  const { data: requests, isLoading } = useQuery({
    queryKey: requestKeys.list(),
    queryFn: () => portalRequestsApi.list(),
    enabled: isAuthenticated,
  })

  // Sync requests to the store for matching
  useEffect(() => {
    if (requests) {
      setUserRequests(requests)
    }
  }, [requests, setUserRequests])

  // Compute matched downloads - useMemo ensures stable reference when inputs don't change
  const downloads = useMemo((): PortalDownload[] => {
    console.log('[usePortalDownloads] Computing downloads', {
      queueLength: queue.length,
      matchesSize: matches.size,
      isAuthenticated,
    })
    const result: PortalDownload[] = []
    for (const item of queue) {
      const match = matches.get(item.id)
      if (match) {
        result.push({
          id: item.id,
          clientId: item.clientId,
          clientName: item.clientName,
          title: item.title,
          mediaType: item.mediaType,
          status: item.status,
          progress: item.progress,
          size: item.size,
          downloadedSize: item.downloadedSize,
          downloadSpeed: item.downloadSpeed,
          eta: item.eta,
          season: item.season,
          episode: item.episode,
          movieId: item.movieId,
          seriesId: item.seriesId,
          seasonNumber: item.seasonNumber,
          isSeasonPack: item.isSeasonPack,
          requestId: match.requestId,
          requestTitle: match.requestTitle,
          requestMediaId: match.requestMediaId,
          tmdbId: match.tmdbId,
          tvdbId: match.tvdbId,
        })
      }
    }
    console.log('[usePortalDownloads] Downloads computed', { count: result.length })
    return result
  }, [queue, matches, isAuthenticated])

  return {
    data: isAuthenticated ? downloads : undefined,
    requests: isAuthenticated ? requests : undefined,
    isLoading,
  }
}
