import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { autosearchApi } from '@/api'
import type {
  AutoSearchResult,
  AutoSearchSettings,
  BatchAutoSearchResult,
  SlotSearchResult,
} from '@/types'

import { queueKeys } from './use-queue'

const autosearchKeys = {
  all: ['autosearch'] as const,
  settings: () => [...autosearchKeys.all, 'settings'] as const,
}

export function useAutoSearchMovie() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (movieId: number) => autosearchApi.searchMovie(movieId),
    onSuccess: (result: AutoSearchResult) => {
      if (result.downloaded) {
        void queryClient.invalidateQueries({ queryKey: queueKeys.all })
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
        void queryClient.invalidateQueries({ queryKey: queueKeys.all })
      }
    },
  })
}

export function useAutoSearchMovieSlot() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ movieId, slotId }: { movieId: number; slotId: number }) =>
      autosearchApi.searchMovieSlot(movieId, slotId),
    onSuccess: (result: SlotSearchResult) => {
      if (result.downloaded) {
        void queryClient.invalidateQueries({ queryKey: queueKeys.all })
      }
    },
  })
}

export function useAutoSearchEpisodeSlot() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ episodeId, slotId }: { episodeId: number; slotId: number }) =>
      autosearchApi.searchEpisodeSlot(episodeId, slotId),
    onSuccess: (result: SlotSearchResult) => {
      if (result.downloaded) {
        void queryClient.invalidateQueries({ queryKey: queueKeys.all })
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
        void queryClient.invalidateQueries({ queryKey: queueKeys.all })
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
        void queryClient.invalidateQueries({ queryKey: queueKeys.all })
      }
    },
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
      void queryClient.invalidateQueries({ queryKey: autosearchKeys.settings() })
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

export function useSearchAllUpgradable() {
  return useMutation({
    mutationFn: () => autosearchApi.searchAllUpgradable(),
  })
}

export function useSearchAllUpgradableMovies() {
  return useMutation({
    mutationFn: () => autosearchApi.searchAllUpgradableMovies(),
  })
}

export function useSearchAllUpgradableSeries() {
  return useMutation({
    mutationFn: () => autosearchApi.searchAllUpgradableSeries(),
  })
}
