import { toast } from 'sonner'

import { useUpdateEpisodeMonitored, useUpdateSeasonMonitored, useUpdateSeries } from '@/hooks'
import type { MissingSeries } from '@/types/missing'

export type EpisodeMonitoredParams = {
  seriesId: number
  episodeId: number
  label: string
  monitored: boolean
}

type SeriesMutation = ReturnType<typeof useUpdateSeries>
type SeasonMutation = ReturnType<typeof useUpdateSeasonMonitored>
type EpisodeMutation = ReturnType<typeof useUpdateEpisodeMonitored>

async function toggleSeriesMonitored(mutation: SeriesMutation, s: MissingSeries, monitored: boolean) {
  try {
    await mutation.mutateAsync({ id: s.id, data: { monitored } })
    toast.success(monitored ? `"${s.title}" monitored` : `"${s.title}" unmonitored`)
  } catch {
    toast.error(`Failed to update "${s.title}"`)
  }
}

type SeasonMonitoredParams = { seriesId: number; seasonNumber: number; monitored: boolean }

async function toggleSeasonMonitored(mutation: SeasonMutation, params: SeasonMonitoredParams) {
  try {
    await mutation.mutateAsync(params)
    toast.success(`Season ${params.seasonNumber} ${params.monitored ? 'monitored' : 'unmonitored'}`)
  } catch {
    toast.error(`Failed to update Season ${params.seasonNumber}`)
  }
}

async function toggleEpisodeMonitored(mutation: EpisodeMutation, params: EpisodeMonitoredParams) {
  try {
    await mutation.mutateAsync({ seriesId: params.seriesId, episodeId: params.episodeId, monitored: params.monitored })
    toast.success(`${params.label} ${params.monitored ? 'monitored' : 'unmonitored'}`)
  } catch {
    toast.error(`Failed to update ${params.label}`)
  }
}

export function useMissingSeriesList() {
  const seriesMutation = useUpdateSeries()
  const seasonMutation = useUpdateSeasonMonitored()
  const episodeMutation = useUpdateEpisodeMonitored()

  return {
    handleSeriesMonitored: (s: MissingSeries, monitored: boolean) => toggleSeriesMonitored(seriesMutation, s, monitored),
    handleSeasonMonitored: (seriesId: number, seasonNumber: number, monitored: boolean) =>
      toggleSeasonMonitored(seasonMutation, { seriesId, seasonNumber, monitored }),
    handleEpisodeMonitored: (params: EpisodeMonitoredParams) => toggleEpisodeMonitored(episodeMutation, params),
    isSeriesPending: seriesMutation.isPending,
    isSeasonPending: seasonMutation.isPending,
    isEpisodePending: episodeMutation.isPending,
  }
}
