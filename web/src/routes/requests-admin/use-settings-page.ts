import { useState } from 'react'

import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

import {
  systemKeys,
  useDeleteNotification,
  useGlobalLoading,
  useNotifications,
  useNotificationSchemas,
  useRequestSettings,
  useRootFolders,
  useTestNotification,
  useUpdateNotification,
  useUpdateRequestSettings,
} from '@/hooks'
import type { Notification, RequestSettings } from '@/types'

function useNotificationMutationHandlers() {
  const deleteMutation = useDeleteNotification()
  const testMutation = useTestNotification()
  const updateMutation = useUpdateNotification()

  const handleToggleNotificationEnabled = async (id: number, enabled: boolean) => {
    try {
      await updateMutation.mutateAsync({ id, data: { enabled } })
      toast.success(enabled ? 'Notification enabled' : 'Notification disabled')
    } catch {
      toast.error('Failed to update notification')
    }
  }

  const handleTestNotification = async (id: number) => {
    try {
      const result = await testMutation.mutateAsync(id)
      if (result.success) {
        toast.success(result.message || 'Test notification sent')
      } else {
        toast.error(result.message || 'Test notification failed')
      }
    } catch {
      toast.error('Failed to test notification')
    }
  }

  const handleDeleteNotification = async (id: number) => {
    try {
      await deleteMutation.mutateAsync(id)
      toast.success('Notification deleted')
    } catch {
      toast.error('Failed to delete notification')
    }
  }

  return {
    testNotificationMutation: testMutation,
    handleToggleNotificationEnabled,
    handleTestNotification,
    handleDeleteNotification,
  }
}

function useSettingsNotifications() {
  const [showNotificationDialog, setShowNotificationDialog] = useState(false)
  const [editingNotification, setEditingNotification] = useState<Notification | null>(null)
  const { data: notifications } = useNotifications()
  const { data: schemas } = useNotificationSchemas()
  const mutations = useNotificationMutationHandlers()

  const getTypeName = (type: string) => schemas?.find((s) => s.type === type)?.name ?? type

  const handleOpenAddNotification = () => {
    setEditingNotification(null)
    setShowNotificationDialog(true)
  }

  const handleOpenEditNotification = (notification: Notification) => {
    setEditingNotification(notification)
    setShowNotificationDialog(true)
  }

  return {
    notifications,
    showNotificationDialog,
    setShowNotificationDialog,
    editingNotification,
    getTypeName,
    handleOpenAddNotification,
    handleOpenEditNotification,
    ...mutations,
  }
}

type SyncFormParams = {
  settings: RequestSettings | undefined
  prevSettings: RequestSettings | undefined
  setFormData: (data: Partial<RequestSettings>) => void
  setHasChanges: (v: boolean) => void
  setPrevSettings: (s: RequestSettings | undefined) => void
}

function syncFormFromSettings(params: SyncFormParams) {
  if (params.settings === params.prevSettings) {
    return
  }
  params.setPrevSettings(params.settings)
  if (!params.settings) {
    return
  }
  params.setFormData({
    enabled: params.settings.enabled,
    defaultMovieQuota: params.settings.defaultMovieQuota,
    defaultSeasonQuota: params.settings.defaultSeasonQuota,
    defaultEpisodeQuota: params.settings.defaultEpisodeQuota,
    defaultRootFolderId: params.settings.defaultRootFolderId,
    adminNotifyNew: params.settings.adminNotifyNew,
    searchRateLimit: params.settings.searchRateLimit,
  })
  params.setHasChanges(false)
}

export function useRequestSettingsPage() {
  const queryClient = useQueryClient()
  const globalLoading = useGlobalLoading()
  const { data: settings, isLoading: queryLoading, isError, refetch } = useRequestSettings()
  const isLoading = queryLoading || globalLoading
  const { data: rootFolders } = useRootFolders()
  const updateMutation = useUpdateRequestSettings()
  const notificationState = useSettingsNotifications()

  const [formData, setFormData] = useState<Partial<RequestSettings>>({})
  const [hasChanges, setHasChanges] = useState(false)
  const [prevSettings, setPrevSettings] = useState<typeof settings>(undefined)
  const portalEnabled = formData.enabled ?? settings?.enabled ?? true

  syncFormFromSettings({ settings, prevSettings, setFormData, setHasChanges, setPrevSettings })

  const handleChange = <K extends keyof RequestSettings>(key: K, value: RequestSettings[K]) => {
    setFormData((prev) => ({ ...prev, [key]: value }))
    setHasChanges(true)
  }

  const handleSave = async () => {
    try {
      await updateMutation.mutateAsync(formData)
      void queryClient.invalidateQueries({ queryKey: systemKeys.status() })
      toast.success('Settings saved')
      setHasChanges(false)
    } catch {
      toast.error('Failed to save settings')
    }
  }

  return {
    isLoading,
    isError,
    refetch,
    formData,
    hasChanges,
    portalEnabled,
    rootFolders,
    updateMutation,
    handleChange,
    handleSave,
    ...notificationState,
  }
}
