export interface QueueItem {
  id: number
  clientId?: number
  externalId?: string
  title: string
  mediaType: 'movie' | 'series'
  mediaId: number
  status: 'queued' | 'downloading' | 'paused' | 'completed' | 'failed'
  progress: number
  size: number
  downloadUrl?: string
  outputPath?: string
  addedAt: string
  completedAt?: string
  eta?: string
  speed?: number
}

export interface QueueStats {
  totalCount: number
  downloadingCount: number
  queuedCount: number
  pausedCount: number
  completedCount: number
  failedCount: number
}
