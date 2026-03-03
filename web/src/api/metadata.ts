import type {
  ExtendedMovieResult,
  ExtendedSeriesResult,
  MetadataImages,
  MovieSearchResult,
  SeriesSearchResult,
} from '@/types'

import { apiFetch } from './client'

// Backend response types (before transformation)
type BackendMovieResult = {
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

type BackendSeriesResult = {
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
    tmdbId: result.tmdbId ?? result.id,
  }
}

export const metadataApi = {
  searchMovies: async (query: string, init?: RequestInit) => {
    const results = await apiFetch<BackendMovieResult[]>(
      `/metadata/movie/search?query=${encodeURIComponent(query)}`,
      init,
    )
    return results.map((r) => transformMovieResult(r))
  },

  getMovie: async (tmdbId: number, init?: RequestInit) => {
    const result = await apiFetch<BackendMovieResult>(`/metadata/movie/${tmdbId}`, init)
    return transformMovieResult(result)
  },

  getMovieImages: (tmdbId: number, init?: RequestInit) =>
    apiFetch<MetadataImages>(`/metadata/movie/${tmdbId}/images`, init),

  searchSeries: async (query: string, init?: RequestInit) => {
    const results = await apiFetch<BackendSeriesResult[]>(
      `/metadata/series/search?query=${encodeURIComponent(query)}`,
      init,
    )
    return results.map((r) => transformSeriesResult(r))
  },

  getSeries: async (tmdbId: number, init?: RequestInit) => {
    const result = await apiFetch<BackendSeriesResult>(`/metadata/series/${tmdbId}`, init)
    return transformSeriesResult(result)
  },

  getSeriesImages: (tmdbId: number, init?: RequestInit) =>
    apiFetch<MetadataImages>(`/metadata/series/${tmdbId}/images`, init),

  getExtendedMovie: (tmdbId: number, init?: RequestInit) =>
    apiFetch<ExtendedMovieResult>(`/metadata/movie/${tmdbId}/extended`, init),

  getExtendedSeries: (tmdbId: number, init?: RequestInit) =>
    apiFetch<ExtendedSeriesResult>(`/metadata/series/${tmdbId}/extended`, init),
}
