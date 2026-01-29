import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { moviesApi, libraryApi } from '@/api'
import type { Movie, CreateMovieInput, AddMovieInput, UpdateMovieInput, ListMoviesOptions } from '@/types'
import { calendarKeys } from './useCalendar'
import { missingKeys } from './useMissing'

export const movieKeys = {
  all: ['movies'] as const,
  lists: () => [...movieKeys.all, 'list'] as const,
  list: (filters: ListMoviesOptions) => [...movieKeys.lists(), filters] as const,
  details: () => [...movieKeys.all, 'detail'] as const,
  detail: (id: number) => [...movieKeys.details(), id] as const,
}

export function useMovies(options?: ListMoviesOptions) {
  return useQuery({
    queryKey: movieKeys.list(options || {}),
    queryFn: () => moviesApi.list(options),
  })
}

export function useMovie(id: number) {
  return useQuery({
    queryKey: movieKeys.detail(id),
    queryFn: () => moviesApi.get(id),
    enabled: !!id,
  })
}

export function useCreateMovie() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateMovieInput) => moviesApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: movieKeys.all })
      queryClient.invalidateQueries({ queryKey: calendarKeys.all })
    },
  })
}

export function useAddMovie() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: AddMovieInput) => libraryApi.addMovie(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: movieKeys.all })
      queryClient.invalidateQueries({ queryKey: calendarKeys.all })
    },
  })
}

export function useUpdateMovie() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateMovieInput }) =>
      moviesApi.update(id, data),
    onSuccess: (movie: Movie) => {
      queryClient.invalidateQueries({ queryKey: movieKeys.all })
      queryClient.invalidateQueries({ queryKey: missingKeys.all })
      queryClient.setQueryData(movieKeys.detail(movie.id), movie)
    },
  })
}

export function useDeleteMovie() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, deleteFiles }: { id: number; deleteFiles?: boolean }) =>
      moviesApi.delete(id, deleteFiles),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: movieKeys.all })
      queryClient.invalidateQueries({ queryKey: calendarKeys.all })
    },
  })
}

export function useBulkDeleteMovies() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ ids, deleteFiles }: { ids: number[]; deleteFiles?: boolean }) =>
      moviesApi.bulkDelete(ids, deleteFiles),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: movieKeys.all })
      queryClient.invalidateQueries({ queryKey: calendarKeys.all })
    },
  })
}

export function useBulkUpdateMovies() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ ids, data }: { ids: number[]; data: UpdateMovieInput }) =>
      moviesApi.bulkUpdate(ids, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: movieKeys.all })
      queryClient.invalidateQueries({ queryKey: missingKeys.all })
    },
  })
}

export function useScanMovie() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => moviesApi.scan(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: movieKeys.detail(id) })
    },
  })
}

export function useSearchMovie() {
  return useMutation({
    mutationFn: (id: number) => moviesApi.search(id),
  })
}

export function useRefreshMovie() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => moviesApi.refresh(id),
    onSuccess: (movie: Movie) => {
      queryClient.setQueryData(movieKeys.detail(movie.id), movie)
    },
  })
}
