import { useState } from 'react'
import { Search, Check, X, MoreHorizontal, Download, Zap, Loader2, ChevronDown } from 'lucide-react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { QualityBadge } from '@/components/media/QualityBadge'
import { EpisodeAvailabilityBadge } from '@/components/media/AvailabilityBadge'
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
  onAutoSearch?: (episode: Episode) => void
  onManualSearch?: (episode: Episode) => void
  onMonitoredChange?: (episode: Episode, monitored: boolean) => void
  onAssignFileToSlot?: (fileId: number, episodeId: number, slotId: number) => void
  onSlotManualSearch?: (episodeId: number, slotId: number) => void
  onSlotAutoSearch?: (episodeId: number, slotId: number) => void
  searchingEpisodeId?: number | null
  searchingSlotId?: number | null
  isMultiVersionEnabled?: boolean
  enabledSlots?: Slot[]
  isAssigning?: boolean
}

export function EpisodeTable({
  seriesId,
  episodes,
  onAutoSearch,
  onManualSearch,
  onMonitoredChange,
  onAssignFileToSlot,
  onSlotManualSearch,
  onSlotAutoSearch,
  searchingEpisodeId,
  searchingSlotId,
  isMultiVersionEnabled = false,
  enabledSlots = [],
  isAssigning = false,
}: EpisodeTableProps) {
  const [expandedEpisodeId, setExpandedEpisodeId] = useState<number | null>(null)

  // Select queueItems directly so component re-renders when it changes
  const queueItems = useDownloadingStore((state) => state.queueItems)

  const getSlotName = (slotId: number | undefined) => {
    if (!slotId) return null
    const slot = enabledSlots.find(s => s.id === slotId)
    return slot?.name ?? null
  }

  const isEpisodeDownloading = (episodeId: number, sId?: number, seasonNum?: number) => {
    return queueItems.some((item) => {
      if (item.status !== 'downloading' && item.status !== 'queued') return false
      // Direct episode match
      if (item.episodeId === episodeId) return true
      // Season pack or complete series covering this episode
      if (sId && item.seriesId === sId) {
        if (item.isCompleteSeries) return true
        if (seasonNum && item.seasonNumber === seasonNum && item.isSeasonPack) return true
      }
      return false
    })
  }

  // Sort by episode number
  const sortedEpisodes = [...episodes].sort(
    (a, b) => a.episodeNumber - b.episodeNumber
  )

  const columnCount = isMultiVersionEnabled ? 10 : 8

  const toggleExpanded = (episodeId: number) => {
    setExpandedEpisodeId(prev => prev === episodeId ? null : episodeId)
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          {isMultiVersionEnabled && <TableHead className="w-8" />}
          <TableHead className="w-16">#</TableHead>
          <TableHead>Title</TableHead>
          <TableHead>Air Date</TableHead>
          <TableHead>Aired</TableHead>
          <TableHead className="w-24">Monitored</TableHead>
          <TableHead className="w-20">Status</TableHead>
          <TableHead>Quality</TableHead>
          {isMultiVersionEnabled && <TableHead>Slot</TableHead>}
          <TableHead className="w-16">Actions</TableHead>
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
            isSearching={searchingEpisodeId === episode.id}
            searchingSlotId={searchingSlotId}
            isMultiVersionEnabled={isMultiVersionEnabled}
            enabledSlots={enabledSlots}
            isAssigning={isAssigning}
            getSlotName={getSlotName}
            onAutoSearch={onAutoSearch}
            onManualSearch={onManualSearch}
            onMonitoredChange={onMonitoredChange}
            onAssignFileToSlot={onAssignFileToSlot}
            onSlotManualSearch={onSlotManualSearch}
            onSlotAutoSearch={onSlotAutoSearch}
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
  isSearching: boolean
  searchingSlotId?: number | null
  isMultiVersionEnabled: boolean
  enabledSlots: Slot[]
  isAssigning: boolean
  getSlotName: (slotId: number | undefined) => string | null
  onAutoSearch?: (episode: Episode) => void
  onManualSearch?: (episode: Episode) => void
  onMonitoredChange?: (episode: Episode, monitored: boolean) => void
  onAssignFileToSlot?: (fileId: number, episodeId: number, slotId: number) => void
  onSlotManualSearch?: (episodeId: number, slotId: number) => void
  onSlotAutoSearch?: (episodeId: number, slotId: number) => void
}

function EpisodeRow({
  episode,
  seriesId: _seriesId,
  columnCount,
  isExpanded,
  onToggleExpanded,
  isDownloading,
  isSearching,
  searchingSlotId,
  isMultiVersionEnabled,
  enabledSlots,
  isAssigning,
  getSlotName,
  onAutoSearch,
  onManualSearch,
  onMonitoredChange,
  onAssignFileToSlot,
  onSlotManualSearch,
  onSlotAutoSearch,
}: EpisodeRowProps) {
  const { data: slotStatus, isLoading: isLoadingSlotStatus } = useEpisodeSlotStatus(
    isExpanded ? episode.id : 0
  )
  const setSlotMonitoredMutation = useSetEpisodeSlotMonitored()

  const handleAutoSearch = () => {
    if (onAutoSearch) {
      onAutoSearch(episode)
    } else {
      toast.info('Automatic search not yet implemented')
    }
  }

  const handleManualSearch = () => {
    if (onManualSearch) {
      onManualSearch(episode)
    } else {
      toast.info('Manual search not yet implemented')
    }
  }

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
          <TableCell className="p-1">
            <button
              onClick={onToggleExpanded}
              className="p-1 hover:bg-muted rounded"
            >
              <ChevronDown className={cn('size-4 transition-transform', isExpanded && 'rotate-180')} />
            </button>
          </TableCell>
        )}
        <TableCell className="font-mono">{episode.episodeNumber}</TableCell>
        <TableCell className="font-medium">{episode.title}</TableCell>
        <TableCell>
          {episode.airDate ? formatDate(episode.airDate) : '-'}
        </TableCell>
        <TableCell>
          <EpisodeAvailabilityBadge released={episode.released} />
        </TableCell>
        <TableCell>
          <Switch
            checked={episode.monitored}
            onCheckedChange={(checked) => onMonitoredChange?.(episode, checked)}
            disabled={!onMonitoredChange}
          />
        </TableCell>
        <TableCell>
          {isDownloading ? (
            <Download className="size-4 text-green-500" />
          ) : episode.hasFile ? (
            <Check className="size-4 text-green-500" />
          ) : (
            <X className="size-4 text-red-500" />
          )}
        </TableCell>
        <TableCell>
          {episode.episodeFile ? (
            <QualityBadge quality={episode.episodeFile.quality} />
          ) : (
            '-'
          )}
        </TableCell>
        {isMultiVersionEnabled && (
          <TableCell>
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
        <TableCell>
          {isDownloading ? (
            <div className="flex items-center justify-center size-8">
              <Download className="size-4 text-green-500" />
            </div>
          ) : isSearching ? (
            <div className="flex items-center justify-center size-8">
              <Loader2 className="size-4 animate-spin" />
            </div>
          ) : (
            <DropdownMenu>
              <DropdownMenuTrigger className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring hover:bg-accent hover:text-accent-foreground size-8">
                <MoreHorizontal className="size-4" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={handleAutoSearch}>
                  <Zap className="size-4 mr-2" />
                  Automatic Search
                </DropdownMenuItem>
                <DropdownMenuItem onClick={handleManualSearch}>
                  <Search className="size-4 mr-2" />
                  Manual Search
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
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

