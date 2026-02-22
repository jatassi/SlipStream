import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import * as adminApi from '@/api/admin'
import type { CreateInvitationRequest } from '@/types'

const adminInvitationKeys = {
  all: ['admin', 'invitations'] as const,
  lists: () => [...adminInvitationKeys.all, 'list'] as const,
  list: () => [...adminInvitationKeys.lists()] as const,
}

export function useAdminInvitations() {
  return useQuery({
    queryKey: adminInvitationKeys.list(),
    queryFn: () => adminApi.listInvitations(),
  })
}

export function useCreateInvitation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateInvitationRequest) => adminApi.createInvitation(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminInvitationKeys.all })
    },
  })
}

export function useDeleteInvitation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => adminApi.deleteInvitation(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminInvitationKeys.all })
    },
  })
}

export function useAdminResendInvitation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => adminApi.resendInvitation(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: adminInvitationKeys.all })
    },
  })
}

export { getInvitationLink } from '@/api/admin/invitations'
