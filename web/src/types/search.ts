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
}

// Usenet-specific release info
export interface UsenetInfo extends ReleaseInfo {
  grabs?: number
  usenetAge?: number
  poster?: string
  group?: string
}

// Search result from API
export interface SearchResult {
  releases: ReleaseInfo[]
  total: number
  indexersSearched: number
  errors?: string[]
}

// Torrent search result
export interface TorrentSearchResult {
  releases: TorrentInfo[]
  total: number
  indexersSearched: number
  errors?: string[]
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
  mediaType?: 'movie' | 'episode'
  mediaId?: number
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
