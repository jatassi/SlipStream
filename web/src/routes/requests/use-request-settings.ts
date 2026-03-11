import { useState } from 'react'

import { toast } from 'sonner'

import {
  useCreateUserNotification,
  useDeleteUserNotification,
  useTestUserNotification,
  useUpdateUserNotification,
  useUserNotifications,
  useUserNotificationSchema,
} from '@/hooks'
import type { CreateNotificationInput, Notification, NotificationEventGroup, NotifierType, UserNotification } from '@/types'

const portalEventGroups: NotificationEventGroup[] = [
  {
    id: 'requests',
    label: 'Request Events',
    events: [
      { id: 'onApproved', label: 'Request Approved', description: 'When your request is approved by an admin' },
      { id: 'onDenied', label: 'Request Denied', description: 'When your request is denied by an admin' },
      { id: 'onAvailable', label: 'Request Available', description: 'When your requested content becomes available' },
    ],
  },
]

function toNotificationForDialog(n: UserNotification): Notification {
  return {
    id: n.id,
    name: n.name,
    type: n.type as NotifierType,
    enabled: n.enabled,
    settings: n.settings,
    eventToggles: {
      onAvailable: n.onAvailable,
      onApproved: n.onApproved,
      onDenied: n.onDenied,
    },
    includeHealthWarnings: false,
    tags: [],
  }
}

function extractPortalEvents(data: CreateNotificationInput) {
  const toggles = data.eventToggles
  return {
    type: data.type,
    name: data.name,
    settings: data.settings,
    onAvailable: toggles ? toggles.onAvailable : true,
    onApproved: toggles ? toggles.onApproved : true,
    onDenied: toggles ? toggles.onDenied : true,
    enabled: data.enabled ?? true,
  }
}

function buildTogglePayload(notification: UserNotification, enabled: boolean) {
  return {
    id: notification.id,
    data: {
      type: notification.type,
      name: notification.name,
      settings: notification.settings,
      onAvailable: notification.onAvailable,
      onApproved: notification.onApproved,
      onDenied: notification.onDenied,
      enabled,
    },
  }
}

const portalDefaultToggles: Record<string, boolean> = {
  onApproved: true,
  onDenied: true,
  onAvailable: true,
}

function useNotificationMutations() {
  const createMutation = useCreateUserNotification()
  const updateMutation = useUpdateUserNotification()
  const deleteMutation = useDeleteUserNotification()
  const testMutation = useTestUserNotification()

  const handleDelete = async (id: number) => {
    try {
      await deleteMutation.mutateAsync(id)
      toast.success('Notification deleted')
    } catch {
      toast.error('Failed to delete notification')
    }
  }

  const handleTest = async (id: number) => {
    try {
      await testMutation.mutateAsync(id)
      toast.success('Test notification sent')
    } catch {
      toast.error('Failed to send test notification')
    }
  }

  const handleToggleEnabled = async (notification: UserNotification, enabled: boolean) => {
    try {
      await updateMutation.mutateAsync(buildTogglePayload(notification, enabled))
      toast.success(enabled ? 'Notification enabled' : 'Notification disabled')
    } catch {
      toast.error('Failed to update notification')
    }
  }

  const handleCreateNotification = async (data: CreateNotificationInput) => {
    await createMutation.mutateAsync(extractPortalEvents(data))
  }

  const handleUpdateNotification = async (id: number, data: CreateNotificationInput) => {
    await updateMutation.mutateAsync({ id, data: extractPortalEvents(data) })
  }

  return {
    isTestPending: testMutation.isPending,
    handleDelete,
    handleTest,
    handleToggleEnabled,
    handleCreateNotification,
    handleUpdateNotification,
  }
}

export function useNotificationsSection() {
  const { data: notifications = [], isLoading } = useUserNotifications()
  const { data: schemas = [] } = useUserNotificationSchema()
  const mutations = useNotificationMutations()

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingNotification, setEditingNotification] = useState<UserNotification | null>(null)

  const getTypeName = (type: string) => schemas.find((s) => s.type === type)?.name ?? type

  const handleCreate = () => {
    setEditingNotification(null)
    setDialogOpen(true)
  }

  const handleEdit = (notification: UserNotification) => {
    setEditingNotification(notification)
    setDialogOpen(true)
  }

  const notificationForDialog: Notification | null = editingNotification
    ? toNotificationForDialog(editingNotification)
    : null

  return {
    notifications,
    isLoading,
    schemas,
    dialogOpen,
    setDialogOpen,
    notificationForDialog,
    portalEventGroups,
    portalDefaultToggles,
    getTypeName,
    handleCreate,
    handleEdit,
    ...mutations,
  }
}
