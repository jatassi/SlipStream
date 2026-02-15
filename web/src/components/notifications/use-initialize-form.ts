import { useEffect } from 'react'

import type { CreateNotificationInput, Notification } from '@/types'

import { adminEventTriggers, defaultFormData } from './notification-dialog-types'
import type { usePlexState } from './use-plex-state'

type InitFormOptions = {
  open: boolean
  notification?: Notification | null
  eventTriggers?: { key: string }[]
  plex: ReturnType<typeof usePlexState>
  setFormData: React.Dispatch<React.SetStateAction<CreateNotificationInput>>
  setShowAdvanced: React.Dispatch<React.SetStateAction<boolean>>
}

export function useInitializeForm({ open, notification, eventTriggers, plex, setFormData, setShowAdvanced }: InitFormOptions) {
  useEffect(() => {
    if (!open) {
      plex.cleanupPolling()
      return
    }
    if (notification) {
      setFormData(notification as unknown as CreateNotificationInput)
      if (notification.type === 'plex' && notification.settings.authToken) {
        void plex.fetchServers(notification.settings.authToken as string)
      }
    } else {
      setFormData(buildResetData(eventTriggers))
    }
    setShowAdvanced(false)
    plex.resetState()
    // eslint-disable-next-line react-hooks/exhaustive-deps -- only run when open/notification/eventTriggers change
  }, [open, notification, eventTriggers])
}

function buildResetData(eventTriggers?: { key: string }[]) {
  const resetData = { ...defaultFormData }
  if (eventTriggers) {
    adminEventTriggers.forEach((t) => {
      ;(resetData as unknown as Record<string, unknown>)[t.key] = false
    })
    eventTriggers.forEach((t) => {
      ;(resetData as unknown as Record<string, unknown>)[t.key] = true
    })
  }
  return resetData
}
