import { toast } from 'sonner'

import { useUpdateEpisodeMonitored, useUpdateSeasonMonitored, useUpdateSeries } from '@/hooks'
import type { UpgradableSeries } from '@/types/missing'

export type EpisodeMonitoredParams = {
  seriesId: number
  episodeId: number
  label: string
  monitored: boolean
}

export function useUpgradableSeriesList() {
  const updateSeriesMutation = useUpdateSeries()
  const updateSeasonMonitoredMutation = useUpdateSeasonMonitored()
  const updateEpisodeMonitoredMutation = useUpdateEpisodeMonitored()

  const handleSeriesMonitored = async (s: UpgradableSeries, monitored: boolean) => {
    try {
      await updateSeriesMutation.mutateAsync({
        id: s.id,
        data: { monitored },
      })
      toast.success(monitored ? `"${s.title}" monitored` : `"${s.title}" unmonitored`)
    } catch {
      toast.error(`Failed to update "${s.title}"`)
    }
  }

  const handleSeasonMonitored = async (seriesId: number, seasonNumber: number, monitored: boolean) => {
    try {
      await updateSeasonMonitoredMutation.mutateAsync({
        seriesId,
        seasonNumber,
        monitored,
      })
      toast.success(`Season ${seasonNumber} ${monitored ? 'monitored' : 'unmonitored'}`)
    } catch {
      toast.error(`Failed to update Season ${seasonNumber}`)
    }
  }

  const handleEpisodeMonitored = async (params: EpisodeMonitoredParams) => {
    try {
      await updateEpisodeMonitoredMutation.mutateAsync({
        seriesId: params.seriesId,
        episodeId: params.episodeId,
        monitored: params.monitored,
      })
      toast.success(`${params.label} ${params.monitored ? 'monitored' : 'unmonitored'}`)
    } catch {
      toast.error(`Failed to update ${params.label}`)
    }
  }

  return {
    handleSeriesMonitored,
    handleSeasonMonitored,
    handleEpisodeMonitored,
    isSeriesPending: updateSeriesMutation.isPending,
    isSeasonPending: updateSeasonMonitoredMutation.isPending,
    isEpisodePending: updateEpisodeMonitoredMutation.isPending,
  }
}
