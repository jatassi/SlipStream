import { Search, Zap, Download, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { SeasonAvailabilityBadge } from '@/components/media/AvailabilityBadge'
import { EpisodeTable } from './EpisodeTable'
import { useDownloadingStore } from '@/stores'
import type { Season, Episode, Slot } from '@/types'

interface SeasonListProps {
  seriesId: number
  seasons: Season[]
  episodes?: Episode[]
  onSeasonMonitoredChange?: (seasonNumber: number, monitored: boolean) => void
  onSeasonSearch?: (seasonNumber: number) => void
  onSeasonAutoSearch?: (seasonNumber: number) => void
  onEpisodeSearch?: (episode: Episode) => void
  onEpisodeAutoSearch?: (episode: Episode) => void
  onEpisodeMonitoredChange?: (episode: Episode, monitored: boolean) => void
  onAssignFileToSlot?: (fileId: number, episodeId: number, slotId: number) => void
  onSlotManualSearch?: (episodeId: number, slotId: number) => void
  onSlotAutoSearch?: (episodeId: number, slotId: number) => void
  searchingSeasonNumber?: number | null
  searchingEpisodeId?: number | null
  searchingSlotId?: number | null
  isMultiVersionEnabled?: boolean
  enabledSlots?: Slot[]
  isAssigning?: boolean
  className?: string
}

export function SeasonList({
  seriesId,
  seasons,
  episodes = [],
  onSeasonMonitoredChange,
  onSeasonSearch,
  onSeasonAutoSearch,
  onEpisodeSearch,
  onEpisodeAutoSearch,
  onEpisodeMonitoredChange,
  onAssignFileToSlot,
  onSlotManualSearch,
  onSlotAutoSearch,
  searchingSeasonNumber,
  searchingEpisodeId,
  searchingSlotId,
  isMultiVersionEnabled = false,
  enabledSlots = [],
  isAssigning = false,
  className,
}: SeasonListProps) {
  // Select queueItems directly so component re-renders when it changes
  const queueItems = useDownloadingStore((state) => state.queueItems)

  const isSeasonDownloading = (sId: number, seasonNum: number) => {
    return queueItems.some(
      (item) =>
        item.seriesId === sId &&
        ((item.seasonNumber === seasonNum && item.isSeasonPack) ||
          item.isCompleteSeries) &&
        (item.status === 'downloading' || item.status === 'queued')
    )
  }
  // Group episodes by season
  const episodesBySeason: Record<number, Episode[]> = {}
  episodes.forEach((ep) => {
    if (!episodesBySeason[ep.seasonNumber]) {
      episodesBySeason[ep.seasonNumber] = []
    }
    episodesBySeason[ep.seasonNumber].push(ep)
  })

  // Sort seasons by number
  const sortedSeasons = [...seasons].sort((a, b) => a.seasonNumber - b.seasonNumber)

  return (
    <Accordion className={cn('space-y-2', className)}>
      {sortedSeasons.map((season) => {
        const seasonEpisodes = episodesBySeason[season.seasonNumber] || []
        const fileCount = seasonEpisodes.filter((e) => e.hasFile).length
        const totalCount = seasonEpisodes.length

        return (
          <AccordionItem
            key={season.id}
            value={`season-${season.seasonNumber}`}
            className="border rounded-lg px-4"
          >
            <AccordionTrigger className="hover:no-underline py-3">
              <div className="flex items-center gap-4 flex-1">
                {season.posterUrl && (
                  <img
                    src={season.posterUrl}
                    alt={`Season ${season.seasonNumber}`}
                    className="w-10 h-14 object-cover rounded shrink-0"
                  />
                )}
                <span className="font-semibold">
                  {season.seasonNumber === 0 ? 'Specials' : `Season ${season.seasonNumber}`}
                </span>
                <Badge variant={fileCount === totalCount && totalCount > 0 ? 'default' : 'secondary'}>
                  {fileCount}/{totalCount}
                </Badge>
                <SeasonAvailabilityBadge season={season} />
                <div className="ml-auto flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
                  {onSeasonSearch && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => onSeasonSearch(season.seasonNumber)}
                    >
                      <Search className="size-4" />
                    </Button>
                  )}
                  {onSeasonAutoSearch && (
                    isSeasonDownloading(seriesId, season.seasonNumber) ? (
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled
                        title="Downloading"
                      >
                        <Download className="size-4 text-green-500" />
                      </Button>
                    ) : searchingSeasonNumber === season.seasonNumber ? (
                      <Button
                        variant="ghost"
                        size="sm"
                        disabled
                        title="Searching..."
                      >
                        <Loader2 className="size-4 animate-spin" />
                      </Button>
                    ) : (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => onSeasonAutoSearch(season.seasonNumber)}
                      >
                        <Zap className="size-4" />
                      </Button>
                    )
                  )}
                  {onSeasonMonitoredChange && (
                    <Switch
                      checked={season.monitored}
                      onCheckedChange={(checked) =>
                        onSeasonMonitoredChange(season.seasonNumber, checked)
                      }
                    />
                  )}
                </div>
              </div>
            </AccordionTrigger>
            <AccordionContent className="pb-4">
              {season.overview && (
                <p className="text-sm text-muted-foreground mb-4 line-clamp-2">
                  {season.overview}
                </p>
              )}
              {seasonEpisodes.length > 0 ? (
                <EpisodeTable
                  seriesId={seriesId}
                  episodes={seasonEpisodes}
                  onManualSearch={onEpisodeSearch}
                  onAutoSearch={onEpisodeAutoSearch}
                  onMonitoredChange={onEpisodeMonitoredChange}
                  onAssignFileToSlot={onAssignFileToSlot}
                  onSlotManualSearch={onSlotManualSearch}
                  onSlotAutoSearch={onSlotAutoSearch}
                  searchingEpisodeId={searchingEpisodeId}
                  searchingSlotId={searchingSlotId}
                  isMultiVersionEnabled={isMultiVersionEnabled}
                  enabledSlots={enabledSlots}
                  isAssigning={isAssigning}
                />
              ) : (
                <p className="text-sm text-muted-foreground py-2">
                  No episodes found
                </p>
              )}
            </AccordionContent>
          </AccordionItem>
        )
      })}
    </Accordion>
  )
}
