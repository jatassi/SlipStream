import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { libraryApi, seriesApi } from '@/api'
import { createQueryKeys } from '@/lib/query-keys'
import type {
  AddSeriesInput,
  ListSeriesOptions,
  Series,
  UpdateSeriesInput,
} from '@/types'

import { calendarKeys } from './use-calendar'
import { missingKeys } from './use-missing'

const baseKeys = createQueryKeys('series')
export const seriesKeys = {
  ...baseKeys,
  list: (filters: ListSeriesOptions) => [...baseKeys.lists(), filters] as const,
  seasons: (seriesId: number) => [...baseKeys.detail(seriesId), 'seasons'] as const,
  episodes: (seriesId: number, seasonNumber?: number) =>
    [...baseKeys.detail(seriesId), 'episodes', seasonNumber] as const,
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

export function useBulkMonitorSeries() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ ids, monitored }: { ids: number[]; monitored: boolean }) =>
      seriesApi.bulkMonitorSeries(ids, monitored),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: seriesKeys.all })
      void queryClient.invalidateQueries({ queryKey: missingKeys.all })
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

