import { useState } from 'react'
import { UserSearch, Eye, EyeOff, ChevronDown, Loader2 } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { QualityBadge } from '@/components/media/QualityBadge'
import { MediaStatusBadge } from '@/components/media/MediaStatusBadge'
import { AutoSearchButton } from '@/components/search/AutoSearchButton'
import { EpisodeSlotRow } from './EpisodeSlotRow'
import { useDownloadingStore } from '@/stores'
import { useEpisodeSlotStatus, useSetEpisodeSlotMonitored } from '@/hooks'
import { formatDate } from '@/lib/formatters'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import type { Episode, Slot } from '@/types'

interface EpisodeTableProps {
  seriesId: number
  episodes: Episode[]
  onManualSearch?: (episode: Episode) => void
  onMonitoredChange?: (episode: Episode, monitored: boolean) => void
  onAssignFileToSlot?: (fileId: number, episodeId: number, slotId: number) => void
  onSlotManualSearch?: (episodeId: number, slotId: number) => void
  onSlotAutoSearch?: (episodeId: number, slotId: number) => void
  searchingSlotId?: number | null
  isMultiVersionEnabled?: boolean
  enabledSlots?: Slot[]
  isAssigning?: boolean
  episodeRatings?: Record<number, number>
}

function getRatingColor(rating: number): string {
  if (rating >= 8.0) return 'text-green-400'
  if (rating >= 6.0) return 'text-yellow-400'
  return 'text-red-400'
}

export function EpisodeTable({
  seriesId,
  episodes,
  onManualSearch,
  onMonitoredChange,
  onAssignFileToSlot,
  onSlotManualSearch,
  onSlotAutoSearch,
  searchingSlotId,
  isMultiVersionEnabled = false,
  enabledSlots = [],
  isAssigning = false,
  episodeRatings,
}: EpisodeTableProps) {
  const [expandedEpisodeId, setExpandedEpisodeId] = useState<number | null>(null)

  const queueItems = useDownloadingStore((state) => state.queueItems)

  const getSlotName = (slotId: number | undefined) => {
    if (!slotId) return null
    const slot = enabledSlots.find(s => s.id === slotId)
    return slot?.name ?? null
  }

  const isEpisodeDownloading = (episodeId: number, sId?: number, seasonNum?: number) => {
    return queueItems.some((item) => {
      if (item.status !== 'downloading' && item.status !== 'queued') return false
      if (item.episodeId === episodeId) return true
      if (sId && item.seriesId === sId) {
        if (item.isCompleteSeries) return true
        if (seasonNum && item.seasonNumber === seasonNum && item.isSeasonPack) return true
      }
      return false
    })
  }

  const sortedEpisodes = [...episodes].sort(
    (a, b) => a.episodeNumber - b.episodeNumber
  )

  const columnCount = isMultiVersionEnabled ? 9 : 7

  const toggleExpanded = (episodeId: number) => {
    setExpandedEpisodeId(prev => prev === episodeId ? null : episodeId)
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          {isMultiVersionEnabled && <TableHead className="w-8 px-2" />}
          <TableHead className="w-10 px-2">#</TableHead>
          <TableHead className="px-2">Title</TableHead>
          <TableHead className="px-2">Air Date</TableHead>
          <TableHead className="w-10 px-2 text-center">Status</TableHead>
          <TableHead className="px-2">Quality</TableHead>
          <TableHead className="w-14 px-2 text-center">Rating</TableHead>
          {isMultiVersionEnabled && <TableHead className="px-2">Slot</TableHead>}
          <TableHead className="w-28 px-2">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {sortedEpisodes.map((episode) => (
          <EpisodeRow
            key={episode.id}
            episode={episode}
            seriesId={seriesId}
            columnCount={columnCount}
            isExpanded={expandedEpisodeId === episode.id}
            onToggleExpanded={() => toggleExpanded(episode.id)}
            isDownloading={isEpisodeDownloading(episode.id, seriesId, episode.seasonNumber)}
            searchingSlotId={searchingSlotId}
            isMultiVersionEnabled={isMultiVersionEnabled}
            enabledSlots={enabledSlots}
            isAssigning={isAssigning}
            getSlotName={getSlotName}
            onManualSearch={onManualSearch}
            onMonitoredChange={onMonitoredChange}
            onAssignFileToSlot={onAssignFileToSlot}
            onSlotManualSearch={onSlotManualSearch}
            onSlotAutoSearch={onSlotAutoSearch}
            imdbRating={episodeRatings?.[episode.episodeNumber]}
          />
        ))}
      </TableBody>
    </Table>
  )
}

interface EpisodeRowProps {
  episode: Episode
  seriesId: number
  columnCount: number
  isExpanded: boolean
  onToggleExpanded: () => void
  isDownloading: boolean
  searchingSlotId?: number | null
  isMultiVersionEnabled: boolean
  enabledSlots: Slot[]
  isAssigning: boolean
  getSlotName: (slotId: number | undefined) => string | null
  onManualSearch?: (episode: Episode) => void
  onMonitoredChange?: (episode: Episode, monitored: boolean) => void
  onAssignFileToSlot?: (fileId: number, episodeId: number, slotId: number) => void
  onSlotManualSearch?: (episodeId: number, slotId: number) => void
  onSlotAutoSearch?: (episodeId: number, slotId: number) => void
  imdbRating?: number
}

function EpisodeRow({
  episode,
  seriesId: _seriesId,
  columnCount,
  isExpanded,
  onToggleExpanded,
  isDownloading: _isDownloading,
  searchingSlotId,
  isMultiVersionEnabled,
  enabledSlots,
  isAssigning,
  getSlotName,
  onManualSearch,
  onMonitoredChange,
  onAssignFileToSlot,
  onSlotManualSearch,
  onSlotAutoSearch,
  imdbRating,
}: EpisodeRowProps) {
  const { data: slotStatus, isLoading: isLoadingSlotStatus } = useEpisodeSlotStatus(
    isExpanded ? episode.id : 0
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
        {isMultiVersionEnabled && (
          <TableCell className="px-2 py-1">
            <button
              onClick={onToggleExpanded}
              className="p-1 hover:bg-muted rounded"
            >
              <ChevronDown className={cn('size-4 transition-transform', isExpanded && 'rotate-180')} />
            </button>
          </TableCell>
        )}
        <TableCell className="font-mono px-2 py-1.5">{episode.episodeNumber}</TableCell>
        <TableCell className="font-medium px-2 py-1.5">{episode.title}</TableCell>
        <TableCell className="px-2 py-1.5">
          {episode.airDate ? formatDate(episode.airDate) : '-'}
        </TableCell>
        <TableCell className="px-2 py-1.5 text-center">
          <MediaStatusBadge status={episode.status} iconOnly />
        </TableCell>
        <TableCell className="px-2 py-1.5">
          {episode.episodeFile ? (
            <QualityBadge quality={episode.episodeFile.quality} />
          ) : (
            '-'
          )}
        </TableCell>
        <TableCell className="px-2 py-1.5 text-center">
          {imdbRating != null ? (
            <span className={cn('text-xs font-medium', getRatingColor(imdbRating))}>
              {imdbRating.toFixed(1)}
            </span>
          ) : (
            <span className="text-muted-foreground text-xs">-</span>
          )}
        </TableCell>
        {isMultiVersionEnabled && (
          <TableCell className="px-2 py-1.5">
            {episode.episodeFile ? (
              <Select
                value={episode.episodeFile.slotId?.toString() ?? 'unassigned'}
                onValueChange={(value) => {
                  if (value && value !== 'unassigned' && onAssignFileToSlot) {
                    onAssignFileToSlot(episode.episodeFile!.id, episode.id, parseInt(value, 10))
                  }
                }}
                disabled={isAssigning}
              >
                <SelectTrigger className="w-28 h-7 text-xs">
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
        )}
        <TableCell className="px-2 py-1.5">
          <div className="flex items-center gap-1">
            {onManualSearch && (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => onManualSearch(episode)}
                    />
                  }
                >
                  <UserSearch className="size-3.5" />
                </TooltipTrigger>
                <TooltipContent>
                  <p>Manual Search</p>
                </TooltipContent>
              </Tooltip>
            )}
            <AutoSearchButton
              mediaType="episode"
              episodeId={episode.id}
              title={`S${episode.seasonNumber.toString().padStart(2, '0')}E${episode.episodeNumber.toString().padStart(2, '0')}`}
              showLabel={false}
              variant="ghost"
              size="icon-sm"
            />
            {onMonitoredChange && (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => onMonitoredChange(episode, !episode.monitored)}
                    />
                  }
                >
                  {episode.monitored ? (
                    <Eye className="size-3.5 text-tv-400 icon-glow-tv" />
                  ) : (
                    <EyeOff className="size-3.5" />
                  )}
                </TooltipTrigger>
                <TooltipContent>
                  <p>{episode.monitored ? 'Monitored' : 'Unmonitored'}</p>
                </TooltipContent>
              </Tooltip>
            )}
          </div>
        </TableCell>
      </TableRow>
      {isMultiVersionEnabled && isExpanded && (
        <TableRow>
          <TableCell colSpan={columnCount} className="p-2 bg-muted/20">
            {isLoadingSlotStatus ? (
              <div className="flex items-center justify-center py-2">
                <Loader2 className="size-4 animate-spin" />
              </div>
            ) : slotStatus?.slotStatuses && slotStatus.slotStatuses.length > 0 ? (
              <EpisodeSlotRow
                slotStatuses={slotStatus.slotStatuses}
                onToggleMonitored={handleSlotMonitoredChange}
                onManualSearch={(slotId) => onSlotManualSearch?.(episode.id, slotId)}
                onAutoSearch={(slotId) => onSlotAutoSearch?.(episode.id, slotId)}
                isUpdating={setSlotMonitoredMutation.isPending}
                isSearching={searchingSlotId}
              />
            ) : (
              <div className="text-xs text-muted-foreground text-center py-2">
                No slot status available
              </div>
            )}
          </TableCell>
        </TableRow>
      )}
    </>
  )
}
