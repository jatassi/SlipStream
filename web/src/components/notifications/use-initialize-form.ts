import { useEffect } from 'react'

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

export function useInitializeForm({ open, notification, eventTriggers, plex, setFormData, setShowAdvanced }: InitFormOptions) {
  const { cleanupPolling, fetchServers, resetState } = plex
  useEffect(() => {
    if (!open) {
      cleanupPolling()
      return
    }
    if (notification) {
      setFormData(notification as unknown as CreateNotificationInput)
      if (notification.type === 'plex' && notification.settings.authToken) {
        void fetchServers(notification.settings.authToken as string)
      }
    } else {
      setFormData(buildResetData(eventTriggers))
    }
    setShowAdvanced(false)
    resetState()
  }, [open, notification, eventTriggers, cleanupPolling, fetchServers, resetState, setFormData, setShowAdvanced])
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
