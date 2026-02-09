import { ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import { MediaSearchMonitorControls } from '@/components/search'
import { MediaStatusBadge, type MediaStatus } from '@/components/media/MediaStatusBadge'
import { EpisodeTable } from './EpisodeTable'
import type { Season, Episode, Slot, StatusCounts } from '@/types'

interface SeasonListProps {
  seriesId: number
  seriesTitle: string
  qualityProfileId: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
  seasons: Season[]
  episodes?: Episode[]
  onSeasonMonitoredChange?: (seasonNumber: number, monitored: boolean) => void
  onEpisodeMonitoredChange?: (episode: Episode, monitored: boolean) => void
  onAssignFileToSlot?: (fileId: number, episodeId: number, slotId: number) => void
  isMultiVersionEnabled?: boolean
  enabledSlots?: Slot[]
  isAssigning?: boolean
  episodeRatings?: Record<number, Record<number, number>>
  className?: string
}

function computeSeasonStatus(counts: StatusCounts): MediaStatus {
  if (counts.downloading > 0) return 'downloading'
  if (counts.failed > 0) return 'failed'
  if (counts.missing > 0) return 'missing'
  if (counts.upgradable > 0) return 'upgradable'
  if (counts.available > 0) return 'available'
  return 'unreleased'
}

export function SeasonList({
  seriesId,
  seriesTitle,
  qualityProfileId,
  tvdbId,
  tmdbId,
  imdbId,
  seasons,
  episodes = [],
  onSeasonMonitoredChange,
  onEpisodeMonitoredChange,
  onAssignFileToSlot,
  isMultiVersionEnabled = false,
  enabledSlots = [],
  isAssigning = false,
  episodeRatings,
  className,
}: SeasonListProps) {
  // Group episodes by season
  const episodesBySeason: Record<number, Episode[]> = {}
  episodes.forEach((ep) => {
    if (!episodesBySeason[ep.seasonNumber]) {
      episodesBySeason[ep.seasonNumber] = []
    }
    episodesBySeason[ep.seasonNumber].push(ep)
  })

  // Sort seasons by number, with specials (season 0) at the bottom
  const sortedSeasons = [...seasons].sort((a, b) => {
    if (a.seasonNumber === 0) return 1
    if (b.seasonNumber === 0) return -1
    return a.seasonNumber - b.seasonNumber
  })

  return (
    <Accordion className={cn('space-y-2', className)}>
      {sortedSeasons.map((season) => {
        const seasonEpisodes = episodesBySeason[season.seasonNumber] || []
        const fileCount = season.statusCounts.available + season.statusCounts.upgradable
        const totalCount = season.statusCounts.total - season.statusCounts.unreleased
        const seasonLabel = season.seasonNumber === 0 ? 'Specials' : `Season ${season.seasonNumber}`
        const firstAirYear = seasonEpisodes
          .filter((ep) => ep.airDate)
          .sort((a, b) => new Date(a.airDate!).getTime() - new Date(b.airDate!).getTime())[0]
          ?.airDate?.slice(0, 4)

        return (
          <AccordionItem
            key={season.id}
            value={`season-${season.seasonNumber}`}
            className="border rounded-lg px-4"
          >
            <AccordionTrigger className="group hover:no-underline py-3 **:data-[slot=accordion-trigger-icon]:!hidden">
              <div className="flex items-center gap-4 flex-1">
                <ChevronRight className="size-4 shrink-0 transition-transform duration-200 text-muted-foreground group-aria-expanded/accordion-trigger:rotate-90 group-hover:text-tv-400 group-hover:icon-glow-tv" />
                {season.posterUrl && (
                  <img
                    src={season.posterUrl}
                    alt={seasonLabel}
                    className="w-10 h-14 object-cover rounded shrink-0"
                  />
                )}
                <span className="font-semibold">
                  {seasonLabel}
                  {firstAirYear && season.seasonNumber > 0 && <span className="ml-1.5 text-muted-foreground font-normal">({firstAirYear})</span>}
                </span>
                <Badge variant={fileCount === totalCount && totalCount > 0 ? 'default' : 'secondary'}>
                  {fileCount}/{totalCount}
                </Badge>
                <MediaStatusBadge status={computeSeasonStatus(season.statusCounts)} />
                <div className="ml-auto flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
                  <MediaSearchMonitorControls
                    mediaType="season"
                    seriesId={seriesId}
                    seriesTitle={seriesTitle}
                    seasonNumber={season.seasonNumber}
                    title={seasonLabel}
                    theme="tv"
                    size="sm"
                    monitored={season.monitored}
                    onMonitoredChange={(m) => onSeasonMonitoredChange?.(season.seasonNumber, m)}
                    monitorDisabled={!onSeasonMonitoredChange}
                    qualityProfileId={qualityProfileId}
                    tvdbId={tvdbId}
                    tmdbId={tmdbId}
                    imdbId={imdbId}
                  />
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
                  seriesTitle={seriesTitle}
                  qualityProfileId={qualityProfileId}
                  tvdbId={tvdbId}
                  tmdbId={tmdbId}
                  imdbId={imdbId}
                  episodes={seasonEpisodes}
                  onMonitoredChange={onEpisodeMonitoredChange}
                  onAssignFileToSlot={onAssignFileToSlot}
                  isMultiVersionEnabled={isMultiVersionEnabled}
                  enabledSlots={enabledSlots}
                  isAssigning={isAssigning}
                  episodeRatings={episodeRatings?.[season.seasonNumber]}
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
