import { useMemo, useState } from 'react'

import { ChevronDown, Loader2 } from 'lucide-react'
import { toast } from 'sonner'

import { MediaStatusBadge } from '@/components/media/media-status-badge'
import { QualityBadge } from '@/components/media/quality-badge'
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

import { EpisodeSlotRow } from './episode-slot-row'

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

function getRatingColor(rating: number): string {
  if (rating >= 8) {
    return 'text-green-400'
  }
  if (rating >= 6) {
    return 'text-yellow-400'
  }
  return 'text-red-400'
}

function buildSlotQualityMap(slots: Slot[]): Record<number, number> {
  const map: Record<number, number> = {}
  for (const slot of slots) {
    if (slot.qualityProfileId !== null) {
      map[slot.id] = slot.qualityProfileId
    }
  }
  return map
}

function findSlotName(slots: Slot[], slotId: number | undefined): string | null {
  if (!slotId) {
    return null
  }
  return slots.find((s) => s.id === slotId)?.name ?? null
}

export function EpisodeTable(props: EpisodeTableProps) {
  const { episodes, isMultiVersionEnabled = false, enabledSlots = [] } = props
  const [expandedEpisodeId, setExpandedEpisodeId] = useState<number | null>(null)
  const slotQualityProfiles = useMemo(() => buildSlotQualityMap(enabledSlots), [enabledSlots])
  const getSlotName = (slotId: number | undefined) => findSlotName(enabledSlots, slotId)
  const sortedEpisodes = episodes.toSorted((a, b) => a.episodeNumber - b.episodeNumber)
  const columnCount = isMultiVersionEnabled ? 9 : 7

  return (
    <Table>
      <EpisodeTableHeader isMultiVersionEnabled={isMultiVersionEnabled} />
      <TableBody>
        {sortedEpisodes.map((episode) => (
          <EpisodeRow
            key={episode.id}
            episode={episode}
            seriesId={props.seriesId}
            seriesTitle={props.seriesTitle}
            qualityProfileId={props.qualityProfileId}
            tvdbId={props.tvdbId}
            tmdbId={props.tmdbId}
            imdbId={props.imdbId}
            columnCount={columnCount}
            isExpanded={expandedEpisodeId === episode.id}
            onToggleExpanded={() => setExpandedEpisodeId((prev) => (prev === episode.id ? null : episode.id))}
            isMultiVersionEnabled={isMultiVersionEnabled}
            enabledSlots={enabledSlots}
            isAssigning={props.isAssigning ?? false}
            getSlotName={getSlotName}
            onMonitoredChange={props.onMonitoredChange}
            onAssignFileToSlot={props.onAssignFileToSlot}
            slotQualityProfiles={slotQualityProfiles}
            imdbRating={props.episodeRatings?.[episode.episodeNumber]}
          />
        ))}
      </TableBody>
    </Table>
  )
}

function EpisodeTableHeader({ isMultiVersionEnabled }: { isMultiVersionEnabled: boolean }) {
  return (
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
  )
}

function EpisodeRow(props: EpisodeRowProps) {
  const { episode, isExpanded, isMultiVersionEnabled, columnCount } = props

  return (
    <>
      <TableRow className={cn(isExpanded && 'border-b-0')}>
        <EpisodeRowCells {...props} />
      </TableRow>
      {isMultiVersionEnabled && isExpanded ? (
        <TableRow>
          <TableCell colSpan={columnCount} className="bg-muted/20 p-2">
            <EpisodeSlotStatusContent
              episode={episode}
              seriesId={props.seriesId}
              seriesTitle={props.seriesTitle}
              qualityProfileId={props.qualityProfileId}
              tvdbId={props.tvdbId}
              tmdbId={props.tmdbId}
              imdbId={props.imdbId}
              slotQualityProfiles={props.slotQualityProfiles}
            />
          </TableCell>
        </TableRow>
      ) : null}
    </>
  )
}

function EpisodeRowCells(props: EpisodeRowProps) {
  const { episode, isMultiVersionEnabled, onToggleExpanded, imdbRating } = props

  return (
    <>
      {isMultiVersionEnabled ? (
        <TableCell className="px-2 py-1">
          <button onClick={onToggleExpanded} className="hover:bg-muted rounded p-1">
            <ChevronDown className={cn('size-4 transition-transform', props.isExpanded && 'rotate-180')} />
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
      <RatingCell imdbRating={imdbRating} />
      {isMultiVersionEnabled ? <SlotAssignCell {...props} /> : null}
      <TableCell className="px-2 py-1.5">
        <EpisodeActions {...props} />
      </TableCell>
    </>
  )
}

function RatingCell({ imdbRating }: { imdbRating?: number }) {
  return (
    <TableCell className="px-2 py-1.5 text-center">
      {/* eslint-disable-next-line @typescript-eslint/no-unnecessary-condition */}
      {imdbRating === null || imdbRating === undefined ? (
        <span className="text-muted-foreground text-xs">-</span>
      ) : (
        <span className={cn('text-xs font-medium', getRatingColor(imdbRating))}>
          {imdbRating.toFixed(1)}
        </span>
      )}
    </TableCell>
  )
}

function SlotAssignCell(props: EpisodeRowProps) {
  const { episode, enabledSlots, isAssigning, getSlotName, onAssignFileToSlot } = props

  if (!episode.episodeFile) {
    return (
      <TableCell className="px-2 py-1.5">
        <span className="text-muted-foreground text-xs">-</span>
      </TableCell>
    )
  }

  const handleSlotChange = (value: string) => {
    if (value && value !== 'unassigned' && onAssignFileToSlot && episode.episodeFile) {
      onAssignFileToSlot(episode.episodeFile.id, episode.id, Number.parseInt(value, 10))
    }
  }

  return (
    <TableCell className="px-2 py-1.5">
      <Select
        value={episode.episodeFile.slotId?.toString() ?? 'unassigned'}
        onValueChange={(v) => v && handleSlotChange(v)}
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
    </TableCell>
  )
}

function EpisodeActions(props: EpisodeRowProps) {
  const { episode, seriesId, seriesTitle, qualityProfileId, tvdbId, tmdbId, imdbId } = props
  const episodeCode = `S${episode.seasonNumber.toString().padStart(2, '0')}E${episode.episodeNumber.toString().padStart(2, '0')}`

  return (
    <MediaSearchMonitorControls
      mediaType="episode"
      episodeId={episode.id}
      seriesId={seriesId}
      seriesTitle={seriesTitle}
      seasonNumber={episode.seasonNumber}
      episodeNumber={episode.episodeNumber}
      title={episodeCode}
      theme="tv"
      size="xs"
      monitored={episode.monitored}
      onMonitoredChange={(m) => props.onMonitoredChange?.(episode, m)}
      monitorDisabled={!props.onMonitoredChange}
      qualityProfileId={qualityProfileId}
      tvdbId={tvdbId}
      tmdbId={tmdbId}
      imdbId={imdbId}
    />
  )
}

type SlotStatusContentProps = {
  episode: Episode
  seriesId: number
  seriesTitle: string
  qualityProfileId: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  slotQualityProfiles: Record<number, number>
}

function EpisodeSlotStatusContent(props: SlotStatusContentProps) {
  const { episode, seriesId, seriesTitle, qualityProfileId } = props
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
      seriesId={seriesId}
      seriesTitle={seriesTitle}
      seasonNumber={episode.seasonNumber}
      episodeNumber={episode.episodeNumber}
      qualityProfileId={qualityProfileId}
      tvdbId={props.tvdbId}
      tmdbId={props.tmdbId}
      imdbId={props.imdbId}
      slotQualityProfiles={props.slotQualityProfiles}
      onSlotMonitoredChange={handleSlotMonitoredChange}
      isMonitorUpdating={setSlotMonitoredMutation.isPending}
    />
  )
}
