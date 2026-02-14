export type NotifierType =
  | 'discord'
  | 'telegram'
  | 'webhook'
  | 'email'
  | 'slack'
  | 'pushover'
  | 'gotify'
  | 'ntfy'
  | 'apprise'
  | 'pushbullet'
  | 'join'
  | 'prowl'
  | 'simplepush'
  | 'signal'
  | 'custom_script'
  | 'plex'

export type Notification = {
  id: number
  name: string
  type: NotifierType
  enabled: boolean
  settings: Record<string, unknown>
  onGrab: boolean
  onImport: boolean
  onUpgrade: boolean
  onMovieAdded: boolean
  onMovieDeleted: boolean
  onSeriesAdded: boolean
  onSeriesDeleted: boolean
  onHealthIssue: boolean
  onHealthRestored: boolean
  onAppUpdate: boolean
  includeHealthWarnings: boolean
  // Portal-specific event triggers
  onAvailable?: boolean
  onApproved?: boolean
  onDenied?: boolean
  tags: number[]
  createdAt?: string
  updatedAt?: string
}

export type CreateNotificationInput = {
  name: string
  type: NotifierType
  enabled?: boolean
  settings: Record<string, unknown>
  onGrab?: boolean
  onImport?: boolean
  onUpgrade?: boolean
  onMovieAdded?: boolean
  onMovieDeleted?: boolean
  onSeriesAdded?: boolean
  onSeriesDeleted?: boolean
  onHealthIssue?: boolean
  onHealthRestored?: boolean
  onAppUpdate?: boolean
  includeHealthWarnings?: boolean
  tags?: number[]
}

export type UpdateNotificationInput = {
  name?: string
  type?: NotifierType
  enabled?: boolean
  settings?: Record<string, unknown>
  onGrab?: boolean
  onImport?: boolean
  onUpgrade?: boolean
  onMovieAdded?: boolean
  onMovieDeleted?: boolean
  onSeriesAdded?: boolean
  onSeriesDeleted?: boolean
  onHealthIssue?: boolean
  onHealthRestored?: boolean
  onAppUpdate?: boolean
  includeHealthWarnings?: boolean
  tags?: number[]
}

export type NotificationTestResult = {
  success: boolean
  message: string
}

export type FieldType = 'text' | 'password' | 'number' | 'bool' | 'select' | 'url' | 'action'

export type SelectOption = {
  value: string
  label: string
}

export type SettingsField = {
  name: string
  label: string
  type: FieldType
  required?: boolean
  placeholder?: string
  helpText?: string
  default?: unknown
  options?: SelectOption[]
  advanced?: boolean
  actionEndpoint?: string
  actionLabel?: string
  actionType?: string
}

export type NotifierSchema = {
  type: NotifierType
  name: string
  description?: string
  infoUrl?: string
  fields: SettingsField[]
}

export type NotificationProviderSchema = NotifierSchema
