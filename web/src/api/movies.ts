import type { CreateMovieInput, ListMoviesOptions, Movie, UpdateMovieInput } from '@/types'

import { apiFetch, buildQueryString } from './client'

export const moviesApi = {
  list: (options?: ListMoviesOptions) =>
    apiFetch<Movie[]>(`/movies${buildQueryString(options ?? {})}`),

  get: (id: number) => apiFetch<Movie>(`/movies/${id}`),

  create: (data: CreateMovieInput) =>
    apiFetch<Movie>('/movies', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  update: (id: number, data: UpdateMovieInput) =>
    apiFetch<Movie>(`/movies/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  delete: (id: number, deleteFiles?: boolean) =>
    apiFetch<undefined>(`/movies/${id}${deleteFiles ? '?deleteFiles=true' : ''}`, { method: 'DELETE' }),

  bulkDelete: (ids: number[], deleteFiles?: boolean) =>
    Promise.all(
      ids.map((id) =>
        apiFetch<undefined>(`/movies/${id}${deleteFiles ? '?deleteFiles=true' : ''}`, {
          method: 'DELETE',
        }),
      ),
    ),

  bulkUpdate: (ids: number[], data: UpdateMovieInput) =>
    Promise.all(
      ids.map((id) =>
        apiFetch<Movie>(`/movies/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
      ),
    ),

  bulkMonitor: (ids: number[], monitored: boolean) =>
    apiFetch<{ status: string }>('/movies/monitor', {
      method: 'PUT',
      body: JSON.stringify({ ids, monitored }),
    }),

  scan: (id: number) => apiFetch<undefined>(`/movies/${id}/scan`, { method: 'POST' }),

  search: (id: number) => apiFetch<undefined>(`/movies/${id}/search`, { method: 'POST' }),

  refresh: (id: number) => apiFetch<Movie>(`/movies/${id}/refresh`, { method: 'POST' }),

  refreshAll: () => apiFetch<{ message: string }>('/movies/refresh', { method: 'POST' }),
}
