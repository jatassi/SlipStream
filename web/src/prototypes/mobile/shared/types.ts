export type MediaKind = 'movie' | 'series'

export type MediaStatus =
  | 'available'
  | 'missing'
  | 'downloading'
  | 'upgradable'
  | 'unreleased'
  | 'failed'

export type MediaItem = {
  id: number
  kind: MediaKind
  title: string
  year: number
  status: MediaStatus
  quality: string
  sizeGb: number
  length: string
  rating: number
  genres: string[]
  studio: string
  overview: string
  monitored: boolean
  profile: string
  art: [string, string]
  added: string
}

export type QueueState = 'downloading' | 'paused' | 'importing' | 'queued'

export type QueueItem = {
  id: string
  mediaId: number
  episode?: string
  release: string
  progress: number
  sizeGb: number
  speedMbps: number
  etaMin: number
  state: QueueState
  protocol: 'torrent' | 'usenet'
  client: string
}

export type HistoryEvent = {
  id: string
  mediaId: number
  event: 'Grabbed' | 'Imported' | 'Upgraded' | 'Failed'
  detail: string
  ago: string
}

export type HealthIssue = {
  id: string
  level: 'warning' | 'error'
  source: string
  message: string
}

export type SettingsSection = {
  id: string
  title: string
  items: { title: string; detail: string }[]
}
