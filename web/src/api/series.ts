import type {
  BulkEpisodeMonitorInput,
  BulkMonitorInput,
  CreateSeriesInput,
  Episode,
  ListSeriesOptions,
  MonitoringStats,
  Season,
  Series,
  UpdateEpisodeInput,
  UpdateSeriesInput,
} from '@/types'

import { apiFetch, buildQueryString } from './client'

export const seriesApi = {
  list: (options?: ListSeriesOptions) =>
    apiFetch<Series[]>(`/series${buildQueryString(options ?? {})}`),

  get: (id: number) => apiFetch<Series>(`/series/${id}`),

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

  delete: (id: number, deleteFiles?: boolean) =>
    apiFetch<undefined>(`/series/${id}${deleteFiles ? '?deleteFiles=true' : ''}`, { method: 'DELETE' }),

  bulkDelete: (ids: number[], deleteFiles?: boolean) =>
    Promise.all(
      ids.map((id) =>
        apiFetch<undefined>(`/series/${id}${deleteFiles ? '?deleteFiles=true' : ''}`, {
          method: 'DELETE',
        }),
      ),
    ),

  bulkUpdate: (ids: number[], data: UpdateSeriesInput) =>
    Promise.all(
      ids.map((id) =>
        apiFetch<Series>(`/series/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
      ),
    ),

  scan: (id: number) => apiFetch<undefined>(`/series/${id}/scan`, { method: 'POST' }),

  search: (id: number) => apiFetch<undefined>(`/series/${id}/search`, { method: 'POST' }),

  refresh: (id: number) => apiFetch<Series>(`/series/${id}/refresh`, { method: 'POST' }),

  refreshAll: () => apiFetch<{ message: string }>('/series/refresh', { method: 'POST' }),

  // Season operations
  getSeasons: (seriesId: number) => apiFetch<Season[]>(`/series/${seriesId}/seasons`),

  getSeason: (seriesId: number, seasonNumber: number) =>
    apiFetch<Season>(`/series/${seriesId}/seasons/${seasonNumber}`),

  updateSeasonMonitored: (seriesId: number, seasonNumber: number, monitored: boolean) =>
    apiFetch<Season>(`/series/${seriesId}/seasons/${seasonNumber}`, {
      method: 'PUT',
      body: JSON.stringify({ monitored }),
    }),

  // Episode operations
  getEpisodes: (seriesId: number, seasonNumber?: number) => {
    const path =
      seasonNumber === undefined
        ? `/series/${seriesId}/episodes`
        : `/series/${seriesId}/seasons/${seasonNumber}/episodes`
    return apiFetch<Episode[]>(path)
  },

  getEpisode: (seriesId: number, seasonNumber: number, episodeNumber: number) =>
    apiFetch<Episode>(`/series/${seriesId}/seasons/${seasonNumber}/episodes/${episodeNumber}`),

  updateEpisode: (params: {
    seriesId: number
    seasonNumber: number
    episodeNumber: number
    data: UpdateEpisodeInput
  }) =>
    apiFetch<Episode>(
      `/series/${params.seriesId}/seasons/${params.seasonNumber}/episodes/${params.episodeNumber}`,
      {
        method: 'PUT',
        body: JSON.stringify(params.data),
      },
    ),

  updateEpisodeById: (seriesId: number, episodeId: number, data: UpdateEpisodeInput) =>
    apiFetch<Episode>(`/series/${seriesId}/episodes/${episodeId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  searchEpisode: (seriesId: number, seasonNumber: number, episodeNumber: number) =>
    apiFetch<undefined>(`/series/${seriesId}/seasons/${seasonNumber}/episodes/${episodeNumber}/search`, {
      method: 'POST',
    }),

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
