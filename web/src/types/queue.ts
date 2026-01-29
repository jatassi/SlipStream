export interface QueueItem {
  id: string
  clientId: number
  clientName: string
  clientType: string
  title: string
  releaseName: string
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
  hdrFormats?: string[]
  season?: number
  episode?: number
  downloadPath: string
  // Library mapping - populated when download is initiated via auto-search
  movieId?: number
  seriesId?: number
  seasonNumber?: number
  episodeId?: number
  isSeasonPack?: boolean
  isCompleteSeries?: boolean
}

export interface QueueStats {
  totalCount: number
  downloadingCount: number
  queuedCount: number
  pausedCount: number
  completedCount: number
  failedCount: number
}
