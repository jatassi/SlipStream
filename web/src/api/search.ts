import type {
  BulkGrabRequest,
  BulkGrabResult,
  GrabHistoryItem,
  GrabRequest,
  GrabResult,
  IndexerStatus,
  ScoredSearchCriteria,
  SearchCriteria,
  SearchResult,
  TorrentSearchResult,
} from '@/types'

import { apiFetch } from './client'

function appendIfPresent(
  params: URLSearchParams,
  key: string,
  value: string | number | undefined,
): void {
  if (value !== undefined) {
    params.set(key, String(value))
  }
}

function buildSearchQuery(criteria: SearchCriteria): string {
  const params = new URLSearchParams()
  appendIfPresent(params, 'query', criteria.query)
  appendIfPresent(params, 'type', criteria.type)
  appendIfPresent(params, 'categories', criteria.categories)
  appendIfPresent(params, 'imdbId', criteria.imdbId)
  appendIfPresent(params, 'tmdbId', criteria.tmdbId)
  appendIfPresent(params, 'tvdbId', criteria.tvdbId)
  appendIfPresent(params, 'season', criteria.season)
  appendIfPresent(params, 'episode', criteria.episode)
  appendIfPresent(params, 'year', criteria.year)
  appendIfPresent(params, 'limit', criteria.limit)
  appendIfPresent(params, 'offset', criteria.offset)
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
    return apiFetch<TorrentSearchResult>(url)
      .then((result) => {
        console.log('[searchApi.searchMovie] Response JSON:', JSON.stringify(result, null, 2))
        return result
      })
      .catch((error: unknown) => {
        console.error('[searchApi.searchMovie] Error:', error)
        throw error
      })
  },

  // TV-specific search with scoring (returns scored TorrentInfo)
  searchTV: (criteria: ScoredSearchCriteria) => {
    const url = `/search/tv?${buildScoredSearchQuery(criteria)}`
    console.log('[searchApi.searchTV] Making request to:', url, 'criteria:', criteria)
    return apiFetch<TorrentSearchResult>(url)
      .then((result) => {
        console.log('[searchApi.searchTV] Response:', result)
        return result
      })
      .catch((error: unknown) => {
        console.error('[searchApi.searchTV] Error:', error)
        throw error
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
  getIndexerStatuses: () => apiFetch<IndexerStatus[]>('/indexers/status'),
}
