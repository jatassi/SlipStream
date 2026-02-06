import { apiFetch, buildQueryString } from './client'
import type { Movie, CreateMovieInput, UpdateMovieInput, ListMoviesOptions } from '@/types'

export const moviesApi = {
  list: (options?: ListMoviesOptions) =>
    apiFetch<Movie[]>(`/movies${buildQueryString(options || {})}`),

  get: (id: number) =>
    apiFetch<Movie>(`/movies/${id}`),

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
    apiFetch<void>(`/movies/${id}${deleteFiles ? '?deleteFiles=true' : ''}`, { method: 'DELETE' }),

  bulkDelete: (ids: number[], deleteFiles?: boolean) =>
    Promise.all(ids.map(id => apiFetch<void>(`/movies/${id}${deleteFiles ? '?deleteFiles=true' : ''}`, { method: 'DELETE' }))),

  bulkUpdate: (ids: number[], data: UpdateMovieInput) =>
    Promise.all(ids.map(id => apiFetch<Movie>(`/movies/${id}`, { method: 'PUT', body: JSON.stringify(data) }))),

  scan: (id: number) =>
    apiFetch<void>(`/movies/${id}/scan`, { method: 'POST' }),

  search: (id: number) =>
    apiFetch<void>(`/movies/${id}/search`, { method: 'POST' }),

  refresh: (id: number) =>
    apiFetch<Movie>(`/movies/${id}/refresh`, { method: 'POST' }),

  refreshAll: () =>
    apiFetch<{ message: string }>('/movies/refresh', { method: 'POST' }),
}
