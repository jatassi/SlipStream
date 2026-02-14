import { useMemo, useState } from 'react'

import { ChevronDown, Loader2 } from 'lucide-react'
import { toast } from 'sonner'

import { MediaStatusBadge } from '@/components/media/MediaStatusBadge'
import { QualityBadge } from '@/components/media/QualityBadge'
import { MediaSearchMonitorControls } from '@/components/search'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useEpisodeSlotStatus, useSetEpisodeSlotMonitored } from '@/hooks'
import { formatDate } from '@/lib/formatters'
import { cn } from '@/lib/utils'
import type { Episode, Slot } from '@/types'

import { EpisodeSlotRow } from './EpisodeSlotRow'

type EpisodeTableProps = {
  seriesId: number
  seriesTitle: string
  qualityProfileId: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  episodes: Episode[]
  onMonitoredChange?: (episode: Episode, monitored: boolean) => void
  onAssignFileToSlot?: (fileId: number, episodeId: number, slotId: number) => void
  isMultiVersionEnabled?: boolean
  enabledSlots?: Slot[]
  isAssigning?: boolean
  episodeRatings?: Record<number, number>
}

function getRatingColor(rating: number): string {
  if (rating >= 8) {
    return 'text-green-400'
  }
  if (rating >= 6) {
    return 'text-yellow-400'
  }
  return 'text-red-400'
}

export function EpisodeTable({
  seriesId,
  seriesTitle,
  qualityProfileId,
  tvdbId,
  tmdbId,
  imdbId,
  episodes,
  onMonitoredChange,
  onAssignFileToSlot,
  isMultiVersionEnabled = false,
  enabledSlots = [],
  isAssigning = false,
  episodeRatings,
}: EpisodeTableProps) {
  const [expandedEpisodeId, setExpandedEpisodeId] = useState<number | null>(null)

  const slotQualityProfiles = useMemo(() => {
    const map: Record<number, number> = {}
    for (const slot of enabledSlots) {
      if (slot.qualityProfileId != null) {
        map[slot.id] = slot.qualityProfileId
      }
    }
    return map
  }, [enabledSlots])

  const getSlotName = (slotId: number | undefined) => {
    if (!slotId) {
      return null
    }
    const slot = enabledSlots.find((s) => s.id === slotId)
    return slot?.name ?? null
  }

  const sortedEpisodes = [...episodes].sort((a, b) => a.episodeNumber - b.episodeNumber)

  const columnCount = isMultiVersionEnabled ? 9 : 7

  const toggleExpanded = (episodeId: number) => {
    setExpandedEpisodeId((prev) => (prev === episodeId ? null : episodeId))
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          {isMultiVersionEnabled ? <TableHead className="w-8 px-2" /> : null}
          <TableHead className="w-10 px-2">#</TableHead>
          <TableHead className="px-2">Title</TableHead>
          <TableHead className="px-2">Air Date</TableHead>
          <TableHead className="w-10 px-2 text-center">Status</TableHead>
          <TableHead className="px-2">Quality</TableHead>
          <TableHead className="w-14 px-2 text-center">Rating</TableHead>
          {isMultiVersionEnabled ? <TableHead className="px-2">Slot</TableHead> : null}
          <TableHead className="w-28 px-2">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {sortedEpisodes.map((episode) => (
          <EpisodeRow
            key={episode.id}
            episode={episode}
            seriesId={seriesId}
            seriesTitle={seriesTitle}
            qualityProfileId={qualityProfileId}
            tvdbId={tvdbId}
            tmdbId={tmdbId}
            imdbId={imdbId}
            columnCount={columnCount}
            isExpanded={expandedEpisodeId === episode.id}
            onToggleExpanded={() => toggleExpanded(episode.id)}
            isMultiVersionEnabled={isMultiVersionEnabled}
            enabledSlots={enabledSlots}
            isAssigning={isAssigning}
            getSlotName={getSlotName}
            onMonitoredChange={onMonitoredChange}
            onAssignFileToSlot={onAssignFileToSlot}
            slotQualityProfiles={slotQualityProfiles}
            imdbRating={episodeRatings?.[episode.episodeNumber]}
          />
        ))}
      </TableBody>
    </Table>
  )
}

type EpisodeRowProps = {
  episode: Episode
  seriesId: number
  seriesTitle: string
  qualityProfileId: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  columnCount: number
  isExpanded: boolean
  onToggleExpanded: () => void
  isMultiVersionEnabled: boolean
  enabledSlots: Slot[]
  isAssigning: boolean
  getSlotName: (slotId: number | undefined) => string | null
  onMonitoredChange?: (episode: Episode, monitored: boolean) => void
  onAssignFileToSlot?: (fileId: number, episodeId: number, slotId: number) => void
  slotQualityProfiles: Record<number, number>
  imdbRating?: number
}

function EpisodeRow({
  episode,
  seriesId,
  seriesTitle,
  qualityProfileId,
  tvdbId,
  tmdbId,
  imdbId,
  columnCount,
  isExpanded,
  onToggleExpanded,
  isMultiVersionEnabled,
  enabledSlots,
  isAssigning,
  getSlotName,
  onMonitoredChange,
  onAssignFileToSlot,
  slotQualityProfiles,
  imdbRating,
}: EpisodeRowProps) {
  const { data: slotStatus, isLoading: isLoadingSlotStatus } = useEpisodeSlotStatus(
    isExpanded ? episode.id : 0,
  )
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

  return (
    <>
      <TableRow className={cn(isExpanded && 'border-b-0')}>
        {isMultiVersionEnabled ? (
          <TableCell className="px-2 py-1">
            <button onClick={onToggleExpanded} className="hover:bg-muted rounded p-1">
              <ChevronDown
                className={cn('size-4 transition-transform', isExpanded && 'rotate-180')}
              />
            </button>
          </TableCell>
        ) : null}
        <TableCell className="px-2 py-1.5 font-mono">{episode.episodeNumber}</TableCell>
        <TableCell className="px-2 py-1.5 font-medium">{episode.title}</TableCell>
        <TableCell className="px-2 py-1.5">
          {episode.airDate ? formatDate(episode.airDate) : '-'}
        </TableCell>
        <TableCell className="px-2 py-1.5 text-center">
          <MediaStatusBadge status={episode.status} iconOnly />
        </TableCell>
        <TableCell className="px-2 py-1.5">
          {episode.episodeFile ? <QualityBadge quality={episode.episodeFile.quality} /> : '-'}
        </TableCell>
        <TableCell className="px-2 py-1.5 text-center">
          {imdbRating == null ? (
            <span className="text-muted-foreground text-xs">-</span>
          ) : (
            <span className={cn('text-xs font-medium', getRatingColor(imdbRating))}>
              {imdbRating.toFixed(1)}
            </span>
          )}
        </TableCell>
        {isMultiVersionEnabled ? (
          <TableCell className="px-2 py-1.5">
            {episode.episodeFile ? (
              <Select
                value={episode.episodeFile.slotId?.toString() ?? 'unassigned'}
                onValueChange={(value) => {
                  if (value && value !== 'unassigned' && onAssignFileToSlot) {
                    onAssignFileToSlot(
                      episode.episodeFile!.id,
                      episode.id,
                      Number.parseInt(value, 10),
                    )
                  }
                }}
                disabled={isAssigning}
              >
                <SelectTrigger className="h-7 w-28 text-xs">
                  {getSlotName(episode.episodeFile.slotId) ?? (
                    <span className="text-muted-foreground">Unassigned</span>
                  )}
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="unassigned" disabled>
                    Unassigned
                  </SelectItem>
                  {enabledSlots.map((slot) => (
                    <SelectItem key={slot.id} value={slot.id.toString()}>
                      {slot.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : (
              <span className="text-muted-foreground text-xs">-</span>
            )}
          </TableCell>
        ) : null}
        <TableCell className="px-2 py-1.5">
          <MediaSearchMonitorControls
            mediaType="episode"
            episodeId={episode.id}
            seriesId={seriesId}
            seriesTitle={seriesTitle}
            seasonNumber={episode.seasonNumber}
            episodeNumber={episode.episodeNumber}
            title={`S${episode.seasonNumber.toString().padStart(2, '0')}E${episode.episodeNumber.toString().padStart(2, '0')}`}
            theme="tv"
            size="xs"
            monitored={episode.monitored}
            onMonitoredChange={(m) => onMonitoredChange?.(episode, m)}
            monitorDisabled={!onMonitoredChange}
            qualityProfileId={qualityProfileId}
            tvdbId={tvdbId}
            tmdbId={tmdbId}
            imdbId={imdbId}
          />
        </TableCell>
      </TableRow>
      {isMultiVersionEnabled && isExpanded ? (
        <TableRow>
          <TableCell colSpan={columnCount} className="bg-muted/20 p-2">
            {isLoadingSlotStatus ? (
              <div className="flex items-center justify-center py-2">
                <Loader2 className="size-4 animate-spin" />
              </div>
            ) : slotStatus?.slotStatuses && slotStatus.slotStatuses.length > 0 ? (
              <EpisodeSlotRow
                slotStatuses={slotStatus.slotStatuses}
                episodeId={episode.id}
                seriesId={seriesId}
                seriesTitle={seriesTitle}
                seasonNumber={episode.seasonNumber}
                episodeNumber={episode.episodeNumber}
                qualityProfileId={qualityProfileId}
                tvdbId={tvdbId}
                tmdbId={tmdbId}
                imdbId={imdbId}
                slotQualityProfiles={slotQualityProfiles}
                onSlotMonitoredChange={handleSlotMonitoredChange}
                isMonitorUpdating={setSlotMonitoredMutation.isPending}
              />
            ) : (
              <div className="text-muted-foreground py-2 text-center text-xs">
                No slot status available
              </div>
            )}
          </TableCell>
        </TableRow>
      ) : null}
    </>
  )
}
