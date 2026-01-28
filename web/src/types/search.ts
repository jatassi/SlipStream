// Search criteria for indexer queries
export interface SearchCriteria {
  query?: string
  type?: 'search' | 'tvsearch' | 'movie'
  categories?: string // comma-separated category IDs
  imdbId?: string
  tmdbId?: number
  tvdbId?: number
  season?: number
  episode?: number
  year?: number
  limit?: number
  offset?: number
}

// Extended search criteria for scored search endpoints
export interface ScoredSearchCriteria extends SearchCriteria {
  qualityProfileId: number
}

// Re-export Protocol from indexer
import type { Protocol } from './indexer'
export type { Protocol }

// Base release info from search results
export interface ReleaseInfo {
  guid: string
  title: string
  description?: string
  downloadUrl: string
  infoUrl?: string
  size: number
  publishDate: string
  categories: number[]
  indexerId: number
  indexer: string
  protocol: Protocol
  imdbId?: number
  tmdbId?: number
  tvdbId?: number
  // Parsed quality info
  quality?: string    // "720p", "1080p", "2160p"
  source?: string     // "BluRay", "WEB-DL", "HDTV"
  resolution?: number // 720, 1080, 2160
}

// Score breakdown for desirability scoring
export interface ScoreBreakdown {
  qualityScore: number
  qualityId?: number
  qualityName?: string
  healthScore: number
  indexerScore: number
  matchScore: number
  ageScore: number
}

// Torrent-specific release info
export interface TorrentInfo extends ReleaseInfo {
  seeders: number
  leechers: number
  infoHash?: string
  magnetUrl?: string
  minimumRatio?: number
  minimumSeedTime?: number
  downloadVolumeFactor: number
  uploadVolumeFactor: number
  // Scoring fields (populated by all torrent search endpoints)
  score?: number
  normalizedScore?: number
  scoreBreakdown?: ScoreBreakdown
  // Slot enrichment fields (populated when multi-version is enabled)
  // Req 11.1.1: Search results indicate which slot each release would fill
  // Req 11.1.2: Show whether grab would be upgrade vs new fill
  targetSlotId?: number
  targetSlotNumber?: number
  targetSlotName?: string
  isSlotUpgrade?: boolean
  isSlotNewFill?: boolean
}

// Usenet-specific release info
export interface UsenetInfo extends ReleaseInfo {
  grabs?: number
  usenetAge?: number
  poster?: string
  group?: string
}

// Error from a specific indexer during search
export interface SearchIndexerError {
  indexerId: number
  indexerName: string
  error: string
}

// Search result from API
export interface SearchResult {
  releases: ReleaseInfo[]
  total: number
  indexersSearched: number
  errors?: SearchIndexerError[]
}

// Torrent search result
export interface TorrentSearchResult {
  releases: TorrentInfo[]
  total: number
  indexersSearched: number
  errors?: SearchIndexerError[]
}

// Grab request to send release to download client
export interface GrabRequest {
  release: {
    guid: string
    title: string
    downloadUrl: string
    indexerId: number
    indexer?: string
    protocol: Protocol
    size?: number
    imdbId?: number
    tmdbId?: number
    tvdbId?: number
  }
  clientId?: number
  mediaType?: 'movie' | 'episode' | 'season'
  mediaId?: number
  seriesId?: number
  seasonNumber?: number
  isSeasonPack?: boolean
  isCompleteSeries?: boolean
  // Req 11.1.3: Allow user to override auto-detected slot when grabbing
  targetSlotId?: number
}

// Result from grabbing a release
export interface GrabResult {
  success: boolean
  downloadId?: string
  clientId?: number
  clientName?: string
  error?: string
}

// Bulk grab request
export interface BulkGrabRequest {
  releases: GrabRequest['release'][]
  clientId?: number
  mediaType?: 'movie' | 'episode'
  mediaId?: number
}

// Bulk grab result
export interface BulkGrabResult {
  totalRequested: number
  successful: number
  failed: number
  results: GrabResult[]
}

// Grab history item
export interface GrabHistoryItem {
  id: number
  indexerId: number
  title: string
  successful: boolean
  createdAt: string
  data?: string
}

// IndexerStatus is defined in ./indexer
