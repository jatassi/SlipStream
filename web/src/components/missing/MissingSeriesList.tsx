import { Link } from '@tanstack/react-router'
import { UserSearch, Calendar, ChevronRight } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { PosterImage } from '@/components/media/PosterImage'
import { MediaSearchMonitorControls } from '@/components/search'
import { EmptyState } from '@/components/data/EmptyState'
import { formatDate } from '@/lib/formatters'
import { cn } from '@/lib/utils'
import { useUpdateSeries, useUpdateSeasonMonitored, useUpdateEpisodeMonitored } from '@/hooks'
import { toast } from 'sonner'
import type { MissingSeries, MissingSeason, MissingEpisode } from '@/types/missing'

interface MissingSeriesListProps {
  series: MissingSeries[]
}

export function MissingSeriesList({ series }: MissingSeriesListProps) {
  const updateSeriesMutation = useUpdateSeries()
  const updateSeasonMonitoredMutation = useUpdateSeasonMonitored()
  const updateEpisodeMonitoredMutation = useUpdateEpisodeMonitored()

  const handleSeriesMonitored = async (s: MissingSeries, monitored: boolean) => {
    try {
      await updateSeriesMutation.mutateAsync({
        id: s.id,
        data: { monitored },
      })
      toast.success(monitored ? `"${s.title}" monitored` : `"${s.title}" unmonitored`)
    } catch {
      toast.error(`Failed to update "${s.title}"`)
    }
  }

  const handleSeasonMonitored = async (seriesId: number, seasonNumber: number, monitored: boolean) => {
    try {
      await updateSeasonMonitoredMutation.mutateAsync({
        seriesId,
        seasonNumber,
        monitored,
      })
      toast.success(`Season ${seasonNumber} ${monitored ? 'monitored' : 'unmonitored'}`)
    } catch {
      toast.error(`Failed to update Season ${seasonNumber}`)
    }
  }

  const handleEpisodeMonitored = async (seriesId: number, episodeId: number, label: string, monitored: boolean) => {
    try {
      await updateEpisodeMonitoredMutation.mutateAsync({
        seriesId,
        episodeId,
        monitored,
      })
      toast.success(`${label} ${monitored ? 'monitored' : 'unmonitored'}`)
    } catch {
      toast.error(`Failed to update ${label}`)
    }
  }

  if (series.length === 0) {
    return (
      <EmptyState
        icon={<UserSearch className="size-8 text-tv-400" />}
        title="No missing episodes"
        description="All monitored episodes that have aired have been downloaded"
        className="py-8"
      />
    )
  }

  return (
    <Accordion className="space-y-2">
      {series.map((s) => (
        <AccordionItem
          key={s.id}
          value={`series-${s.id}`}
          className="border rounded-lg px-4 bg-card transition-colors data-open:border-tv-500/30"
        >
          <AccordionTrigger className="group hover:no-underline py-3 **:data-[slot=accordion-trigger-icon]:!hidden">
            <div className="flex items-center gap-4 flex-1">
              <ChevronRight className="size-4 shrink-0 transition-transform duration-200 text-muted-foreground group-aria-expanded/accordion-trigger:rotate-90 group-hover:text-tv-400 group-hover:icon-glow-tv" />
              <Link
                to="/series/$id"
                params={{ id: s.id.toString() }}
                className="shrink-0"
                onClick={(e) => e.stopPropagation()}
              >
                <PosterImage
                  tmdbId={s.tmdbId}
                  tvdbId={s.tvdbId}
                  alt={s.title}
                  type="series"
                  className="w-10 h-14 rounded shrink-0"
                />
              </Link>
              <div className="min-w-0">
                <Link
                  to="/series/$id"
                  params={{ id: s.id.toString() }}
                  className="font-semibold hover:text-tv-400 transition-colors line-clamp-1"
                  onClick={(e) => e.stopPropagation()}
                >
                  {s.title}
                </Link>
                {s.year && (
                  <span className="text-xs text-muted-foreground">({s.year})</span>
                )}
              </div>
              <Badge variant="secondary">
                {s.missingCount} missing
              </Badge>
              <div className="ml-auto flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
                <MediaSearchMonitorControls
                  mediaType="series"
                  seriesId={s.id}
                  title={s.title}
                  theme="tv"
                  size="sm"
                  monitored={true}
                  onMonitoredChange={(m) => handleSeriesMonitored(s, m)}
                  monitorDisabled={updateSeriesMutation.isPending}
                  qualityProfileId={s.qualityProfileId}
                  tvdbId={s.tvdbId}
                  tmdbId={s.tmdbId}
                  imdbId={s.imdbId}
                />
              </div>
            </div>
          </AccordionTrigger>

          <AccordionContent className="pb-4">
            <Accordion className="space-y-1">
              {s.missingSeasons.map((season) => (
                <MissingSeasonItem
                  key={`${s.id}-${season.seasonNumber}`}
                  series={s}
                  season={season}
                  onSeasonMonitored={handleSeasonMonitored}
                  onEpisodeMonitored={handleEpisodeMonitored}
                  isSeasonDisabled={updateSeasonMonitoredMutation.isPending}
                  isEpisodeDisabled={updateEpisodeMonitoredMutation.isPending}
                />
              ))}
            </Accordion>
          </AccordionContent>
        </AccordionItem>
      ))}
    </Accordion>
  )
}

interface MissingSeasonItemProps {
  series: MissingSeries
  season: MissingSeason
  onSeasonMonitored: (seriesId: number, seasonNumber: number, monitored: boolean) => void
  onEpisodeMonitored: (seriesId: number, episodeId: number, label: string, monitored: boolean) => void
  isSeasonDisabled: boolean
  isEpisodeDisabled: boolean
}

function MissingSeasonItem({
  series: s,
  season,
  onSeasonMonitored,
  onEpisodeMonitored,
  isSeasonDisabled,
  isEpisodeDisabled,
}: MissingSeasonItemProps) {
  return (
    <AccordionItem
      value={`season-${s.id}-${season.seasonNumber}`}
      className="border rounded-lg px-3"
    >
      <AccordionTrigger className="group hover:no-underline py-2 **:data-[slot=accordion-trigger-icon]:!hidden">
        <div className="flex items-center gap-3 flex-1">
          <ChevronRight className="size-3.5 shrink-0 transition-transform duration-200 text-muted-foreground group-aria-expanded/accordion-trigger:rotate-90 group-hover:text-tv-400" />
          <span className="font-medium text-sm">Season {season.seasonNumber}</span>
          <Badge variant="secondary" className="text-xs">
            {season.missingEpisodes.length} missing
          </Badge>
          <div className="ml-auto flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
            <MediaSearchMonitorControls
              mediaType="season"
              seriesId={s.id}
              seriesTitle={s.title}
              seasonNumber={season.seasonNumber}
              title={`${s.title} Season ${season.seasonNumber}`}
              theme="tv"
              size="xs"
              monitored={true}
              onMonitoredChange={(m) => onSeasonMonitored(s.id, season.seasonNumber, m)}
              monitorDisabled={isSeasonDisabled}
              qualityProfileId={s.qualityProfileId}
              tvdbId={s.tvdbId}
              tmdbId={s.tmdbId}
              imdbId={s.imdbId}
            />
          </div>
        </div>
      </AccordionTrigger>

      <AccordionContent className="pb-2">
        <div className="space-y-0.5">
          {[...season.missingEpisodes]
            .sort((a, b) => a.episodeNumber - b.episodeNumber)
            .map((episode) => (
              <MissingEpisodeRow
                key={episode.id}
                series={s}
                episode={episode}
                onMonitored={onEpisodeMonitored}
                isDisabled={isEpisodeDisabled}
              />
            ))}
        </div>
      </AccordionContent>
    </AccordionItem>
  )
}

interface MissingEpisodeRowProps {
  series: MissingSeries
  episode: MissingEpisode
  onMonitored: (seriesId: number, episodeId: number, label: string, monitored: boolean) => void
  isDisabled: boolean
}

function MissingEpisodeRow({ series: s, episode, onMonitored, isDisabled }: MissingEpisodeRowProps) {
  const epLabel = `S${episode.seasonNumber.toString().padStart(2, '0')}E${episode.episodeNumber.toString().padStart(2, '0')}`

  return (
    <div
      className="flex items-center justify-between py-1.5 px-1 text-sm rounded hover:bg-muted/50"
    >
      <div className="flex items-center gap-2 min-w-0">
        <span className="text-tv-400 font-mono text-xs w-6 shrink-0 text-center">
          {episode.episodeNumber.toString().padStart(2, '0')}
        </span>
        <span
          className={cn(
            'truncate',
            !episode.title && 'text-muted-foreground italic'
          )}
          title={episode.title || 'TBA'}
        >
          {episode.title || 'TBA'}
        </span>
        {episode.airDate && (
          <span className="flex items-center gap-1 text-muted-foreground text-xs shrink-0">
            <Calendar className="size-3" />
            {formatDate(episode.airDate)}
          </span>
        )}
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
          monitored={true}
          onMonitoredChange={(m) => onMonitored(s.id, episode.id, epLabel, m)}
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
