import { useEffect, useState } from 'react'

import type { CreateNotificationInput, Notification } from '@/types'

import { adminEventTriggers, defaultFormData } from './notification-dialog-types'
import type { usePlexState } from './use-plex-state'

type TriggerFields = Record<string, boolean>

export function asTriggerFields(data: unknown): TriggerFields {
  return data as TriggerFields
}

type InitFormOptions = {
  open: boolean
  notification?: Notification | null
  eventTriggers?: { key: string }[]
  plex: ReturnType<typeof usePlexState>
  setFormData: React.Dispatch<React.SetStateAction<CreateNotificationInput>>
  setShowAdvanced: React.Dispatch<React.SetStateAction<boolean>>
}

function toFormData(notification: Notification): CreateNotificationInput {
  const { id: _, createdAt: _1, updatedAt: _2, ...rest } = notification
  return rest
}

export function useInitializeForm({ open, notification, eventTriggers, plex, setFormData, setShowAdvanced }: InitFormOptions) {
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
      setFormData(buildResetData(eventTriggers))
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

function buildResetData(eventTriggers?: { key: string }[]) {
  const resetData = { ...defaultFormData }
  if (eventTriggers) {
    adminEventTriggers.forEach((t) => {
      asTriggerFields(resetData)[t.key] = false
    })
    eventTriggers.forEach((t) => {
      asTriggerFields(resetData)[t.key] = true
    })
  }
  return resetData
}
