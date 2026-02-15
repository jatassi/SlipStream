import type { CreateNotificationInput, Notification, NotifierSchema } from '@/types'

export type EventTrigger = {
  key: string
  label: string
  description?: string
}

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
  /** Custom event triggers to display. If not provided, uses default admin triggers */
  eventTriggers?: EventTrigger[]
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
  onGrab: true,
  onImport: true,
  onUpgrade: true,
  onMovieAdded: false,
  onMovieDeleted: false,
  onSeriesAdded: false,
  onSeriesDeleted: false,
  onHealthIssue: true,
  onHealthRestored: true,
  onAppUpdate: false,
  includeHealthWarnings: true,
  tags: [],
}

export const adminEventTriggers: EventTrigger[] = [
  { key: 'onGrab', label: 'On Grab', description: 'When a release is grabbed' },
  { key: 'onImport', label: 'On Import', description: 'When a file is imported to the library' },
  { key: 'onUpgrade', label: 'On Upgrade', description: 'When a quality upgrade is imported' },
  { key: 'onMovieAdded', label: 'On Movie Added', description: 'When a movie is added' },
  { key: 'onMovieDeleted', label: 'On Movie Deleted', description: 'When a movie is removed' },
  { key: 'onSeriesAdded', label: 'On Series Added', description: 'When a series is added' },
  { key: 'onSeriesDeleted', label: 'On Series Deleted', description: 'When a series is removed' },
  { key: 'onHealthIssue', label: 'On Health Issue', description: 'When a health check fails' },
  {
    key: 'onHealthRestored',
    label: 'On Health Restored',
    description: 'When a health issue is resolved',
  },
  { key: 'onAppUpdate', label: 'On App Update', description: 'When the application is updated' },
]
