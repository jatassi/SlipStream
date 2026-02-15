import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { libraryApi, seriesApi } from '@/api'
import type {
  AddSeriesInput,
  BulkEpisodeMonitorInput,
  BulkMonitorInput,
  CreateSeriesInput,
  ListSeriesOptions,
  Series,
  UpdateEpisodeInput,
  UpdateSeriesInput,
} from '@/types'

import { calendarKeys } from './use-calendar'
import { missingKeys } from './use-missing'

export const seriesKeys = {
  all: ['series'] as const,
  lists: () => [...seriesKeys.all, 'list'] as const,
  list: (filters: ListSeriesOptions) => [...seriesKeys.lists(), filters] as const,
  details: () => [...seriesKeys.all, 'detail'] as const,
  detail: (id: number) => [...seriesKeys.details(), id] as const,
  seasons: (seriesId: number) => [...seriesKeys.detail(seriesId), 'seasons'] as const,
  episodes: (seriesId: number, seasonNumber?: number) =>
    [...seriesKeys.detail(seriesId), 'episodes', seasonNumber] as const,
}

export function useSeries(options?: ListSeriesOptions) {
  return useQuery({
    queryKey: seriesKeys.list(options ?? {}),
    queryFn: () => seriesApi.list(options),
  })
}

export function useSeriesDetail(id: number) {
  return useQuery({
    queryKey: seriesKeys.detail(id),
    queryFn: () => seriesApi.get(id),
    enabled: !!id,
  })
}

export function useCreateSeries() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateSeriesInput) => seriesApi.create(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: seriesKeys.all })
      void queryClient.invalidateQueries({ queryKey: calendarKeys.all })
    },
  })
}

export function useAddSeries() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: AddSeriesInput) => libraryApi.addSeries(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: seriesKeys.all })
      void queryClient.invalidateQueries({ queryKey: calendarKeys.all })
    },
  })
}

export function useUpdateSeries() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateSeriesInput }) =>
      seriesApi.update(id, data),
    onSuccess: (series: Series) => {
      void queryClient.invalidateQueries({ queryKey: seriesKeys.all })
      void queryClient.invalidateQueries({ queryKey: missingKeys.all })
      void queryClient.setQueryData(seriesKeys.detail(series.id), series)
    },
  })
}

export function useDeleteSeries() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, deleteFiles }: { id: number; deleteFiles?: boolean }) =>
      seriesApi.delete(id, deleteFiles),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: seriesKeys.all })
      void queryClient.invalidateQueries({ queryKey: calendarKeys.all })
    },
  })
}

export function useBulkDeleteSeries() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ ids, deleteFiles }: { ids: number[]; deleteFiles?: boolean }) =>
      seriesApi.bulkDelete(ids, deleteFiles),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: seriesKeys.all })
      void queryClient.invalidateQueries({ queryKey: calendarKeys.all })
    },
  })
}

export function useBulkUpdateSeries() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ ids, data }: { ids: number[]; data: UpdateSeriesInput }) =>
      seriesApi.bulkUpdate(ids, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: seriesKeys.all })
      void queryClient.invalidateQueries({ queryKey: missingKeys.all })
    },
  })
}

export function useScanSeries() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => seriesApi.scan(id),
    onSuccess: (_, id) => {
      void queryClient.invalidateQueries({ queryKey: seriesKeys.detail(id) })
    },
  })
}

export function useSearchSeries() {
  return useMutation({
    mutationFn: (id: number) => seriesApi.search(id),
  })
}

export function useRefreshSeries() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => seriesApi.refresh(id),
    onSuccess: (series: Series) => {
      void queryClient.setQueryData(seriesKeys.detail(series.id), series)
      void queryClient.invalidateQueries({ queryKey: seriesKeys.seasons(series.id) })
      void queryClient.invalidateQueries({
        queryKey: [...seriesKeys.detail(series.id), 'episodes'],
      })
    },
  })
}

export function useRefreshAllSeries() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => seriesApi.refreshAll(),
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: seriesKeys.all })
    },
  })
}

export function useSeasons(seriesId: number) {
  return useQuery({
    queryKey: seriesKeys.seasons(seriesId),
    queryFn: () => seriesApi.getSeasons(seriesId),
    enabled: !!seriesId,
  })
}

export function useEpisodes(seriesId: number, seasonNumber?: number) {
  return useQuery({
    queryKey: seriesKeys.episodes(seriesId, seasonNumber),
    queryFn: () => seriesApi.getEpisodes(seriesId, seasonNumber),
    enabled: !!seriesId,
  })
}

export function useUpdateSeasonMonitored() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      seriesId,
      seasonNumber,
      monitored,
    }: {
      seriesId: number
      seasonNumber: number
      monitored: boolean
    }) => seriesApi.updateSeasonMonitored(seriesId, seasonNumber, monitored),
    onSuccess: (_, { seriesId }) => {
      void queryClient.invalidateQueries({ queryKey: seriesKeys.detail(seriesId) })
      void queryClient.invalidateQueries({ queryKey: missingKeys.all })
    },
  })
}

export function useUpdateEpisode() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      seriesId,
      seasonNumber,
      episodeNumber,
      data,
    }: {
      seriesId: number
      seasonNumber: number
      episodeNumber: number
      data: UpdateEpisodeInput
    }) =>
      seriesApi.updateEpisode({ seriesId, seasonNumber, episodeNumber, data }),
    onSuccess: (_, { seriesId }) => {
      void queryClient.invalidateQueries({ queryKey: seriesKeys.detail(seriesId) })
    },
  })
}

export function useUpdateEpisodeMonitored() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      seriesId,
      episodeId,
      monitored,
    }: {
      seriesId: number
      episodeId: number
      monitored: boolean
    }) => seriesApi.updateEpisodeById(seriesId, episodeId, { monitored }),
    onSuccess: (_, { seriesId }) => {
      void queryClient.invalidateQueries({ queryKey: seriesKeys.detail(seriesId) })
      void queryClient.invalidateQueries({ queryKey: [...seriesKeys.detail(seriesId), 'episodes'] })
      void queryClient.invalidateQueries({ queryKey: missingKeys.all })
    },
  })
}

export function useSearchEpisode() {
  return useMutation({
    mutationFn: ({
      seriesId,
      seasonNumber,
      episodeNumber,
    }: {
      seriesId: number
      seasonNumber: number
      episodeNumber: number
    }) => seriesApi.searchEpisode(seriesId, seasonNumber, episodeNumber),
  })
}

// Bulk monitoring hooks
export function useBulkMonitor() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ seriesId, data }: { seriesId: number; data: BulkMonitorInput }) =>
      seriesApi.bulkMonitor(seriesId, data),
    onSuccess: (_, { seriesId }) => {
      void queryClient.invalidateQueries({ queryKey: seriesKeys.detail(seriesId) })
      void queryClient.invalidateQueries({ queryKey: seriesKeys.seasons(seriesId) })
      void queryClient.invalidateQueries({ queryKey: [...seriesKeys.detail(seriesId), 'episodes'] })
    },
  })
}

export function useBulkMonitorEpisodes() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ seriesId, data }: { seriesId: number; data: BulkEpisodeMonitorInput }) =>
      seriesApi.bulkMonitorEpisodes(seriesId, data),
    onSuccess: (_, { seriesId }) => {
      void queryClient.invalidateQueries({ queryKey: seriesKeys.detail(seriesId) })
      void queryClient.invalidateQueries({ queryKey: [...seriesKeys.detail(seriesId), 'episodes'] })
    },
  })
}

export function useMonitoringStats(seriesId: number) {
  return useQuery({
    queryKey: [...seriesKeys.detail(seriesId), 'monitoringStats'],
    queryFn: () => seriesApi.getMonitoringStats(seriesId),
    enabled: !!seriesId,
  })
}
