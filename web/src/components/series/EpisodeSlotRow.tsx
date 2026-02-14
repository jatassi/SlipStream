import {
  AlertCircle,
  ArrowDownCircle,
  ArrowUpCircle,
  CheckCircle,
  Clock,
  EyeOff,
  XCircle,
} from 'lucide-react'

import { MediaSearchMonitorControls } from '@/components/search'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { SlotStatus } from '@/types'

type EpisodeSlotRowProps = {
  slotStatuses: SlotStatus[]
  episodeId: number
  seriesId: number
  seriesTitle: string
  seasonNumber: number
  episodeNumber: number
  qualityProfileId: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  slotQualityProfiles?: Record<number, number>
  onSlotMonitoredChange?: (slotId: number, monitored: boolean) => void
  isMonitorUpdating?: boolean
}

export function EpisodeSlotRow({
  slotStatuses,
  episodeId,
  seriesId,
  seriesTitle,
  seasonNumber,
  episodeNumber,
  qualityProfileId,
  tvdbId,
  tmdbId,
  imdbId,
  slotQualityProfiles,
  onSlotMonitoredChange,
  isMonitorUpdating,
}: EpisodeSlotRowProps) {
  if (!slotStatuses || slotStatuses.length === 0) {
    return null
  }

  return (
    <div className="bg-muted/30 space-y-1 rounded-md px-3 py-2">
      {slotStatuses.map((slot) => (
        <CompactSlotItem
          key={slot.slotId}
          slot={slot}
          episodeId={episodeId}
          seriesId={seriesId}
          seriesTitle={seriesTitle}
          seasonNumber={seasonNumber}
          episodeNumber={episodeNumber}
          qualityProfileId={slotQualityProfiles?.[slot.slotId] ?? qualityProfileId}
          tvdbId={tvdbId}
          tmdbId={tmdbId}
          imdbId={imdbId}
          onMonitoredChange={onSlotMonitoredChange}
          isMonitorUpdating={isMonitorUpdating}
        />
      ))}
    </div>
  )
}

type CompactSlotItemProps = {
  slot: SlotStatus
  episodeId: number
  seriesId: number
  seriesTitle: string
  seasonNumber: number
  episodeNumber: number
  qualityProfileId: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  onMonitoredChange?: (slotId: number, monitored: boolean) => void
  isMonitorUpdating?: boolean
}

function CompactSlotItem({
  slot,
  episodeId,
  seriesId,
  seriesTitle,
  seasonNumber,
  episodeNumber,
  qualityProfileId,
  tvdbId,
  tmdbId,
  imdbId,
  onMonitoredChange,
  isMonitorUpdating,
}: CompactSlotItemProps) {
  return (
    <div className="flex items-center justify-between gap-2 py-1 text-xs">
      <div className="flex min-w-0 items-center gap-2">
        <span className="shrink-0 font-medium">{slot.slotName}</span>
        <CompactSlotBadge slot={slot} />
        {slot.currentQuality ? (
          <Badge variant="outline" className="h-4 px-1.5 py-0 text-[10px]">
            {slot.currentQuality}
          </Badge>
        ) : null}
      </div>

      <div className="flex shrink-0 items-center gap-1">
        <MediaSearchMonitorControls
          mediaType="episode-slot"
          episodeId={episodeId}
          slotId={slot.slotId}
          seriesId={seriesId}
          seriesTitle={seriesTitle}
          seasonNumber={seasonNumber}
          episodeNumber={episodeNumber}
          title={`${slot.slotName} S${seasonNumber.toString().padStart(2, '0')}E${episodeNumber.toString().padStart(2, '0')}`}
          theme="tv"
          size="xs"
          monitored={slot.monitored}
          onMonitoredChange={(m) => onMonitoredChange?.(slot.slotId, m)}
          monitorDisabled={isMonitorUpdating}
          qualityProfileId={qualityProfileId}
          tvdbId={tvdbId}
          tmdbId={tmdbId}
          imdbId={imdbId}
        />
      </div>
    </div>
  )
}

function CompactSlotBadge({ slot }: { slot: SlotStatus }) {
  if (!slot.monitored && slot.status !== 'available' && slot.status !== 'upgradable') {
    return (
      <Badge
        variant="outline"
        className="text-muted-foreground h-4 gap-0.5 px-1.5 py-0 text-[10px]"
      >
        <EyeOff className="size-2.5" />
        Not Monitored
      </Badge>
    )
  }

  switch (slot.status) {
    case 'failed': {
      return (
        <Badge
          variant="destructive"
          className="h-4 gap-0.5 border border-red-500 bg-red-900/50 px-1.5 py-0 text-[10px] text-red-400"
        >
          <XCircle className="size-2.5" />
          Failed
        </Badge>
      )
    }
    case 'downloading': {
      return (
        <Badge className="h-4 gap-0.5 bg-blue-600 px-1.5 py-0 text-[10px] text-white hover:bg-blue-600">
          <ArrowDownCircle className="size-2.5" />
          Downloading
        </Badge>
      )
    }
    case 'missing': {
      return (
        <Badge variant="destructive" className="h-4 gap-0.5 px-1.5 py-0 text-[10px]">
          <AlertCircle className="size-2.5" />
          Missing
        </Badge>
      )
    }
    case 'upgradable': {
      return (
        <Badge variant="secondary" className="h-4 gap-0.5 px-1.5 py-0 text-[10px]">
          <ArrowUpCircle className="size-2.5" />
          Upgrade
        </Badge>
      )
    }
    case 'available': {
      return (
        <Badge
          variant="outline"
          className={cn('h-4 gap-0.5 px-1.5 py-0 text-[10px]', 'border-green-500 text-green-500')}
        >
          <CheckCircle className="size-2.5" />
          OK
        </Badge>
      )
    }
    case 'unreleased': {
      return (
        <Badge
          variant="outline"
          className="h-4 gap-0.5 border-amber-500 px-1.5 py-0 text-[10px] text-amber-500"
        >
          <Clock className="size-2.5" />
          Unreleased
        </Badge>
      )
    }
    default: {
      return (
        <Badge variant="outline" className="text-muted-foreground h-4 px-1.5 py-0 text-[10px]">
          Empty
        </Badge>
      )
    }
  }
}
