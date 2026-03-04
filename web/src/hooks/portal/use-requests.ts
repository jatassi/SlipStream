import { useEffect } from 'react'

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { portalRequestsApi } from '@/api'
import { createQueryKeys } from '@/lib/query-keys'
import { usePortalAuthStore, usePortalDownloadsStore } from '@/stores'
import type { CreateRequestInput, PortalDownload, Request, RequestListFilters, RequestStatus } from '@/types'

const baseKeys = createQueryKeys('requests')
export const requestKeys = {
  ...baseKeys,
  list: (filters?: RequestListFilters) => [...baseKeys.list(), filters] as const,
}

const portalDownloadBaseKeys = createQueryKeys('portal-downloads')
export const portalDownloadKeys = {
  ...portalDownloadBaseKeys,
}

const ACTIVE_REQUEST_STATUSES = new Set<RequestStatus>([
  'approved',
  'searching',
  'downloading',
])

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
      void queryClient.invalidateQueries({ queryKey: requestKeys.all })
    },
  })
}

export function useCancelRequest() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => portalRequestsApi.cancel(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: requestKeys.all })
    },
  })
}

export function useWatchRequest() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => portalRequestsApi.watch(id),
    onSuccess: (_, id) => {
      void queryClient.invalidateQueries({ queryKey: requestKeys.detail(id) })
      void queryClient.invalidateQueries({ queryKey: requestKeys.list() })
    },
  })
}

export function useUnwatchRequest() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => portalRequestsApi.unwatch(id),
    onSuccess: (_, id) => {
      void queryClient.invalidateQueries({ queryKey: requestKeys.detail(id) })
      void queryClient.invalidateQueries({ queryKey: requestKeys.list() })
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

  const { data: requests, isLoading: requestsLoading } = useQuery({
    queryKey: requestKeys.list(),
    queryFn: () => portalRequestsApi.list(),
    enabled: isAuthenticated,
    refetchInterval: (query) => {
      const data = query.state.data
      if (data?.some((r) => ACTIVE_REQUEST_STATUSES.has(r.status))) {
        return 5000
      }
      return false
    },
  })

  useEffect(() => {
    if (requests) {
      setUserRequests(requests)
    }
  }, [requests, setUserRequests])

  const hasActiveRequests = requests?.some((r) => ACTIVE_REQUEST_STATUSES.has(r.status)) ?? false

  const { data: downloads, isLoading: downloadsLoading } = useQuery({
    queryKey: portalDownloadKeys.all,
    queryFn: () => portalRequestsApi.downloads(),
    enabled: isAuthenticated && hasActiveRequests,
    refetchInterval: hasActiveRequests ? 3000 : false,
  })

  return {
    data: isAuthenticated ? downloads : undefined,
    requests: isAuthenticated ? requests : undefined,
    isLoading: requestsLoading || downloadsLoading,
  }
}
