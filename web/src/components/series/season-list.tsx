import { ChevronRight } from 'lucide-react'

import { type MediaStatus, MediaStatusBadge } from '@/components/media/media-status-badge'
import { MediaSearchMonitorControls } from '@/components/search'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { Episode, Season, Slot, StatusCounts } from '@/types'

import { EpisodeTable } from './episode-table'

type SeriesContext = {
  seriesId: number
  seriesTitle: string
  qualityProfileId: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
}

type SeasonListProps = SeriesContext & {
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

type SeasonItemProps = SeriesContext & {
  season: Season
  seasonEpisodes: Episode[]
  onSeasonMonitoredChange?: (seasonNumber: number, monitored: boolean) => void
  onEpisodeMonitoredChange?: (episode: Episode, monitored: boolean) => void
  onAssignFileToSlot?: (fileId: number, episodeId: number, slotId: number) => void
  isMultiVersionEnabled: boolean
  enabledSlots: Slot[]
  isAssigning: boolean
  seasonEpisodeRatings?: Record<number, number>
}

function computeSeasonStatus(counts: StatusCounts): MediaStatus {
  if (counts.downloading > 0) {
    return 'downloading'
  }
  if (counts.failed > 0) {
    return 'failed'
  }
  if (counts.missing > 0) {
    return 'missing'
  }
  if (counts.upgradable > 0) {
    return 'upgradable'
  }
  if (counts.available > 0) {
    return 'available'
  }
  return 'unreleased'
}

function groupEpisodesBySeason(episodes: Episode[]): Partial<Record<number, Episode[]>> {
  const map: Partial<Record<number, Episode[]>> = {}
  for (const ep of episodes) {
    map[ep.seasonNumber] ??= []
    map[ep.seasonNumber]?.push(ep)
  }
  return map
}

function sortSeasons(seasons: Season[]): Season[] {
  return seasons.toSorted((a, b) => {
    if (a.seasonNumber === 0) {
      return 1
    }
    if (b.seasonNumber === 0) {
      return -1
    }
    return a.seasonNumber - b.seasonNumber
  })
}

function getFirstAirYear(episodes: Episode[]): string | undefined {
  return episodes
    .filter((ep) => ep.airDate)
    .toSorted(
      (a, b) => new Date(a.airDate ?? 0).getTime() - new Date(b.airDate ?? 0).getTime(),
    )[0]
    ?.airDate?.slice(0, 4)
}

export function SeasonList(props: SeasonListProps) {
  const { seasons, episodes = [], episodeRatings, className } = props
  const { isMultiVersionEnabled = false, enabledSlots = [], isAssigning = false } = props
  const episodesBySeason = groupEpisodesBySeason(episodes)

  return (
    <Accordion className={cn('space-y-2', className)}>
      {sortSeasons(seasons).map((season) => (
        <SeasonItem
          key={season.id}
          season={season}
          seasonEpisodes={episodesBySeason[season.seasonNumber] ?? []}
          seriesId={props.seriesId}
          seriesTitle={props.seriesTitle}
          qualityProfileId={props.qualityProfileId}
          tvdbId={props.tvdbId}
          tmdbId={props.tmdbId}
          imdbId={props.imdbId}
          onSeasonMonitoredChange={props.onSeasonMonitoredChange}
          onEpisodeMonitoredChange={props.onEpisodeMonitoredChange}
          onAssignFileToSlot={props.onAssignFileToSlot}
          isMultiVersionEnabled={isMultiVersionEnabled}
          enabledSlots={enabledSlots}
          isAssigning={isAssigning}
          seasonEpisodeRatings={episodeRatings?.[season.seasonNumber]}
        />
      ))}
    </Accordion>
  )
}

function SeasonItem(props: SeasonItemProps) {
  const { season, seasonEpisodes, onSeasonMonitoredChange } = props
  const fileCount = season.statusCounts.available + season.statusCounts.upgradable
  const totalCount = season.statusCounts.total - season.statusCounts.unreleased
  const seasonLabel = season.seasonNumber === 0 ? 'Specials' : `Season ${season.seasonNumber}`

  return (
    <AccordionItem value={`season-${season.seasonNumber}`} className="rounded-lg border px-4">
      <SeasonTrigger
        season={season}
        seasonLabel={seasonLabel}
        firstAirYear={getFirstAirYear(seasonEpisodes)}
        fileCount={fileCount}
        totalCount={totalCount}
        seriesId={props.seriesId}
        seriesTitle={props.seriesTitle}
        qualityProfileId={props.qualityProfileId}
        tvdbId={props.tvdbId}
        tmdbId={props.tmdbId}
        imdbId={props.imdbId}
        onSeasonMonitoredChange={onSeasonMonitoredChange}
      />
      <SeasonContent {...props} seasonLabel={seasonLabel} />
    </AccordionItem>
  )
}

function SeasonContent(props: SeasonItemProps & { seasonLabel: string }) {
  const { season, seasonEpisodes, seasonEpisodeRatings } = props

  return (
    <AccordionContent className="pb-4">
      {season.overview ? (
        <p className="text-muted-foreground mb-4 line-clamp-2 text-sm">{season.overview}</p>
      ) : null}
      {seasonEpisodes.length > 0 ? (
        <EpisodeTable
          seriesId={props.seriesId}
          seriesTitle={props.seriesTitle}
          qualityProfileId={props.qualityProfileId}
          tvdbId={props.tvdbId}
          tmdbId={props.tmdbId}
          imdbId={props.imdbId}
          episodes={seasonEpisodes}
          onMonitoredChange={props.onEpisodeMonitoredChange}
          onAssignFileToSlot={props.onAssignFileToSlot}
          isMultiVersionEnabled={props.isMultiVersionEnabled}
          enabledSlots={props.enabledSlots}
          isAssigning={props.isAssigning}
          episodeRatings={seasonEpisodeRatings}
        />
      ) : (
        <p className="text-muted-foreground py-2 text-sm">No episodes found</p>
      )}
    </AccordionContent>
  )
}

type SeasonTriggerProps = SeriesContext & {
  season: Season
  seasonLabel: string
  firstAirYear?: string
  fileCount: number
  totalCount: number
  onSeasonMonitoredChange?: (seasonNumber: number, monitored: boolean) => void
}

function SeasonTrigger(props: SeasonTriggerProps) {
  const { season, seasonLabel, firstAirYear, fileCount, totalCount } = props

  return (
    <AccordionTrigger className="group py-3 hover:no-underline **:data-[slot=accordion-trigger-icon]:!hidden">
      <div className="flex flex-1 items-center gap-4">
        <ChevronRight className="text-muted-foreground group-hover:text-tv-400 group-hover:icon-glow-tv size-4 shrink-0 transition-transform duration-200 group-aria-expanded/accordion-trigger:rotate-90" />
        {season.posterUrl ? (
          <img src={season.posterUrl} alt={seasonLabel} className="h-14 w-10 shrink-0 rounded object-cover" />
        ) : null}
        <SeasonLabel label={seasonLabel} firstAirYear={firstAirYear} seasonNumber={season.seasonNumber} />
        <Badge variant={fileCount === totalCount && totalCount > 0 ? 'default' : 'secondary'}>
          {fileCount}/{totalCount}
        </Badge>
        <MediaStatusBadge status={computeSeasonStatus(season.statusCounts)} />
        <button type="button" className="ml-auto flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
          <MediaSearchMonitorControls
            mediaType="season"
            seriesId={props.seriesId}
            seriesTitle={props.seriesTitle}
            seasonNumber={season.seasonNumber}
            title={seasonLabel}
            theme="tv"
            size="sm"
            monitored={season.monitored}
            onMonitoredChange={(m) => props.onSeasonMonitoredChange?.(season.seasonNumber, m)}
            monitorDisabled={!props.onSeasonMonitoredChange}
            qualityProfileId={props.qualityProfileId}
            tvdbId={props.tvdbId}
            tmdbId={props.tmdbId}
            imdbId={props.imdbId}
          />
        </button>
      </div>
    </AccordionTrigger>
  )
}

function SeasonLabel({ label, firstAirYear, seasonNumber }: {
  label: string
  firstAirYear?: string
  seasonNumber: number
}) {
  return (
    <span className="font-semibold">
      {label}
      {firstAirYear && seasonNumber > 0 ? (
        <span className="text-muted-foreground ml-1.5 font-normal">({firstAirYear})</span>
      ) : null}
    </span>
  )
}
