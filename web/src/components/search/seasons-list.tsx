import { Check, Circle, Minus } from 'lucide-react'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Badge } from '@/components/ui/badge'
import type { EnrichedEpisode, EnrichedSeason, EpisodeResult, SeasonResult } from '@/types'

const isFutureDate = (dateStr?: string) => {
  if (!dateStr) {
    return false
  }
  return new Date(dateStr) > new Date()
}

type SeasonsListProps = {
  seasons: SeasonResult[]
  enrichedSeasons?: EnrichedSeason[]
}

export function SeasonsList({ seasons, enrichedSeasons }: SeasonsListProps) {
  const regularSeasons = seasons.filter((s) => s.seasonNumber > 0)

  if (regularSeasons.length === 0) {
    return null
  }

  const enrichedMap = new Map(enrichedSeasons?.map((s) => [s.seasonNumber, s]))

  return (
    <div>
      <h3 className="mb-2 text-sm font-semibold">
        {regularSeasons.length} {regularSeasons.length === 1 ? 'Season' : 'Seasons'}
      </h3>
      <Accordion>
        {regularSeasons.map((season) => (
          <SeasonItem
            key={season.seasonNumber}
            season={season}
            enriched={enrichedMap.get(season.seasonNumber)}
          />
        ))}
      </Accordion>
    </div>
  )
}

function SeasonStatusBadge({ enriched }: { enriched: EnrichedSeason }) {
  if (enriched.available) {
    return (
      <Badge className="bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 text-xs">
        Available
      </Badge>
    )
  }
  if (enriched.existingRequestId) {
    return (
      <Badge className="bg-blue-500/15 text-blue-600 dark:text-blue-400 text-xs">Requested</Badge>
    )
  }
  if (enriched.airedEpisodesWithFiles > 0) {
    return (
      <Badge className="bg-orange-500/15 text-orange-600 dark:text-orange-400 text-xs">
        Partial ({enriched.airedEpisodesWithFiles}/{enriched.totalAiredEpisodes})
      </Badge>
    )
  }
  if (enriched.inLibrary) {
    return (
      <Badge variant="secondary" className="text-xs">
        Missing
      </Badge>
    )
  }
  return null
}

function SeasonItem({ season, enriched }: { season: SeasonResult; enriched?: EnrichedSeason }) {
  const firstEpisode = season.episodes?.[0]
  const seasonIsFuture = firstEpisode ? isFutureDate(firstEpisode.airDate) : false

  return (
    <AccordionItem value={`season-${season.seasonNumber}`}>
      <AccordionTrigger>
        <div className="flex items-center gap-2">
          <span className={seasonIsFuture ? 'text-muted-foreground' : ''}>
            {season.name || `Season ${season.seasonNumber}`}
          </span>
          {enriched ? <SeasonStatusBadge enriched={enriched} /> : null}
          {season.episodes ? (
            <Badge variant="secondary" className="text-xs">
              {season.episodes.length} episodes
            </Badge>
          ) : null}
        </div>
      </AccordionTrigger>
      <AccordionContent>
        {season.overview ? (
          <p className="text-muted-foreground mb-2 text-sm">{season.overview}</p>
        ) : null}
        <EpisodeList episodes={season.episodes} enrichedEpisodes={enriched?.episodes} />
      </AccordionContent>
    </AccordionItem>
  )
}

function EpisodeList({
  episodes,
  enrichedEpisodes,
}: {
  episodes?: EpisodeResult[]
  enrichedEpisodes?: EnrichedEpisode[]
}) {
  if (!episodes || episodes.length === 0) {
    return null
  }

  const enrichedMap = new Map(enrichedEpisodes?.map((e) => [e.episodeNumber, e]))

  return (
    <div className="space-y-1">
      {episodes.map((ep) => (
        <EpisodeRow key={ep.episodeNumber} episode={ep} enriched={enrichedMap.get(ep.episodeNumber)} />
      ))}
    </div>
  )
}

function EpisodeRow({ episode, enriched }: { episode: EpisodeResult; enriched?: EnrichedEpisode }) {
  const isFuture = isFutureDate(episode.airDate)
  return (
    <div className={`flex items-center gap-2 text-sm ${isFuture ? 'text-muted-foreground' : ''}`}>
      {enriched ? <EpisodeStatusIcon hasFile={enriched.hasFile} aired={enriched.aired} /> : null}
      <span className={isFuture ? '' : 'text-muted-foreground'}>E{episode.episodeNumber}</span>
      <span className="truncate">{episode.title}</span>
      {episode.airDate ? (
        <span className="text-muted-foreground ml-auto shrink-0 text-xs">{episode.airDate}</span>
      ) : null}
    </div>
  )
}

function EpisodeStatusIcon({ hasFile, aired }: { hasFile: boolean; aired: boolean }) {
  if (hasFile) {
    return <Check className="size-3.5 shrink-0 text-emerald-500" />
  }
  if (aired) {
    return <Minus className="size-3.5 shrink-0 text-orange-500" />
  }
  return <Circle className="text-muted-foreground/50 size-3.5 shrink-0" />
}
