import type { TorrentInfo } from '@/types'

export type SearchModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  qualityProfileId: number
  movieId?: number
  movieTitle?: string
  tmdbId?: number
  imdbId?: string
  year?: number
  seriesId?: number
  seriesTitle?: string
  tvdbId?: number
  season?: number
  episode?: number
  onGrabSuccess?: () => void
}

export type SortColumn = 'score' | 'title' | 'quality' | 'slot' | 'indexer' | 'size' | 'age' | 'peers'
export type SortDirection = 'asc' | 'desc'

export const RESOLUTION_ORDER: Record<string, number> = {
  '2160p': 4,
  '1080p': 3,
  '720p': 2,
  '480p': 1,
  SD: 0,
}

export type ReleaseGrabHandler = (release: TorrentInfo) => void
