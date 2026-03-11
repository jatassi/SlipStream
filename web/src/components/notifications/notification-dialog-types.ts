import type { CreateNotificationInput, Notification, NotificationEventGroup, NotifierSchema } from '@/types'

export type PlexServer = {
  id: string
  name: string
  owned: boolean
  address?: string
}

export type PlexSection = {
  key: number
  title: string
  type: string
}

export type NotificationDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  notification?: Notification | null
  /** Custom event groups to override the catalog. If not provided, fetches from API */
  eventGroups?: NotificationEventGroup[]
  /** Default event toggles for new notifications. If not provided, uses defaultFormData */
  defaultEventToggles?: Record<string, boolean>
  /** Custom schemas. If not provided, fetches from API */
  schemas?: NotifierSchema[]
  /** Custom create handler. If not provided, uses admin API */
  onCreate?: (data: CreateNotificationInput) => Promise<void>
  /** Custom update handler. If not provided, uses admin API */
  onUpdate?: (id: number, data: CreateNotificationInput) => Promise<void>
  /** Custom test handler. If not provided, uses admin API */
  onTest?: (data: CreateNotificationInput) => Promise<{ success: boolean; message?: string }>
}

export const defaultFormData: CreateNotificationInput = {
  name: '',
  type: 'discord',
  enabled: true,
  settings: {},
  eventToggles: {
    grab: true,
    import: true,
    upgrade: true,
    health_issue: true,
    health_restored: true,
  },
  includeHealthWarnings: true,
  tags: [],
}
