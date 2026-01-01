export interface QueueItem {
  id: string
  clientId: number
  clientName: string
  title: string
  mediaType: 'movie' | 'series' | 'unknown'
  status: 'queued' | 'downloading' | 'paused' | 'completed' | 'failed'
  progress: number
  size: number
  downloadedSize: number
  downloadSpeed: number
  eta: number
  quality?: string
  source?: string
  codec?: string
  attributes: string[]
  season?: number
  episode?: number
  downloadPath: string
}

export interface QueueStats {
  totalCount: number
  downloadingCount: number
  queuedCount: number
  pausedCount: number
  completedCount: number
  failedCount: number
}
