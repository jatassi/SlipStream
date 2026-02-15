import { Link } from '@tanstack/react-router'
import { ChevronRight, SlidersVertical } from 'lucide-react'

import { PosterImage } from '@/components/media/poster-image'
import { MediaSearchMonitorControls } from '@/components/search'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import type { UpgradableSeries } from '@/types/missing'
import type { QualityProfile } from '@/types/quality-profile'

import { UpgradableSeasonItem } from './upgradable-season-item'
import type { EpisodeMonitoredParams } from './use-upgradable-series-list'

type UpgradableSeriesItemProps = {
  series: UpgradableSeries
  profile?: QualityProfile
  onSeriesMonitored: (s: UpgradableSeries, monitored: boolean) => void
  onSeasonMonitored: (seriesId: number, seasonNumber: number, monitored: boolean) => void
  onEpisodeMonitored: (params: EpisodeMonitoredParams) => void
  isSeriesPending: boolean
  isSeasonPending: boolean
  isEpisodePending: boolean
}

export function UpgradableSeriesItem({
  series,
  profile,
  onSeriesMonitored,
  onSeasonMonitored,
  onEpisodeMonitored,
  isSeriesPending,
  isSeasonPending,
  isEpisodePending,
}: UpgradableSeriesItemProps) {
  return (
    <AccordionItem
      value={`series-${series.id}`}
      className="bg-card data-open:border-tv-500/30 rounded-lg border px-4 transition-colors"
    >
      <SeriesTrigger series={series} profile={profile} onSeriesMonitored={onSeriesMonitored} isSeriesPending={isSeriesPending} />
      <AccordionContent className="pb-4">
        <Accordion className="space-y-1">
          {series.upgradableSeasons.map((season) => (
            <UpgradableSeasonItem
              key={`${series.id}-${season.seasonNumber}`}
              series={series}
              season={season}
              profile={profile}
              onSeasonMonitored={onSeasonMonitored}
              onEpisodeMonitored={onEpisodeMonitored}
              isSeasonDisabled={isSeasonPending}
              isEpisodeDisabled={isEpisodePending}
            />
          ))}
        </Accordion>
      </AccordionContent>
    </AccordionItem>
  )
}

type SeriesTriggerProps = {
  series: UpgradableSeries
  profile?: QualityProfile
  onSeriesMonitored: (s: UpgradableSeries, monitored: boolean) => void
  isSeriesPending: boolean
}

function SeriesTrigger({ series, profile, onSeriesMonitored, isSeriesPending }: SeriesTriggerProps) {
  return (
    <AccordionTrigger className="group py-3 hover:no-underline **:data-[slot=accordion-trigger-icon]:!hidden">
      <div className="flex flex-1 flex-wrap items-center gap-x-4 gap-y-1 sm:flex-nowrap">
        <ChevronRight className="text-muted-foreground group-hover:text-tv-400 group-hover:icon-glow-tv size-4 shrink-0 transition-transform duration-200 group-aria-expanded/accordion-trigger:rotate-90" />
        <SeriesInfo series={series} profile={profile} />
        <button
          type="button"
          className="ml-auto flex items-center gap-2"
          onClick={(e) => e.stopPropagation()}
        >
          <MediaSearchMonitorControls
            mediaType="series"
            seriesId={series.id}
            title={series.title}
            theme="tv"
            size="sm"
            monitored
            onMonitoredChange={(m) => onSeriesMonitored(series, m)}
            monitorDisabled={isSeriesPending}
            qualityProfileId={series.qualityProfileId}
            tvdbId={series.tvdbId}
            tmdbId={series.tmdbId}
            imdbId={series.imdbId}
          />
        </button>
      </div>
    </AccordionTrigger>
  )
}

type SeriesInfoProps = {
  series: UpgradableSeries
  profile?: QualityProfile
}

const stopPropagation = (e: React.MouseEvent) => e.stopPropagation()

function SeriesInfo({ series, profile }: SeriesInfoProps) {
  const idStr = series.id.toString()

  return (
    <>
      <Link
        to="/series/$id"
        params={{ id: idStr }}
        className="hidden shrink-0 sm:block"
        onClick={stopPropagation}
      >
        <PosterImage
          tmdbId={series.tmdbId}
          tvdbId={series.tvdbId}
          alt={series.title}
          type="series"
          className="h-14 w-10 shrink-0 rounded"
        />
      </Link>
      <div className="min-w-0 flex-1 sm:flex-initial">
        <div className="flex items-baseline gap-2">
          <Link
            to="/series/$id"
            params={{ id: idStr }}
            className="hover:text-tv-400 font-semibold transition-colors sm:line-clamp-1"
            onClick={stopPropagation}
          >
            {series.title}
          </Link>
          {series.year ? (
            <span className="text-muted-foreground shrink-0 text-xs">({series.year})</span>
          ) : null}
        </div>
        <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1">
          <Badge variant="secondary">{series.upgradableCount} upgradable</Badge>
          {profile ? (
            <Badge variant="secondary" className="gap-1 px-1.5 py-0 text-[10px]">
              <SlidersVertical className="size-2.5" />
              {profile.name}
            </Badge>
          ) : null}
        </div>
      </div>
    </>
  )
}
