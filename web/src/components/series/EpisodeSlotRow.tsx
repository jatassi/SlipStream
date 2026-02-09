import { EyeOff, ArrowUpCircle, ArrowDownCircle, AlertCircle, CheckCircle, XCircle, Clock } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { MediaSearchMonitorControls } from '@/components/search'
import type { SlotStatus } from '@/types'
import { cn } from '@/lib/utils'

interface EpisodeSlotRowProps {
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
    <div className="space-y-1 py-2 px-3 bg-muted/30 rounded-md">
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

interface CompactSlotItemProps {
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
      <div className="flex items-center gap-2 min-w-0">
        <span className="font-medium shrink-0">{slot.slotName}</span>
        <CompactSlotBadge slot={slot} />
        {slot.currentQuality && (
          <Badge variant="outline" className="text-[10px] px-1.5 py-0 h-4">
            {slot.currentQuality}
          </Badge>
        )}
      </div>

      <div className="flex items-center gap-1 shrink-0">
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
      <Badge variant="outline" className="text-[10px] px-1.5 py-0 h-4 gap-0.5 text-muted-foreground">
        <EyeOff className="size-2.5" />
        Not Monitored
      </Badge>
    )
  }

  switch (slot.status) {
    case 'failed':
      return (
        <Badge variant="destructive" className="text-[10px] px-1.5 py-0 h-4 gap-0.5 bg-red-900/50 border border-red-500 text-red-400">
          <XCircle className="size-2.5" />
          Failed
        </Badge>
      )
    case 'downloading':
      return (
        <Badge className="text-[10px] px-1.5 py-0 h-4 gap-0.5 bg-blue-600 hover:bg-blue-600 text-white">
          <ArrowDownCircle className="size-2.5" />
          Downloading
        </Badge>
      )
    case 'missing':
      return (
        <Badge variant="destructive" className="text-[10px] px-1.5 py-0 h-4 gap-0.5">
          <AlertCircle className="size-2.5" />
          Missing
        </Badge>
      )
    case 'upgradable':
      return (
        <Badge variant="secondary" className="text-[10px] px-1.5 py-0 h-4 gap-0.5">
          <ArrowUpCircle className="size-2.5" />
          Upgrade
        </Badge>
      )
    case 'available':
      return (
        <Badge variant="outline" className={cn('text-[10px] px-1.5 py-0 h-4 gap-0.5', 'border-green-500 text-green-500')}>
          <CheckCircle className="size-2.5" />
          OK
        </Badge>
      )
    case 'unreleased':
      return (
        <Badge variant="outline" className="text-[10px] px-1.5 py-0 h-4 gap-0.5 border-amber-500 text-amber-500">
          <Clock className="size-2.5" />
          Unreleased
        </Badge>
      )
    default:
      return (
        <Badge variant="outline" className="text-[10px] px-1.5 py-0 h-4 text-muted-foreground">
          Empty
        </Badge>
      )
  }
}
