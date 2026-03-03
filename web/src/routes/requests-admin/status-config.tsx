import { getStatusConfig } from '@/lib/request-status-config'

export type { StatusConfigEntry } from '@/lib/request-status-config'

export type SearchModalState = {
  open: boolean
  mediaType: 'movie' | 'series'
  mediaId: number
  mediaTitle: string
  tmdbId?: number
  imdbId?: string
  tvdbId?: number
  qualityProfileId: number
  year?: number
  season?: number
  pendingSeasons?: number[]
}

export const STATUS_CONFIG = getStatusConfig('sm')
