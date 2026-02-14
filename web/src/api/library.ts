import type { AddMovieInput, AddSeriesInput, Movie, Series } from '@/types'

import { apiFetch } from './client'

export type ScanResponse = {
  message: string
  rootFolderIds: number[]
}

export type ScanStatus = {
  rootFolderId: number
  active: boolean
  activityId?: string
  activity?: {
    id: string
    type: string
    title: string
    status: string
    progress: number
    message: string
  }
}

export const libraryApi = {
  /** Trigger a scan of all root folders */
  scanAll: () => apiFetch<ScanResponse>('/scans', { method: 'POST' }),

  /** Get all active scan statuses */
  getScanStatuses: () => apiFetch<ScanStatus[]>('/scans'),

  /** Trigger a scan of a specific root folder */
  scanRootFolder: (id: number) =>
    apiFetch<{ message: string; rootFolderId: number }>(`/rootfolders/${id}/scan`, {
      method: 'POST',
    }),

  /** Get scan status for a specific root folder */
  getScanStatus: (id: number) => apiFetch<ScanStatus>(`/rootfolders/${id}/scan`),

  /** Cancel a scan for a specific root folder */
  cancelScan: (id: number) =>
    apiFetch<{ message: string; rootFolderId: number }>(`/rootfolders/${id}/scan`, {
      method: 'DELETE',
    }),

  /** Add a movie with artwork download */
  addMovie: (data: AddMovieInput) =>
    apiFetch<Movie>('/library/movies', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  /** Add a series with artwork download */
  addSeries: (data: AddSeriesInput) =>
    apiFetch<Series>('/library/series', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
}
