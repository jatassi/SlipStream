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

export interface Notification {
  id: number
  name: string
  type: NotifierType
  enabled: boolean
  settings: Record<string, unknown>
  onGrab: boolean
  onDownload: boolean
  onUpgrade: boolean
  onMovieAdded: boolean
  onMovieDeleted: boolean
  onSeriesAdded: boolean
  onSeriesDeleted: boolean
  onHealthIssue: boolean
  onHealthRestored: boolean
  onAppUpdate: boolean
  includeHealthWarnings: boolean
  tags: number[]
  createdAt: string
  updatedAt: string
}

export interface CreateNotificationInput {
  name: string
  type: NotifierType
  enabled?: boolean
  settings: Record<string, unknown>
  onGrab?: boolean
  onDownload?: boolean
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

export interface UpdateNotificationInput {
  name?: string
  type?: NotifierType
  enabled?: boolean
  settings?: Record<string, unknown>
  onGrab?: boolean
  onDownload?: boolean
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

export interface NotificationTestResult {
  success: boolean
  message: string
}

export type FieldType = 'text' | 'password' | 'number' | 'bool' | 'select' | 'url'

export interface SelectOption {
  value: string
  label: string
}

export interface SettingsField {
  name: string
  label: string
  type: FieldType
  required?: boolean
  placeholder?: string
  helpText?: string
  default?: unknown
  options?: SelectOption[]
  advanced?: boolean
}

export interface NotifierSchema {
  type: NotifierType
  name: string
  description?: string
  infoUrl?: string
  fields: SettingsField[]
}
