import { useQuery } from '@tanstack/react-query'

import { metadataApi } from '@/api'

const metadataKeys = {
  all: ['metadata'] as const,
  movieSearch: (query: string) => [...metadataKeys.all, 'movie', 'search', query] as const,
  movie: (tmdbId: number) => [...metadataKeys.all, 'movie', tmdbId] as const,
  movieExtended: (tmdbId: number) => [...metadataKeys.all, 'movie', tmdbId, 'extended'] as const,
  seriesSearch: (query: string) => [...metadataKeys.all, 'series', 'search', query] as const,
  series: (tmdbId: number) => [...metadataKeys.all, 'series', tmdbId] as const,
  seriesExtended: (tmdbId: number) => [...metadataKeys.all, 'series', tmdbId, 'extended'] as const,
}

export function useMovieSearch(query: string) {
  return useQuery({
    queryKey: metadataKeys.movieSearch(query),
    queryFn: ({ signal }) => metadataApi.searchMovies(query, { signal }),
    enabled: query.length >= 2,
    staleTime: 1000 * 60 * 5, // 5 minutes
  })
}

export function useMovieMetadata(tmdbId: number) {
  return useQuery({
    queryKey: metadataKeys.movie(tmdbId),
    queryFn: ({ signal }) => metadataApi.getMovie(tmdbId, { signal }),
    enabled: !!tmdbId,
  })
}

export function useSeriesSearch(query: string) {
  return useQuery({
    queryKey: metadataKeys.seriesSearch(query),
    queryFn: ({ signal }) => metadataApi.searchSeries(query, { signal }),
    enabled: query.length >= 2,
    staleTime: 1000 * 60 * 5, // 5 minutes
  })
}

export function useSeriesMetadata(tmdbId: number) {
  return useQuery({
    queryKey: metadataKeys.series(tmdbId),
    queryFn: ({ signal }) => metadataApi.getSeries(tmdbId, { signal }),
    enabled: !!tmdbId,
  })
}

export const extendedMovieMetadataOptions = (tmdbId: number) => ({
  queryKey: metadataKeys.movieExtended(tmdbId),
  queryFn: ({ signal }: { signal: AbortSignal }) => metadataApi.getExtendedMovie(tmdbId, { signal }),
  enabled: !!tmdbId,
  staleTime: 1000 * 60 * 10, // 10 minutes
})

export function useExtendedMovieMetadata(tmdbId: number) {
  return useQuery(extendedMovieMetadataOptions(tmdbId))
}

export const extendedSeriesMetadataOptions = (tmdbId: number) => ({
  queryKey: metadataKeys.seriesExtended(tmdbId),
  queryFn: ({ signal }: { signal: AbortSignal }) => metadataApi.getExtendedSeries(tmdbId, { signal }),
  enabled: !!tmdbId,
  staleTime: 1000 * 60 * 10, // 10 minutes
})

export function useExtendedSeriesMetadata(tmdbId: number) {
  return useQuery(extendedSeriesMetadataOptions(tmdbId))
}
