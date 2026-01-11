import { apiFetch, buildQueryString } from './client'
import type {
  Series,
  Season,
  Episode,
  CreateSeriesInput,
  UpdateSeriesInput,
  UpdateEpisodeInput,
  ListSeriesOptions,
  BulkMonitorInput,
  BulkEpisodeMonitorInput,
  MonitoringStats,
} from '@/types'

export const seriesApi = {
  list: (options?: ListSeriesOptions) =>
    apiFetch<Series[]>(`/series${buildQueryString(options || {})}`),

  get: (id: number) =>
    apiFetch<Series>(`/series/${id}`),

  create: (data: CreateSeriesInput) =>
    apiFetch<Series>('/series', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  update: (id: number, data: UpdateSeriesInput) =>
    apiFetch<Series>(`/series/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  delete: (id: number) =>
    apiFetch<void>(`/series/${id}`, { method: 'DELETE' }),

  scan: (id: number) =>
    apiFetch<void>(`/series/${id}/scan`, { method: 'POST' }),

  search: (id: number) =>
    apiFetch<void>(`/series/${id}/search`, { method: 'POST' }),

  refresh: (id: number) =>
    apiFetch<Series>(`/series/${id}/refresh`, { method: 'POST' }),

  // Season operations
  getSeasons: (seriesId: number) =>
    apiFetch<Season[]>(`/series/${seriesId}/seasons`),

  getSeason: (seriesId: number, seasonNumber: number) =>
    apiFetch<Season>(`/series/${seriesId}/seasons/${seasonNumber}`),

  updateSeasonMonitored: (seriesId: number, seasonNumber: number, monitored: boolean) =>
    apiFetch<Season>(`/series/${seriesId}/seasons/${seasonNumber}`, {
      method: 'PUT',
      body: JSON.stringify({ monitored }),
    }),

  // Episode operations
  getEpisodes: (seriesId: number, seasonNumber?: number) => {
    const path = seasonNumber !== undefined
      ? `/series/${seriesId}/seasons/${seasonNumber}/episodes`
      : `/series/${seriesId}/episodes`
    return apiFetch<Episode[]>(path)
  },

  getEpisode: (seriesId: number, seasonNumber: number, episodeNumber: number) =>
    apiFetch<Episode>(`/series/${seriesId}/seasons/${seasonNumber}/episodes/${episodeNumber}`),

  updateEpisode: (
    seriesId: number,
    seasonNumber: number,
    episodeNumber: number,
    data: UpdateEpisodeInput
  ) =>
    apiFetch<Episode>(
      `/series/${seriesId}/seasons/${seasonNumber}/episodes/${episodeNumber}`,
      {
        method: 'PUT',
        body: JSON.stringify(data),
      }
    ),

  updateEpisodeById: (seriesId: number, episodeId: number, data: UpdateEpisodeInput) =>
    apiFetch<Episode>(`/series/${seriesId}/episodes/${episodeId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  searchEpisode: (seriesId: number, seasonNumber: number, episodeNumber: number) =>
    apiFetch<void>(
      `/series/${seriesId}/seasons/${seasonNumber}/episodes/${episodeNumber}/search`,
      { method: 'POST' }
    ),

  // Bulk monitoring operations
  bulkMonitor: (seriesId: number, data: BulkMonitorInput) =>
    apiFetch<{ status: string }>(`/series/${seriesId}/monitor`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  bulkMonitorEpisodes: (seriesId: number, data: BulkEpisodeMonitorInput) =>
    apiFetch<{ status: string }>(`/series/${seriesId}/episodes/monitor`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  getMonitoringStats: (seriesId: number) =>
    apiFetch<MonitoringStats>(`/series/${seriesId}/monitor/stats`),
}
