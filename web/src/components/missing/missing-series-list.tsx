import { Link } from '@tanstack/react-router'
import { Calendar, ChevronRight, SlidersVertical, UserSearch } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import { PosterImage } from '@/components/media/poster-image'
import { MediaSearchMonitorControls } from '@/components/search'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import { formatDate } from '@/lib/formatters'
import { cn } from '@/lib/utils'
import type { MissingEpisode, MissingSeason, MissingSeries } from '@/types/missing'

import type { EpisodeMonitoredParams } from './use-missing-series-list'
import { useMissingSeriesList } from './use-missing-series-list'

type MissingSeriesListProps = {
  series: MissingSeries[]
  qualityProfileNames: Map<number, string>
}

export function MissingSeriesList({ series, qualityProfileNames }: MissingSeriesListProps) {
  const actions = useMissingSeriesList()

  if (series.length === 0) {
    return (
      <EmptyState
        icon={<UserSearch className="text-tv-400 size-8" />}
        title="No missing episodes"
        description="All monitored episodes that have aired have been downloaded"
        className="py-8"
      />
    )
  }

  return (
    <Accordion className="space-y-2">
      {series.map((s) => (
        <MissingSeriesItem
          key={s.id}
          series={s}
          qualityProfileNames={qualityProfileNames}
          actions={actions}
        />
      ))}
    </Accordion>
  )
}

type MissingSeriesActions = ReturnType<typeof useMissingSeriesList>

type MissingSeriesItemProps = {
  series: MissingSeries
  qualityProfileNames: Map<number, string>
  actions: MissingSeriesActions
}

function MissingSeriesItem({ series: s, qualityProfileNames, actions }: MissingSeriesItemProps) {
  return (
    <AccordionItem
      value={`series-${s.id}`}
      className="bg-card data-open:border-tv-500/30 rounded-lg border px-4 transition-colors"
    >
      <AccordionTrigger className="group py-3 hover:no-underline **:data-[slot=accordion-trigger-icon]:!hidden">
        <SeriesTriggerContent series={s} qualityProfileNames={qualityProfileNames} actions={actions} />
      </AccordionTrigger>
      <AccordionContent className="pb-4">
        <Accordion className="space-y-1">
          {s.missingSeasons.map((season) => (
            <MissingSeasonItem
              key={`${s.id}-${season.seasonNumber}`}
              series={s}
              season={season}
              actions={actions}
            />
          ))}
        </Accordion>
      </AccordionContent>
    </AccordionItem>
  )
}

type SeriesTriggerContentProps = {
  series: MissingSeries
  qualityProfileNames: Map<number, string>
  actions: MissingSeriesActions
}

function SeriesTriggerContent({ series: s, qualityProfileNames, actions }: SeriesTriggerContentProps) {
  return (
    <div className="flex flex-1 flex-wrap items-center gap-x-4 gap-y-1 sm:flex-nowrap">
      <ChevronRight className="text-muted-foreground group-hover:text-tv-400 group-hover:icon-glow-tv size-4 shrink-0 transition-transform duration-200 group-aria-expanded/accordion-trigger:rotate-90" />
      <SeriesInfo series={s} profileName={qualityProfileNames.get(s.qualityProfileId)} />
      <button
        type="button"
        className="ml-auto flex items-center gap-2"
        onClick={(e) => e.stopPropagation()}
      >
        <MediaSearchMonitorControls
          mediaType="series"
          seriesId={s.id}
          title={s.title}
          theme="tv"
          size="sm"
          monitored
          onMonitoredChange={(m) => actions.handleSeriesMonitored(s, m)}
          monitorDisabled={actions.isSeriesPending}
          qualityProfileId={s.qualityProfileId}
          tvdbId={s.tvdbId}
          tmdbId={s.tmdbId}
          imdbId={s.imdbId}
        />
      </button>
    </div>
  )
}

type SeriesInfoProps = {
  series: MissingSeries
  profileName: string | undefined
}

function SeriesInfo({ series: s, profileName }: SeriesInfoProps) {
  return (
    <>
      <Link
        to="/series/$id"
        params={{ id: s.id.toString() }}
        className="hidden shrink-0 sm:block"
        onClick={(e) => e.stopPropagation()}
      >
        <PosterImage
          tmdbId={s.tmdbId}
          tvdbId={s.tvdbId}
          alt={s.title}
          type="series"
          className="h-14 w-10 shrink-0 rounded"
        />
      </Link>
      <div className="min-w-0 flex-1 sm:flex-initial">
        <div className="flex items-baseline gap-2">
          <Link
            to="/series/$id"
            params={{ id: s.id.toString() }}
            className="hover:text-tv-400 font-semibold transition-colors sm:line-clamp-1"
            onClick={(e) => e.stopPropagation()}
          >
            {s.title}
          </Link>
          {s.year ? (
            <span className="text-muted-foreground shrink-0 text-xs">({s.year})</span>
          ) : null}
        </div>
        <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1">
          <Badge variant="secondary">{s.missingCount} missing</Badge>
          {profileName ? (
            <Badge variant="secondary" className="gap-1 px-1.5 py-0 text-[10px]">
              <SlidersVertical className="size-2.5" />
              {profileName}
            </Badge>
          ) : null}
        </div>
      </div>
    </>
  )
}

type MissingSeasonItemProps = {
  series: MissingSeries
  season: MissingSeason
  actions: MissingSeriesActions
}

function MissingSeasonItem({ series: s, season, actions }: MissingSeasonItemProps) {
  return (
    <AccordionItem
      value={`season-${s.id}-${season.seasonNumber}`}
      className="rounded-lg border px-3"
    >
      <SeasonTrigger series={s} season={season} actions={actions} />
      <AccordionContent className="pb-2">
        <div className="space-y-0.5">
          {season.missingEpisodes
            .toSorted((a, b) => a.episodeNumber - b.episodeNumber)
            .map((episode) => (
              <MissingEpisodeRow
                key={episode.id}
                series={s}
                episode={episode}
                onMonitored={actions.handleEpisodeMonitored}
                isDisabled={actions.isEpisodePending}
              />
            ))}
        </div>
      </AccordionContent>
    </AccordionItem>
  )
}

type SeasonTriggerProps = {
  series: MissingSeries
  season: MissingSeason
  actions: MissingSeriesActions
}

function SeasonTrigger({ series: s, season, actions }: SeasonTriggerProps) {
  return (
    <AccordionTrigger className="group py-2 hover:no-underline **:data-[slot=accordion-trigger-icon]:!hidden">
      <div className="flex flex-1 items-center gap-3">
        <ChevronRight className="text-muted-foreground group-hover:text-tv-400 size-3.5 shrink-0 transition-transform duration-200 group-aria-expanded/accordion-trigger:rotate-90" />
        <span className="text-sm font-medium">Season {season.seasonNumber}</span>
        <Badge variant="secondary" className="text-xs">
          {season.missingEpisodes.length} missing
        </Badge>
        <button
          type="button"
          className="ml-auto flex items-center gap-1"
          onClick={(e) => e.stopPropagation()}
        >
          <MediaSearchMonitorControls
            mediaType="season"
            seriesId={s.id}
            seriesTitle={s.title}
            seasonNumber={season.seasonNumber}
            title={`${s.title} Season ${season.seasonNumber}`}
            theme="tv"
            size="xs"
            monitored
            onMonitoredChange={(m) => actions.handleSeasonMonitored(s.id, season.seasonNumber, m)}
            monitorDisabled={actions.isSeasonPending}
            qualityProfileId={s.qualityProfileId}
            tvdbId={s.tvdbId}
            tmdbId={s.tmdbId}
            imdbId={s.imdbId}
          />
        </button>
      </div>
    </AccordionTrigger>
  )
}

type MissingEpisodeRowProps = {
  series: MissingSeries
  episode: MissingEpisode
  onMonitored: (params: EpisodeMonitoredParams) => void
  isDisabled: boolean
}

function MissingEpisodeRow({ series: s, episode, onMonitored, isDisabled }: MissingEpisodeRowProps) {
  const epLabel = `S${episode.seasonNumber.toString().padStart(2, '0')}E${episode.episodeNumber.toString().padStart(2, '0')}`

  return (
    <div className="hover:bg-muted/50 flex items-center justify-between rounded px-1 py-1.5 text-sm">
      <div className="flex min-w-0 items-center gap-2">
        <span className="text-tv-400 w-6 shrink-0 text-center font-mono text-xs">
          {episode.episodeNumber.toString().padStart(2, '0')}
        </span>
        <span
          className={cn('truncate', !episode.title && 'text-muted-foreground italic')}
          title={episode.title || 'TBA'}
        >
          {episode.title || 'TBA'}
        </span>
        {episode.airDate ? (
          <span className="text-muted-foreground flex shrink-0 items-center gap-1 text-xs">
            <Calendar className="size-3" />
            {formatDate(episode.airDate)}
          </span>
        ) : null}
      </div>

      <div className="shrink-0">
        <MediaSearchMonitorControls
          mediaType="episode"
          episodeId={episode.id}
          seriesId={s.id}
          seriesTitle={s.title}
          seasonNumber={episode.seasonNumber}
          episodeNumber={episode.episodeNumber}
          title={epLabel}
          theme="tv"
          size="xs"
          monitored
          onMonitoredChange={(m) =>
            onMonitored({ seriesId: s.id, episodeId: episode.id, label: epLabel, monitored: m })
          }
          monitorDisabled={isDisabled}
          qualityProfileId={s.qualityProfileId}
          tvdbId={s.tvdbId}
          tmdbId={s.tmdbId}
          imdbId={s.imdbId}
        />
      </div>
    </div>
  )
}
