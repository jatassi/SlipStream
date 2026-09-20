import { useState } from 'react'

import { toast } from 'sonner'

import {
  useAdminRequests,
  useAdminUsers,
  useDeleteRequest,
  useDenyRequest,
  usePortalEnabled,
} from '@/hooks'
import { useUIStore } from '@/stores'
import type { Request } from '@/types'

import type { RequestAction } from './request-actions'
import type { QueueSegment } from './request-status'
import { segmentOf } from './request-status'
import { useRequestApprove } from './use-request-approve'

function useRequestDecisions() {
  const denyMutation = useDenyRequest()
  const deleteMutation = useDeleteRequest()

  return {
    handleDeny: async (request: Request, reason: string) => {
      try {
        await denyMutation.mutateAsync({
          id: request.id,
          input: reason === '' ? undefined : { reason },
        })
        toast.success('Request denied')
      } catch {
        toast.error('Failed to deny request')
      }
    },
    handleDelete: async (request: Request) => {
      try {
        await deleteMutation.mutateAsync(request.id)
        toast.success('Request deleted')
      } catch {
        toast.error('Failed to delete request')
      }
    },
  }
}

export function useRequestQueuePage() {
  const globalLoading = useUIStore((s) => s.globalLoading)
  const { data: requests = [], isLoading: queryLoading, isError, refetch } = useAdminRequests()
  const { data: portalUsers } = useAdminUsers()
  const portalEnabled = usePortalEnabled()
  const approve = useRequestApprove()
  const decisions = useRequestDecisions()

  const [segment, setSegment] = useState<QueueSegment>('pending')

  const handleAction = (request: Request, action: RequestAction) => {
    switch (action.kind) {
      case 'approve': {
        void approve.handleApproveOnly(request)
        break
      }
      case 'approve-manual-search': {
        void approve.handleApproveAndManualSearch(request)
        break
      }
      case 'approve-auto-search': {
        void approve.handleApproveAndAutoSearch(request)
        break
      }
      case 'deny': {
        void decisions.handleDeny(request, action.reason)
        break
      }
      case 'delete': {
        void decisions.handleDelete(request)
        break
      }
    }
  }

  return {
    isLoading: queryLoading || globalLoading,
    isError,
    refetch,
    portalEnabled,
    segment,
    setSegment,
    visibleRequests: requests.filter((request) => segmentOf(request.status) === segment),
    requesterFor: (userId: number) => portalUsers?.find((user) => user.id === userId)?.username,
    processingRequest: approve.processingRequest,
    searchModal: approve.searchModal,
    handleSearchModalClose: approve.handleSearchModalClose,
    handleAction,
  }
}
