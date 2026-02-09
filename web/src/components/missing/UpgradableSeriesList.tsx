import { Link } from '@tanstack/react-router'
import { ArrowRight, Calendar, ChevronRight, SlidersVertical, TrendingUp } from 'lucide-react'
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
import { PREDEFINED_QUALITIES } from '@/types/qualityProfile'
import type { UpgradableSeries, UpgradableSeason, UpgradableEpisode } from '@/types/missing'
import type { QualityProfile } from '@/types/qualityProfile'

const qualityById = new Map(PREDEFINED_QUALITIES.map((q) => [q.id, q.name]))

interface UpgradableSeriesListProps {
  series: UpgradableSeries[]
  qualityProfiles: Map<number, QualityProfile>
}

export function UpgradableSeriesList({ series, qualityProfiles }: UpgradableSeriesListProps) {
  const updateSeriesMutation = useUpdateSeries()
  const updateSeasonMonitoredMutation = useUpdateSeasonMonitored()
  const updateEpisodeMonitoredMutation = useUpdateEpisodeMonitored()

  const handleSeriesMonitored = async (s: UpgradableSeries, monitored: boolean) => {
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
        icon={<TrendingUp className="size-8 text-tv-400" />}
        title="No upgradable episodes"
        description="All monitored episodes meet their quality cutoff"
        className="py-8"
      />
    )
  }

  return (
    <Accordion className="space-y-2">
      {series.map((s) => {
        const profile = qualityProfiles.get(s.qualityProfileId)

        return (
          <AccordionItem
            key={s.id}
            value={`series-${s.id}`}
            className="border rounded-lg px-4 bg-card transition-colors data-open:border-tv-500/30"
          >
            <AccordionTrigger className="group hover:no-underline py-3 **:data-[slot=accordion-trigger-icon]:!hidden">
              <div className="flex flex-wrap sm:flex-nowrap items-center gap-x-4 gap-y-1 flex-1">
                <ChevronRight className="size-4 shrink-0 transition-transform duration-200 text-muted-foreground group-aria-expanded/accordion-trigger:rotate-90 group-hover:text-tv-400 group-hover:icon-glow-tv" />
                <Link
                  to="/series/$id"
                  params={{ id: s.id.toString() }}
                  className="hidden sm:block shrink-0"
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
                <div className="min-w-0 flex-1 sm:flex-initial">
                  <div className="flex items-baseline gap-2">
                    <Link
                      to="/series/$id"
                      params={{ id: s.id.toString() }}
                      className="font-semibold hover:text-tv-400 transition-colors sm:line-clamp-1"
                      onClick={(e) => e.stopPropagation()}
                    >
                      {s.title}
                    </Link>
                    {s.year && (
                      <span className="shrink-0 text-xs text-muted-foreground">({s.year})</span>
                    )}
                  </div>
                  <div className="flex flex-wrap items-center gap-x-2 gap-y-1 mt-0.5">
                    <Badge variant="secondary">
                      {s.upgradableCount} upgradable
                    </Badge>
                    {profile && (
                      <Badge variant="secondary" className="gap-1 text-[10px] px-1.5 py-0">
                        <SlidersVertical className="size-2.5" />
                        {profile.name}
                      </Badge>
                    )}
                  </div>
                </div>
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
                {s.upgradableSeasons.map((season) => (
                  <UpgradableSeasonItem
                    key={`${s.id}-${season.seasonNumber}`}
                    series={s}
                    season={season}
                    profile={profile}
                    onSeasonMonitored={handleSeasonMonitored}
                    onEpisodeMonitored={handleEpisodeMonitored}
                    isSeasonDisabled={updateSeasonMonitoredMutation.isPending}
                    isEpisodeDisabled={updateEpisodeMonitoredMutation.isPending}
                  />
                ))}
              </Accordion>
            </AccordionContent>
          </AccordionItem>
        )
      })}
    </Accordion>
  )
}

interface UpgradableSeasonItemProps {
  series: UpgradableSeries
  season: UpgradableSeason
  profile?: QualityProfile
  onSeasonMonitored: (seriesId: number, seasonNumber: number, monitored: boolean) => void
  onEpisodeMonitored: (seriesId: number, episodeId: number, label: string, monitored: boolean) => void
  isSeasonDisabled: boolean
  isEpisodeDisabled: boolean
}

function UpgradableSeasonItem({
  series: s,
  season,
  profile,
  onSeasonMonitored,
  onEpisodeMonitored,
  isSeasonDisabled,
  isEpisodeDisabled,
}: UpgradableSeasonItemProps) {
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
            {season.upgradableEpisodes.length} upgradable
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
          {[...season.upgradableEpisodes]
            .sort((a, b) => a.episodeNumber - b.episodeNumber)
            .map((episode) => (
              <UpgradableEpisodeRow
                key={episode.id}
                series={s}
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

interface UpgradableEpisodeRowProps {
  series: UpgradableSeries
  episode: UpgradableEpisode
  profile?: QualityProfile
  onMonitored: (seriesId: number, episodeId: number, label: string, monitored: boolean) => void
  isDisabled: boolean
}

function UpgradableEpisodeRow({ series: s, episode, profile, onMonitored, isDisabled }: UpgradableEpisodeRowProps) {
  const epLabel = `S${episode.seasonNumber.toString().padStart(2, '0')}E${episode.episodeNumber.toString().padStart(2, '0')}`
  const currentName = qualityById.get(episode.currentQualityId) ?? 'Unknown'
  const cutoffName = profile ? (qualityById.get(profile.cutoff) ?? 'Unknown') : 'Unknown'

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
        <Badge variant="secondary" className="gap-1 text-[10px] px-1.5 py-0 shrink-0">
          <span className="text-yellow-500">{currentName}</span>
          <ArrowRight className="size-2.5 text-muted-foreground" />
          <span className="text-green-500">{cutoffName}</span>
        </Badge>
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
