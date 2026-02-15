import { useState } from 'react'

import { toast } from 'sonner'

import {
  useDeleteNotification,
  useGlobalLoading,
  useNotifications,
  useNotificationSchemas,
  useTestNotification,
  useUpdateNotification,
} from '@/hooks'
import { withToast } from '@/lib/with-toast'
import type { Notification } from '@/types'

function openDialog(
  setEditing: (n: Notification | null) => void,
  setShow: (v: boolean) => void,
  notification: Notification | null,
) {
  setEditing(notification)
  setShow(true)
}

export function useNotificationsPage() {
  const [showDialog, setShowDialog] = useState(false)
  const [editingNotification, setEditingNotification] = useState<Notification | null>(null)
  const globalLoading = useGlobalLoading()
  const { data: notifications, isLoading: queryLoading, isError, refetch } = useNotifications()
  const { data: schemas } = useNotificationSchemas()
  const deleteMutation = useDeleteNotification()
  const testMutation = useTestNotification()
  const updateMutation = useUpdateNotification()

  const getTypeName = (type: string) => schemas?.find((s) => s.type === type)?.name ?? type

  const handleToggleEnabled = withToast(async (id: number, enabled: boolean) => {
    await updateMutation.mutateAsync({ id, data: { enabled } })
    toast.success(enabled ? 'Notification enabled' : 'Notification disabled')
  }, 'Failed to update notification')

  const handleTest = withToast(async (id: number) => {
    const result = await testMutation.mutateAsync(id)
    // intentional ||: empty string should use fallback
    const toastFn = result.success ? toast.success : toast.error
    toastFn(result.message || (result.success ? 'Test notification sent' : 'Test notification failed'))
  }, 'Failed to test notification')

  const handleDelete = withToast(async (id: number) => {
    await deleteMutation.mutateAsync(id)
    toast.success('Notification deleted')
  }, 'Failed to delete notification')

  return {
    notifications, isLoading: queryLoading || globalLoading, isError, refetch,
    showDialog, setShowDialog, editingNotification,
    testPending: testMutation.isPending, getTypeName,
    handleOpenAdd: () => openDialog(setEditingNotification, setShowDialog, null),
    handleOpenEdit: (n: Notification) => openDialog(setEditingNotification, setShowDialog, n),
    handleToggleEnabled, handleTest, handleDelete,
  }
}
