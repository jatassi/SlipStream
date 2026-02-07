import { apiFetch } from './client'
import type {
  MovieSearchResult,
  SeriesSearchResult,
  MetadataImages,
  ExtendedMovieResult,
  ExtendedSeriesResult,
} from '@/types'

// Backend response types (before transformation)
interface BackendMovieResult {
  id: number
  title: string
  year?: number
  overview?: string
  posterUrl?: string
  backdropUrl?: string
  imdbId?: string
  genres?: string[]
  runtime?: number
  studio?: string
}

interface BackendSeriesResult {
  id: number
  title: string
  year?: number
  overview?: string
  posterUrl?: string
  backdropUrl?: string
  imdbId?: string
  tvdbId?: number
  tmdbId?: number
  genres?: string[]
  status?: string
  runtime?: number
  network?: string
  networkLogoUrl?: string
}

// Transform backend movie result to frontend format
function transformMovieResult(result: BackendMovieResult): MovieSearchResult {
  return {
    ...result,
    tmdbId: result.id,
  }
}

// Transform backend series result to frontend format
function transformSeriesResult(result: BackendSeriesResult): SeriesSearchResult {
  return {
    ...result,
    tmdbId: result.tmdbId || result.id,
  }
}

export const metadataApi = {
  searchMovies: async (query: string) => {
    const results = await apiFetch<BackendMovieResult[]>(`/metadata/movie/search?query=${encodeURIComponent(query)}`)
    return results.map(transformMovieResult)
  },

  getMovie: async (tmdbId: number) => {
    const result = await apiFetch<BackendMovieResult>(`/metadata/movie/${tmdbId}`)
    return transformMovieResult(result)
  },

  getMovieImages: (tmdbId: number) =>
    apiFetch<MetadataImages>(`/metadata/movie/${tmdbId}/images`),

  searchSeries: async (query: string) => {
    const results = await apiFetch<BackendSeriesResult[]>(`/metadata/series/search?query=${encodeURIComponent(query)}`)
    return results.map(transformSeriesResult)
  },

  getSeries: async (tmdbId: number) => {
    const result = await apiFetch<BackendSeriesResult>(`/metadata/series/${tmdbId}`)
    return transformSeriesResult(result)
  },

  getSeriesImages: (tmdbId: number) =>
    apiFetch<MetadataImages>(`/metadata/series/${tmdbId}/images`),

  getExtendedMovie: (tmdbId: number) =>
    apiFetch<ExtendedMovieResult>(`/metadata/movie/${tmdbId}/extended`),

  getExtendedSeries: (tmdbId: number) =>
    apiFetch<ExtendedSeriesResult>(`/metadata/series/${tmdbId}/extended`),
}
