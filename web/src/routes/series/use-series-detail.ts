import { useMemo, useState } from 'react'

import { useQueryClient } from '@tanstack/react-query'
import { useNavigate, useParams } from '@tanstack/react-router'
import { toast } from 'sonner'

import {
  useAssignEpisodeFile,
  useDeleteSeries,
  useEpisodes,
  useExtendedSeriesMetadata,
  useMultiVersionSettings,
  useQualityProfiles,
  useRefreshSeries,
  useSeriesDetail,
  useSlots,
  useUpdateEpisodeMonitored,
  useUpdateSeasonMonitored,
  useUpdateSeries,
} from '@/hooks'
import { seriesKeys } from '@/hooks/use-series'
import { withToast } from '@/lib/with-toast'
import { useUIStore } from '@/stores'
import type { Episode } from '@/types'

function buildEpisodeRatings(
  extendedSeasons: { seasonNumber: number; episodes?: { episodeNumber: number; imdbRating?: number }[] }[] | undefined,
): Record<number, Record<number, number>> | undefined {
  if (!extendedSeasons) {return undefined}

  const map: Record<number, Record<number, number>> = {}
  for (const season of extendedSeasons) {
    if (!season.episodes) {continue}
    const seasonMap: Record<number, number> = {}
    for (const ep of season.episodes) {
      if (ep.imdbRating) {
        seasonMap[ep.episodeNumber] = ep.imdbRating
      }
    }
    if (Object.keys(seasonMap).length > 0) {
      map[season.seasonNumber] = seasonMap
    }
  }
  return Object.keys(map).length > 0 ? map : undefined
}

function useSeriesQueries(seriesId: number) {
  const globalLoading = useUIStore((s) => s.globalLoading)
  const queryClient = useQueryClient()
  const { data: series, isLoading: queryLoading, isError, refetch } = useSeriesDetail(seriesId)
  const isLoading = queryLoading || globalLoading
  const cachedSeries = queryClient.getQueryData<{ tmdbId?: number }>(seriesKeys.detail(seriesId))
  const tmdbId = series?.tmdbId ?? cachedSeries?.tmdbId ?? 0
  const { data: extendedData, isLoading: isExtendedDataLoading } = useExtendedSeriesMetadata(tmdbId)
  const { data: qualityProfiles } = useQualityProfiles('tv')
  const { data: episodes } = useEpisodes(seriesId)
  const { data: multiVersionSettings } = useMultiVersionSettings()
  const { data: slots } = useSlots()
  const enabledSlots = useMemo(() => slots?.filter((s) => s.enabled) ?? [], [slots])

  return {
    series, isLoading, isError, refetch,
    extendedData, isExtendedDataLoading, qualityProfiles, episodes,
    isMultiVersionEnabled: multiVersionSettings?.enabled ?? false,
    enabledSlots,
  }
}

function formatEpisodeCode(ep: Episode): string {
  return `S${ep.seasonNumber.toString().padStart(2, '0')}E${ep.episodeNumber.toString().padStart(2, '0')}`
}

function useSeriesMutations(seriesId: number, refetch: () => void) {
  const navigate = useNavigate()
  const update = useUpdateSeries()
  const remove = useDeleteSeries()
  const refresh = useRefreshSeries()
  const seasonMonitor = useUpdateSeasonMonitored()
  const episodeMonitor = useUpdateEpisodeMonitored()
  const assign = useAssignEpisodeFile()

  return {
    isAssigning: assign.isPending,
    isRefreshing: refresh.isPending,
    handleAssignFileToSlot: withToast(async (fileId: number, episodeId: number, slotId: number) => {
      await assign.mutateAsync({ episodeId, slotId, data: { fileId } })
      refetch()
      toast.success('File assigned to slot')
    }),
    handleToggleMonitored: async (series: { id: number; monitored: boolean } | undefined, newMonitored?: boolean) => {
      if (!series) {return}
      const target = newMonitored ?? !series.monitored
      await withToast(async () => {
        await update.mutateAsync({ id: series.id, data: { monitored: target } })
        toast.success(target ? 'Series monitored' : 'Series unmonitored')
      })()
    },
    handleRefresh: withToast(async () => {
      await refresh.mutateAsync(seriesId)
      toast.success('Metadata refreshed')
    }),
    handleDelete: withToast(async () => {
      await remove.mutateAsync({ id: seriesId })
      toast.success('Series deleted')
      void navigate({ to: '/series' })
    }),
    handleSeasonMonitoredChange: withToast(async (seasonNumber: number, monitored: boolean) => {
      await seasonMonitor.mutateAsync({ seriesId, seasonNumber, monitored })
      toast.success(`Season ${seasonNumber} ${monitored ? 'monitored' : 'unmonitored'}`)
    }),
    handleEpisodeMonitoredChange: withToast(async (episode: Episode, monitored: boolean) => {
      await episodeMonitor.mutateAsync({ seriesId, episodeId: episode.id, monitored })
      toast.success(`${formatEpisodeCode(episode)} ${monitored ? 'monitored' : 'unmonitored'}`)
    }),
  }
}

export function useSeriesDetailPage() {
  const params: { id: string } = useParams({ from: '/series/$id' })
  const seriesId = Number.parseInt(params.id)

  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [overviewExpanded, setOverviewExpanded] = useState(false)

  const queries = useSeriesQueries(seriesId)
  const mutations = useSeriesMutations(seriesId, () => { void queries.refetch() })

  const qualityProfileName = useMemo(
    () => queries.qualityProfiles?.find((p) => p.id === queries.series?.qualityProfileId)?.name,
    [queries.qualityProfiles, queries.series?.qualityProfileId],
  )

  const episodeRatings = useMemo(
    () => buildEpisodeRatings(queries.extendedData?.seasons),
    [queries.extendedData?.seasons],
  )

  return {
    ...queries,
    qualityProfileName,
    episodeRatings,
    ...mutations,
    handleToggleMonitored: (newMonitored?: boolean) => mutations.handleToggleMonitored(queries.series, newMonitored),
    editDialogOpen,
    setEditDialogOpen,
    overviewExpanded,
    setOverviewExpanded,
  }
}
