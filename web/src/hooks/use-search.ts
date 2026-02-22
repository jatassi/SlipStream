import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { searchApi } from '@/api/search'
import type { GrabRequest, ScoredSearchCriteria } from '@/types'

// Query keys
const searchKeys = {
  all: ['search'] as const,
  movieResults: (criteria: ScoredSearchCriteria) => [...searchKeys.all, 'movie', criteria] as const,
  tvResults: (criteria: ScoredSearchCriteria) => [...searchKeys.all, 'tv', criteria] as const,
  grabHistory: (limit?: number, offset?: number) =>
    [...searchKeys.all, 'history', { limit, offset }] as const,
}

// Movie search hook with scoring (searches indexers for movie releases)
export function useIndexerMovieSearch(
  criteria: ScoredSearchCriteria,
  options?: { enabled?: boolean },
) {
  const defaultEnabled =
    !!criteria.qualityProfileId && (!!criteria.query || !!criteria.tmdbId || !!criteria.imdbId)

  return useQuery({
    queryKey: searchKeys.movieResults(criteria),
    queryFn: () => searchApi.searchMovie(criteria),
    enabled: options?.enabled ?? defaultEnabled,
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

// Note: Indexer status hooks are now in useIndexers.ts
