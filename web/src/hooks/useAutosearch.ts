import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { autosearchApi } from '@/api'
import { queueKeys } from './useQueue'
import type { AutoSearchMediaType, AutoSearchSettings, AutoSearchResult, BatchAutoSearchResult } from '@/types'

export const autosearchKeys = {
  all: ['autosearch'] as const,
  status: (mediaType: AutoSearchMediaType, mediaId: number) =>
    [...autosearchKeys.all, 'status', mediaType, mediaId] as const,
  settings: () => [...autosearchKeys.all, 'settings'] as const,
}

export function useAutoSearchMovie() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (movieId: number) => autosearchApi.searchMovie(movieId),
    onSuccess: (result: AutoSearchResult) => {
      if (result.downloaded) {
        queryClient.invalidateQueries({ queryKey: queueKeys.all })
      }
    },
  })
}

export function useAutoSearchEpisode() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (episodeId: number) => autosearchApi.searchEpisode(episodeId),
    onSuccess: (result: AutoSearchResult) => {
      if (result.downloaded) {
        queryClient.invalidateQueries({ queryKey: queueKeys.all })
      }
    },
  })
}

export function useAutoSearchSeason() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ seriesId, seasonNumber }: { seriesId: number; seasonNumber: number }) =>
      autosearchApi.searchSeason(seriesId, seasonNumber),
    onSuccess: (result: BatchAutoSearchResult) => {
      if (result.downloaded > 0) {
        queryClient.invalidateQueries({ queryKey: queueKeys.all })
      }
    },
  })
}

export function useAutoSearchSeries() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (seriesId: number) => autosearchApi.searchSeries(seriesId),
    onSuccess: (result: BatchAutoSearchResult) => {
      if (result.downloaded > 0) {
        queryClient.invalidateQueries({ queryKey: queueKeys.all })
      }
    },
  })
}

export function useAutoSearchStatus(mediaType: AutoSearchMediaType, mediaId: number) {
  return useQuery({
    queryKey: autosearchKeys.status(mediaType, mediaId),
    queryFn: () => autosearchApi.getStatus(mediaType, mediaId),
    enabled: !!mediaId,
    staleTime: 5000,
  })
}

export function useAutoSearchSettings() {
  return useQuery({
    queryKey: autosearchKeys.settings(),
    queryFn: () => autosearchApi.getSettings(),
  })
}

export function useUpdateAutoSearchSettings() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (settings: AutoSearchSettings) => autosearchApi.updateSettings(settings),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: autosearchKeys.settings() })
    },
  })
}

export function useSearchAllMissing() {
  return useMutation({
    mutationFn: () => autosearchApi.searchAllMissing(),
  })
}

export function useSearchAllMissingMovies() {
  return useMutation({
    mutationFn: () => autosearchApi.searchAllMissingMovies(),
  })
}

export function useSearchAllMissingSeries() {
  return useMutation({
    mutationFn: () => autosearchApi.searchAllMissingSeries(),
  })
}
