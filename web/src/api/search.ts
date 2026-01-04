import { apiFetch } from './client'
import type {
  SearchCriteria,
  SearchResult,
  TorrentSearchResult,
  GrabRequest,
  GrabResult,
  BulkGrabRequest,
  BulkGrabResult,
  GrabHistoryItem,
  IndexerStatus,
} from '@/types'

// Build query string from search criteria
function buildSearchQuery(criteria: SearchCriteria): string {
  const params = new URLSearchParams()

  if (criteria.query) params.set('query', criteria.query)
  if (criteria.type) params.set('type', criteria.type)
  if (criteria.categories) params.set('categories', criteria.categories)
  if (criteria.imdbId) params.set('imdbId', criteria.imdbId)
  if (criteria.tmdbId) params.set('tmdbId', String(criteria.tmdbId))
  if (criteria.tvdbId) params.set('tvdbId', String(criteria.tvdbId))
  if (criteria.season) params.set('season', String(criteria.season))
  if (criteria.episode) params.set('episode', String(criteria.episode))
  if (criteria.year) params.set('year', String(criteria.year))
  if (criteria.limit) params.set('limit', String(criteria.limit))
  if (criteria.offset) params.set('offset', String(criteria.offset))

  return params.toString()
}

export const searchApi = {
  // General search
  search: (criteria: SearchCriteria) =>
    apiFetch<SearchResult>(`/search?${buildSearchQuery(criteria)}`),

  // Movie-specific search (returns torrent info with seeders/leechers)
  searchMovie: (criteria: SearchCriteria) =>
    apiFetch<TorrentSearchResult>(`/search/movie?${buildSearchQuery(criteria)}`),

  // TV-specific search (returns torrent info with seeders/leechers)
  searchTV: (criteria: SearchCriteria) =>
    apiFetch<TorrentSearchResult>(`/search/tv?${buildSearchQuery(criteria)}`),

  // Torrent search with torrent-specific info
  searchTorrents: (criteria: SearchCriteria) =>
    apiFetch<TorrentSearchResult>(`/search/torrents?${buildSearchQuery(criteria)}`),

  // Grab a release
  grab: (request: GrabRequest) =>
    apiFetch<GrabResult>('/search/grab', {
      method: 'POST',
      body: JSON.stringify(request),
    }),

  // Grab multiple releases
  grabBulk: (request: BulkGrabRequest) =>
    apiFetch<BulkGrabResult>('/search/grab/bulk', {
      method: 'POST',
      body: JSON.stringify(request),
    }),

  // Get grab history
  getGrabHistory: (limit = 50, offset = 0) =>
    apiFetch<GrabHistoryItem[]>(`/search/grab/history?limit=${limit}&offset=${offset}`),

  // Get indexer statuses
  getIndexerStatuses: () =>
    apiFetch<IndexerStatus[]>('/indexers/status'),
}
