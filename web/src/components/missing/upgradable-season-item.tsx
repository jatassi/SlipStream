import { ChevronRight } from 'lucide-react'

import { MediaSearchMonitorControls } from '@/components/search'
import {
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import type { UpgradableSeason, UpgradableSeries } from '@/types/missing'
import type { QualityProfile } from '@/types/quality-profile'

import { UpgradableEpisodeRow } from './upgradable-episode-row'
import type { EpisodeMonitoredParams } from './use-upgradable-series-list'

type UpgradableSeasonItemProps = {
  series: UpgradableSeries
  season: UpgradableSeason
  profile?: QualityProfile
  onSeasonMonitored: (seriesId: number, seasonNumber: number, monitored: boolean) => void
  onEpisodeMonitored: (params: EpisodeMonitoredParams) => void
  isSeasonDisabled: boolean
  isEpisodeDisabled: boolean
}

export function UpgradableSeasonItem({
  series,
  season,
  profile,
  onSeasonMonitored,
  onEpisodeMonitored,
  isSeasonDisabled,
  isEpisodeDisabled,
}: UpgradableSeasonItemProps) {
  return (
    <AccordionItem
      value={`season-${series.id}-${season.seasonNumber}`}
      className="rounded-lg border px-3"
    >
      <SeasonTrigger series={series} season={season} onSeasonMonitored={onSeasonMonitored} isSeasonDisabled={isSeasonDisabled} />
      <AccordionContent className="pb-2">
        <div className="space-y-0.5">
          {season.upgradableEpisodes
            .toSorted((a, b) => a.episodeNumber - b.episodeNumber)
            .map((episode) => (
              <UpgradableEpisodeRow
                key={episode.id}
                series={series}
                episode={episode}
                profile={profile}
                onMonitored={onEpisodeMonitored}
                isDisabled={isEpisodeDisabled}
              />
            ))}
        </div>
      </AccordionContent>
    </AccordionItem>
  )
}

type SeasonTriggerProps = {
  series: UpgradableSeries
  season: UpgradableSeason
  onSeasonMonitored: (seriesId: number, seasonNumber: number, monitored: boolean) => void
  isSeasonDisabled: boolean
}

function SeasonTrigger({ series, season, onSeasonMonitored, isSeasonDisabled }: SeasonTriggerProps) {
  return (
    <AccordionTrigger className="group py-2 hover:no-underline **:data-[slot=accordion-trigger-icon]:!hidden">
      <div className="flex flex-1 items-center gap-3">
        <ChevronRight className="text-muted-foreground group-hover:text-tv-400 size-3.5 shrink-0 transition-transform duration-200 group-aria-expanded/accordion-trigger:rotate-90" />
        <span className="text-sm font-medium">Season {season.seasonNumber}</span>
        <Badge variant="secondary" className="text-xs">
          {season.upgradableEpisodes.length} upgradable
        </Badge>
        <button
          type="button"
          className="ml-auto flex items-center gap-1"
          onClick={(e) => e.stopPropagation()}
        >
          <MediaSearchMonitorControls
            mediaType="season"
            seriesId={series.id}
            seriesTitle={series.title}
            seasonNumber={season.seasonNumber}
            title={`${series.title} Season ${season.seasonNumber}`}
            theme="tv"
            size="xs"
            monitored
            onMonitoredChange={(m) => onSeasonMonitored(series.id, season.seasonNumber, m)}
            monitorDisabled={isSeasonDisabled}
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
