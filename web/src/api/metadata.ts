import { apiFetch } from './client'
import type { MovieSearchResult, SeriesSearchResult, MetadataImages } from '@/types'

export const metadataApi = {
  searchMovies: (query: string) =>
    apiFetch<MovieSearchResult[]>(`/metadata/movie/search?query=${encodeURIComponent(query)}`),

  getMovie: (tmdbId: number) =>
    apiFetch<MovieSearchResult>(`/metadata/movie/${tmdbId}`),

  getMovieImages: (tmdbId: number) =>
    apiFetch<MetadataImages>(`/metadata/movie/${tmdbId}/images`),

  searchSeries: (query: string) =>
    apiFetch<SeriesSearchResult[]>(`/metadata/series/search?query=${encodeURIComponent(query)}`),

  getSeries: (tmdbId: number) =>
    apiFetch<SeriesSearchResult>(`/metadata/series/${tmdbId}`),

  getSeriesImages: (tmdbId: number) =>
    apiFetch<MetadataImages>(`/metadata/series/${tmdbId}/images`),
}
