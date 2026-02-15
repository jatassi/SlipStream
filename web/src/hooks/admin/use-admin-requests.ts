import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import * as adminApi from '@/api/admin'
import type {
  ApproveRequestInput,
  BatchApproveInput,
  BatchDenyInput,
  DenyRequestInput,
  RequestListFilters,
} from '@/types'

export const adminRequestKeys = {
  all: ['admin', 'requests'] as const,
  lists: () => [...adminRequestKeys.all, 'list'] as const,
  list: (filters?: RequestListFilters) => [...adminRequestKeys.lists(), filters] as const,
  details: () => [...adminRequestKeys.all, 'detail'] as const,
  detail: (id: number) => [...adminRequestKeys.details(), id] as const,
}

export function useAdminRequests(filters?: RequestListFilters) {
  return useQuery({
    queryKey: adminRequestKeys.list(filters),
    queryFn: () => adminApi.listRequests(filters),
  })
}

export function useAdminRequest(id: number) {
  return useQuery({
    queryKey: adminRequestKeys.detail(id),
    queryFn: () => adminApi.getRequest(id),
    enabled: !!id,
  })
}

export function useApproveRequest() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: number; input: ApproveRequestInput }) =>
      adminApi.approveRequest(id, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminRequestKeys.all })
    },
  })
}

export function useDenyRequest() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: number; input?: DenyRequestInput }) =>
      adminApi.denyRequest(id, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminRequestKeys.all })
    },
  })
}

export function useBatchApproveRequests() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: BatchApproveInput) => adminApi.batchApprove(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminRequestKeys.all })
    },
  })
}

export function useBatchDenyRequests() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: BatchDenyInput) => adminApi.batchDeny(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminRequestKeys.all })
    },
  })
}

export function useDeleteRequest() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => adminApi.deleteRequest(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminRequestKeys.all })
    },
  })
}

export function useBatchDeleteRequests() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (ids: number[]) => adminApi.batchDelete(ids),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminRequestKeys.all })
    },
  })
}
