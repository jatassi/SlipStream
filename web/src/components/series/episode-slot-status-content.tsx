import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'

import { useEpisodeSlotStatus, useSetEpisodeSlotMonitored } from '@/hooks'
import type { Episode } from '@/types'

import { EpisodeSlotRow } from './episode-slot-row'
import { useSeriesInfo } from './series-context'

type EpisodeSlotStatusContentProps = {
  episode: Episode
  slotQualityProfiles: Record<number, number>
}

export function EpisodeSlotStatusContent({ episode, slotQualityProfiles }: EpisodeSlotStatusContentProps) {
  const { qualityProfileId } = useSeriesInfo()
  const { data: slotStatus, isLoading } = useEpisodeSlotStatus(episode.id)
  const setSlotMonitoredMutation = useSetEpisodeSlotMonitored()

  const handleSlotMonitoredChange = async (slotId: number, monitored: boolean) => {
    try {
      await setSlotMonitoredMutation.mutateAsync({
        episodeId: episode.id,
        slotId,
        data: { monitored },
      })
      toast.success(monitored ? 'Slot monitored' : 'Slot unmonitored')
    } catch {
      toast.error('Failed to update slot monitoring')
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-2">
        <Loader2 className="size-4 animate-spin" />
      </div>
    )
  }

  if (!slotStatus?.slotStatuses || slotStatus.slotStatuses.length === 0) {
    return (
      <div className="text-muted-foreground py-2 text-center text-xs">
        No slot status available
      </div>
    )
  }

  return (
    <EpisodeSlotRow
      slotStatuses={slotStatus.slotStatuses}
      episodeId={episode.id}
      seasonNumber={episode.seasonNumber}
      episodeNumber={episode.episodeNumber}
      qualityProfileId={qualityProfileId}
      slotQualityProfiles={slotQualityProfiles}
      onSlotMonitoredChange={handleSlotMonitoredChange}
      isMonitorUpdating={setSlotMonitoredMutation.isPending}
    />
  )
}
