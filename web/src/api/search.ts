import { apiFetch } from './client'
import type {
  SearchCriteria,
  ScoredSearchCriteria,
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

// Build query string for scored search criteria (includes qualityProfileId)
function buildScoredSearchQuery(criteria: ScoredSearchCriteria): string {
  const baseQuery = buildSearchQuery(criteria)
  const params = new URLSearchParams(baseQuery)
  params.set('qualityProfileId', String(criteria.qualityProfileId))
  return params.toString()
}

export const searchApi = {
  // General search (basic ReleaseInfo, no scoring)
  search: (criteria: SearchCriteria) =>
    apiFetch<SearchResult>(`/search?${buildSearchQuery(criteria)}`),

  // Movie-specific search with scoring (returns scored TorrentInfo)
  searchMovie: (criteria: ScoredSearchCriteria) => {
    const url = `/search/movie?${buildScoredSearchQuery(criteria)}`
    console.log('[searchApi.searchMovie] Making request to:', url, 'criteria:', criteria)
    return apiFetch<TorrentSearchResult>(url).then(result => {
      console.log('[searchApi.searchMovie] Response JSON:', JSON.stringify(result, null, 2))
      return result
    }).catch(err => {
      console.error('[searchApi.searchMovie] Error:', err)
      throw err
    })
  },

  // TV-specific search with scoring (returns scored TorrentInfo)
  searchTV: (criteria: ScoredSearchCriteria) => {
    const url = `/search/tv?${buildScoredSearchQuery(criteria)}`
    console.log('[searchApi.searchTV] Making request to:', url, 'criteria:', criteria)
    return apiFetch<TorrentSearchResult>(url).then(result => {
      console.log('[searchApi.searchTV] Response:', result)
      return result
    }).catch(err => {
      console.error('[searchApi.searchTV] Error:', err)
      throw err
    })
  },

  // Torrent search with scoring (returns scored TorrentInfo)
  searchTorrents: (criteria: ScoredSearchCriteria) =>
    apiFetch<TorrentSearchResult>(`/search/torrents?${buildScoredSearchQuery(criteria)}`),

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
