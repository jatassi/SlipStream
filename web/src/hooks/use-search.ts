import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { searchApi } from '@/api/search'
import type { BulkGrabRequest, GrabRequest, ScoredSearchCriteria, SearchCriteria } from '@/types'

// Query keys
export const searchKeys = {
  all: ['search'] as const,
  results: (criteria: SearchCriteria) => [...searchKeys.all, 'results', criteria] as const,
  movieResults: (criteria: ScoredSearchCriteria) => [...searchKeys.all, 'movie', criteria] as const,
  tvResults: (criteria: ScoredSearchCriteria) => [...searchKeys.all, 'tv', criteria] as const,
  torrentResults: (criteria: ScoredSearchCriteria) =>
    [...searchKeys.all, 'torrents', criteria] as const,
  grabHistory: (limit?: number, offset?: number) =>
    [...searchKeys.all, 'history', { limit, offset }] as const,
  indexerStatuses: () => [...searchKeys.all, 'statuses'] as const,
}

// General search hook (basic ReleaseInfo, no scoring)
export function useSearch(criteria: SearchCriteria, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: searchKeys.results(criteria),
    queryFn: () => searchApi.search(criteria),
    enabled:
      options?.enabled ??
      (!!criteria.query || !!criteria.tmdbId || !!criteria.tvdbId || !!criteria.imdbId),
    staleTime: 30_000, // 30 seconds
  })
}

// Movie search hook with scoring (searches indexers for movie releases)
export function useIndexerMovieSearch(
  criteria: ScoredSearchCriteria,
  options?: { enabled?: boolean },
) {
  const defaultEnabled =
    !!criteria.qualityProfileId && (!!criteria.query || !!criteria.tmdbId || !!criteria.imdbId)
  const finalEnabled = options?.enabled ?? defaultEnabled

  console.log(
    '[useIndexerMovieSearch] criteria:',
    criteria,
    'options.enabled:',
    options?.enabled,
    'defaultEnabled:',
    defaultEnabled,
    'finalEnabled:',
    finalEnabled,
  )

  return useQuery({
    queryKey: searchKeys.movieResults(criteria),
    queryFn: () => {
      console.log('[useIndexerMovieSearch] queryFn executing!')
      return searchApi.searchMovie(criteria)
    },
    enabled: finalEnabled,
    staleTime: 30_000,
  })
}

// TV search hook with scoring (searches indexers for TV releases)
export function useIndexerTVSearch(
  criteria: ScoredSearchCriteria,
  options?: { enabled?: boolean },
) {
  return useQuery({
    queryKey: searchKeys.tvResults(criteria),
    queryFn: () => searchApi.searchTV(criteria),
    enabled:
      options?.enabled ?? (!!criteria.qualityProfileId && (!!criteria.query || !!criteria.tvdbId)),
    staleTime: 30_000,
  })
}

// Torrent search hook with scoring
export function useSearchTorrents(criteria: ScoredSearchCriteria, options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: searchKeys.torrentResults(criteria),
    queryFn: () => searchApi.searchTorrents(criteria),
    enabled:
      options?.enabled ??
      (!!criteria.qualityProfileId &&
        (!!criteria.query || !!criteria.tmdbId || !!criteria.tvdbId || !!criteria.imdbId)),
    staleTime: 30_000,
  })
}

// Grab a release mutation
export function useGrab() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: GrabRequest) => searchApi.grab(request),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['queue'] })
      void queryClient.invalidateQueries({ queryKey: searchKeys.grabHistory() })
    },
  })
}

// Bulk grab mutation
export function useGrabBulk() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: BulkGrabRequest) => searchApi.grabBulk(request),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['queue'] })
      void queryClient.invalidateQueries({ queryKey: searchKeys.grabHistory() })
    },
  })
}

// Grab history hook
export function useGrabHistory(limit = 50, offset = 0) {
  return useQuery({
    queryKey: searchKeys.grabHistory(limit, offset),
    queryFn: () => searchApi.getGrabHistory(limit, offset),
  })
}

// Note: Indexer status hooks are now in useIndexers.ts
