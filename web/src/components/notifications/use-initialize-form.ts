import { useEffect, useState } from 'react'

import type { CreateNotificationInput, Notification } from '@/types'

import { defaultFormData } from './notification-dialog-types'
import type { usePlexState } from './use-plex-state'

type InitFormOptions = {
  open: boolean
  notification?: Notification | null
  defaultEventToggles?: Record<string, boolean>
  plex: ReturnType<typeof usePlexState>
  setFormData: React.Dispatch<React.SetStateAction<CreateNotificationInput>>
  setShowAdvanced: React.Dispatch<React.SetStateAction<boolean>>
}

function toFormData(notification: Notification): CreateNotificationInput {
  const { id: _, createdAt: _1, updatedAt: _2, ...rest } = notification
  return rest
}

export function useInitializeForm({ open, notification, defaultEventToggles, plex, setFormData, setShowAdvanced }: InitFormOptions) {
  const { cleanupPolling, fetchServers, resetState } = plex
  const [prevOpen, setPrevOpen] = useState(false)
  const [prevNotification, setPrevNotification] = useState(notification)

  // Render-time state adjustment: sync form state when dialog opens or notification changes
  if (open && (!prevOpen || notification !== prevNotification)) {
    setPrevOpen(open)
    setPrevNotification(notification)
    if (notification) {
      setFormData(toFormData(notification))
    } else {
      const resetData = { ...defaultFormData }
      if (defaultEventToggles) {
        resetData.eventToggles = { ...defaultEventToggles }
      }
      setFormData(resetData)
    }
    setShowAdvanced(false)
    resetState()
  }
  if (!open && prevOpen) {
    setPrevOpen(false)
  }

  // Side effects: cleanup polling on close, fetch Plex servers on open
  useEffect(() => {
    if (!open) {
      cleanupPolling()
      return
    }
    if (notification?.type === 'plex' && notification.settings.authToken) {
      void fetchServers(notification.settings.authToken as string)
    }
  }, [open, notification, cleanupPolling, fetchServers])
}
