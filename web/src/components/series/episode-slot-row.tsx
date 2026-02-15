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
  if (slotStatuses.length === 0) {
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

type SlotBadgeEntry = {
  icon: typeof CheckCircle
  label: string
  variant?: 'destructive' | 'secondary' | 'outline'
  className?: string
}

const SLOT_BADGE_CONFIG: Record<SlotStatus['status'], SlotBadgeEntry> = {
  failed: {
    icon: XCircle,
    label: 'Failed',
    variant: 'destructive',
    className: 'border border-red-500 bg-red-900/50 text-red-400',
  },
  downloading: {
    icon: ArrowDownCircle,
    label: 'Downloading',
    className: 'bg-blue-600 text-white hover:bg-blue-600',
  },
  missing: {
    icon: AlertCircle,
    label: 'Missing',
    variant: 'destructive',
  },
  upgradable: {
    icon: ArrowUpCircle,
    label: 'Upgrade',
    variant: 'secondary',
  },
  available: {
    icon: CheckCircle,
    label: 'OK',
    variant: 'outline',
    className: 'border-green-500 text-green-500',
  },
  unreleased: {
    icon: Clock,
    label: 'Unreleased',
    variant: 'outline',
    className: 'border-amber-500 text-amber-500',
  },
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

  const config = SLOT_BADGE_CONFIG[slot.status]
  const Icon = config.icon
  return (
    <Badge
      variant={config.variant}
      className={cn('h-4 gap-0.5 px-1.5 py-0 text-[10px]', config.className)}
    >
      <Icon className="size-2.5" />
      {config.label}
    </Badge>
  )
}
