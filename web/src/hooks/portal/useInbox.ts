import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { portalInboxApi } from '@/api/portal'
import { usePortalAuthStore } from '@/stores/portalAuth'

export const inboxKeys = {
  all: ['portalInbox'] as const,
  list: () => [...inboxKeys.all, 'list'] as const,
  count: () => [...inboxKeys.all, 'count'] as const,
}

export function useInbox(limit = 50, offset = 0) {
  const { isAuthenticated } = usePortalAuthStore()

  return useQuery({
    queryKey: [...inboxKeys.list(), limit, offset],
    queryFn: () => portalInboxApi.list(limit, offset),
    enabled: isAuthenticated,
  })
}

export function useUnreadCount() {
  const { isAuthenticated } = usePortalAuthStore()

  return useQuery({
    queryKey: inboxKeys.count(),
    queryFn: () => portalInboxApi.unreadCount(),
    enabled: isAuthenticated,
  })
}

export function useMarkRead() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => portalInboxApi.markRead(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: inboxKeys.all })
    },
  })
}

export function useMarkAllRead() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => portalInboxApi.markAllRead(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: inboxKeys.all })
    },
  })
}
