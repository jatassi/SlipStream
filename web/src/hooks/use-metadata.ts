import { useQuery } from '@tanstack/react-query'

import { metadataApi } from '@/api'

export const metadataKeys = {
  all: ['metadata'] as const,
  movieSearch: (query: string) => [...metadataKeys.all, 'movie', 'search', query] as const,
  movie: (tmdbId: number) => [...metadataKeys.all, 'movie', tmdbId] as const,
  movieImages: (tmdbId: number) => [...metadataKeys.all, 'movie', tmdbId, 'images'] as const,
  movieExtended: (tmdbId: number) => [...metadataKeys.all, 'movie', tmdbId, 'extended'] as const,
  seriesSearch: (query: string) => [...metadataKeys.all, 'series', 'search', query] as const,
  series: (tmdbId: number) => [...metadataKeys.all, 'series', tmdbId] as const,
  seriesImages: (tmdbId: number) => [...metadataKeys.all, 'series', tmdbId, 'images'] as const,
  seriesExtended: (tmdbId: number) => [...metadataKeys.all, 'series', tmdbId, 'extended'] as const,
}

export function useMovieSearch(query: string) {
  return useQuery({
    queryKey: metadataKeys.movieSearch(query),
    queryFn: () => metadataApi.searchMovies(query),
    enabled: query.length >= 2,
    staleTime: 1000 * 60 * 5, // 5 minutes
  })
}

export function useMovieMetadata(tmdbId: number) {
  return useQuery({
    queryKey: metadataKeys.movie(tmdbId),
    queryFn: () => metadataApi.getMovie(tmdbId),
    enabled: !!tmdbId,
  })
}

export function useMovieImages(tmdbId: number) {
  return useQuery({
    queryKey: metadataKeys.movieImages(tmdbId),
    queryFn: () => metadataApi.getMovieImages(tmdbId),
    enabled: !!tmdbId,
  })
}

export function useSeriesSearch(query: string) {
  return useQuery({
    queryKey: metadataKeys.seriesSearch(query),
    queryFn: () => metadataApi.searchSeries(query),
    enabled: query.length >= 2,
    staleTime: 1000 * 60 * 5, // 5 minutes
  })
}

export function useSeriesMetadata(tmdbId: number) {
  return useQuery({
    queryKey: metadataKeys.series(tmdbId),
    queryFn: () => metadataApi.getSeries(tmdbId),
    enabled: !!tmdbId,
  })
}

export function useSeriesImages(tmdbId: number) {
  return useQuery({
    queryKey: metadataKeys.seriesImages(tmdbId),
    queryFn: () => metadataApi.getSeriesImages(tmdbId),
    enabled: !!tmdbId,
  })
}

export function useExtendedMovieMetadata(tmdbId: number) {
  return useQuery({
    queryKey: metadataKeys.movieExtended(tmdbId),
    queryFn: () => metadataApi.getExtendedMovie(tmdbId),
    enabled: !!tmdbId,
    staleTime: 1000 * 60 * 10, // 10 minutes
  })
}

export function useExtendedSeriesMetadata(tmdbId: number) {
  return useQuery({
    queryKey: metadataKeys.seriesExtended(tmdbId),
    queryFn: () => metadataApi.getExtendedSeries(tmdbId),
    enabled: !!tmdbId,
    staleTime: 1000 * 60 * 10, // 10 minutes
  })
}
