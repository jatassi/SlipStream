import { apiFetch } from './client'
import type { AutoSearchResult, BatchAutoSearchResult, AutoSearchStatus, AutoSearchMediaType, AutoSearchSettings } from '@/types'

export interface BulkSearchStartedResponse {
  message: string
}

export const autosearchApi = {
  searchMovie: (movieId: number) =>
    apiFetch<AutoSearchResult>(`/autosearch/movie/${movieId}`, { method: 'POST' }),

  searchEpisode: (episodeId: number) =>
    apiFetch<AutoSearchResult>(`/autosearch/episode/${episodeId}`, { method: 'POST' }),

  searchSeason: (seriesId: number, seasonNumber: number) =>
    apiFetch<BatchAutoSearchResult>(`/autosearch/season/${seriesId}/${seasonNumber}`, { method: 'POST' }),

  searchSeries: (seriesId: number) =>
    apiFetch<BatchAutoSearchResult>(`/autosearch/series/${seriesId}`, { method: 'POST' }),

  getStatus: (mediaType: AutoSearchMediaType, mediaId: number) =>
    apiFetch<AutoSearchStatus>(`/autosearch/status/${mediaType}/${mediaId}`),

  getSettings: () =>
    apiFetch<AutoSearchSettings>('/settings/autosearch'),

  updateSettings: (settings: AutoSearchSettings) =>
    apiFetch<AutoSearchSettings>('/settings/autosearch', {
      method: 'PUT',
      body: JSON.stringify(settings),
    }),

  // Bulk search operations
  searchAllMissing: () =>
    apiFetch<BulkSearchStartedResponse>('/autosearch/missing/all', { method: 'POST' }),

  searchAllMissingMovies: () =>
    apiFetch<BulkSearchStartedResponse>('/autosearch/missing/movies', { method: 'POST' }),

  searchAllMissingSeries: () =>
    apiFetch<BulkSearchStartedResponse>('/autosearch/missing/series', { method: 'POST' }),
}
