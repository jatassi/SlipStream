import { useAdminRequests, useDeveloperMode, useGlobalLoading, usePortalEnabled } from '@/hooks'

import { useRequestApprove } from './use-request-approve'
import { useRequestDialogs } from './use-request-dialogs'
import { useRequestSelection } from './use-request-selection'

export function useRequestQueuePage() {
  const globalLoading = useGlobalLoading()
  const { data: requests = [], isLoading: queryLoading, isError, refetch } = useAdminRequests()
  const isLoading = queryLoading || globalLoading
  const developerMode = useDeveloperMode()
  const portalEnabled = usePortalEnabled()

  const selection = useRequestSelection(requests)
  const approve = useRequestApprove()
  const dialogs = useRequestDialogs(selection.selectedIds, selection.clearSelection)

  return {
    ...selection,
    ...approve,
    ...dialogs,
    isLoading,
    isError,
    refetch,
    developerMode,
    portalEnabled,
  }
}
