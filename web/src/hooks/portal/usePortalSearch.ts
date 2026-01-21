import { useQuery } from '@tanstack/react-query'
import { portalSearchApi } from '@/api'

export const portalSearchKeys = {
  all: ['portalSearch'] as const,
  movies: (query: string) => [...portalSearchKeys.all, 'movies', query] as const,
  series: (query: string) => [...portalSearchKeys.all, 'series', query] as const,
  seasons: (tmdbId?: number, tvdbId?: number) => [...portalSearchKeys.all, 'seasons', tmdbId, tvdbId] as const,
}

export function usePortalMovieSearch(query: string) {
  return useQuery({
    queryKey: portalSearchKeys.movies(query),
    queryFn: () => portalSearchApi.searchMovies(query),
    enabled: query.length >= 2,
    staleTime: 5 * 60 * 1000, // 5 minutes
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  })
}

export function usePortalSeriesSearch(query: string) {
  return useQuery({
    queryKey: portalSearchKeys.series(query),
    queryFn: () => portalSearchApi.searchSeries(query),
    enabled: query.length >= 2,
    staleTime: 5 * 60 * 1000, // 5 minutes
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  })
}

export function useSeriesSeasons(tmdbId?: number, tvdbId?: number) {
  return useQuery({
    queryKey: portalSearchKeys.seasons(tmdbId, tvdbId),
    queryFn: () => portalSearchApi.getSeriesSeasons(tmdbId, tvdbId),
    enabled: (tmdbId !== undefined && tmdbId > 0) || (tvdbId !== undefined && tvdbId > 0),
    staleTime: 10 * 60 * 1000, // 10 minutes
  })
}
