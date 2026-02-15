import { useState } from 'react'

import { toast } from 'sonner'

import {
  useBatchDeleteRequests,
  useBatchDenyRequests,
  useDeleteRequest,
  useDenyRequest,
} from '@/hooks'

export function useRequestDialogs(selectedIds: Set<number>, clearSelection: () => void) {
  const deny = useDenyDialogs(selectedIds, clearSelection)
  const del = useDeleteDialogs(selectedIds, clearSelection)

  return { ...deny, ...del }
}

function useDenyDialogs(selectedIds: Set<number>, clearSelection: () => void) {
  const [showDenyDialog, setShowDenyDialog] = useState(false)
  const [denyReason, setDenyReason] = useState('')
  const [pendingDenyId, setPendingDenyId] = useState<number | null>(null)

  const denyMutation = useDenyRequest()
  const batchDenyMutation = useBatchDenyRequests()

  const openDenyDialog = (id: number | null = null) => {
    setPendingDenyId(id)
    setDenyReason('')
    setShowDenyDialog(true)
  }

  const handleDeny = async () => {
    try {
      if (pendingDenyId) {
        await denyMutation.mutateAsync({
          id: pendingDenyId,
          input: denyReason ? { reason: denyReason } : undefined,
        })
        toast.success('Request denied')
      } else if (selectedIds.size > 0) {
        await batchDenyMutation.mutateAsync({
          ids: [...selectedIds],
          reason: denyReason || undefined, // intentional ||: empty string should fallback
        })
        toast.success(`${selectedIds.size} requests denied`)
        clearSelection()
      }
      setShowDenyDialog(false)
    } catch {
      toast.error('Failed to deny request(s)')
    }
  }

  return {
    showDenyDialog,
    setShowDenyDialog,
    denyReason,
    setDenyReason,
    pendingDenyId,
    denyMutation,
    batchDenyMutation,
    handleDeny,
    openDenyDialog,
  }
}

function useDeleteDialogs(selectedIds: Set<number>, clearSelection: () => void) {
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [pendingDeleteId, setPendingDeleteId] = useState<number | null>(null)
  const [showBatchDeleteDialog, setShowBatchDeleteDialog] = useState(false)
  const deleteMutation = useDeleteRequest()
  const batchDeleteMutation = useBatchDeleteRequests()

  const openDeleteDialog = (id: number) => {
    setPendingDeleteId(id)
    setShowDeleteDialog(true)
  }

  const handleDelete = async () => {
    if (!pendingDeleteId) { return }
    try {
      await deleteMutation.mutateAsync(pendingDeleteId)
      toast.success('Request deleted')
      setShowDeleteDialog(false)
      setPendingDeleteId(null)
    } catch {
      toast.error('Failed to delete request')
    }
  }

  const handleBatchDelete = async () => {
    if (selectedIds.size === 0) { return }
    try {
      const result = await batchDeleteMutation.mutateAsync([...selectedIds])
      toast.success(`${result.deleted} request${result.deleted === 1 ? '' : 's'} deleted`)
      setShowBatchDeleteDialog(false)
      clearSelection()
    } catch {
      toast.error('Failed to delete requests')
    }
  }

  return {
    showDeleteDialog, setShowDeleteDialog, deleteMutation, handleDelete, openDeleteDialog,
    showBatchDeleteDialog, setShowBatchDeleteDialog, batchDeleteMutation, handleBatchDelete,
  }
}
