import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { seriesApi, libraryApi } from '@/api'
import type {
  Series,
  CreateSeriesInput,
  AddSeriesInput,
  UpdateSeriesInput,
  UpdateEpisodeInput,
  ListSeriesOptions,
} from '@/types'
import { calendarKeys } from './useCalendar'

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
    queryKey: seriesKeys.list(options || {}),
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
      queryClient.invalidateQueries({ queryKey: seriesKeys.all })
      queryClient.invalidateQueries({ queryKey: calendarKeys.all })
    },
  })
}

export function useAddSeries() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: AddSeriesInput) => libraryApi.addSeries(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: seriesKeys.all })
      queryClient.invalidateQueries({ queryKey: calendarKeys.all })
    },
  })
}

export function useUpdateSeries() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateSeriesInput }) =>
      seriesApi.update(id, data),
    onSuccess: (series: Series) => {
      queryClient.invalidateQueries({ queryKey: seriesKeys.all })
      queryClient.setQueryData(seriesKeys.detail(series.id), series)
    },
  })
}

export function useDeleteSeries() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => seriesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: seriesKeys.all })
      queryClient.invalidateQueries({ queryKey: calendarKeys.all })
    },
  })
}

export function useScanSeries() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => seriesApi.scan(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: seriesKeys.detail(id) })
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
      // Invalidate all series-related queries to refresh seasons and episodes
      queryClient.setQueryData(seriesKeys.detail(series.id), series)
      queryClient.invalidateQueries({ queryKey: seriesKeys.seasons(series.id) })
      // Invalidate all episode queries for this series (any season)
      queryClient.invalidateQueries({
        queryKey: [...seriesKeys.detail(series.id), 'episodes'],
      })
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
      queryClient.invalidateQueries({ queryKey: seriesKeys.detail(seriesId) })
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
    }) => seriesApi.updateEpisode(seriesId, seasonNumber, episodeNumber, data),
    onSuccess: (_, { seriesId }) => {
      queryClient.invalidateQueries({ queryKey: seriesKeys.detail(seriesId) })
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
