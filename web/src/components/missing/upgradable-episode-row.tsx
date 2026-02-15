import { ArrowRight, Calendar } from 'lucide-react'

import { MediaSearchMonitorControls } from '@/components/search'
import { Badge } from '@/components/ui/badge'
import { formatDate } from '@/lib/formatters'
import { cn } from '@/lib/utils'
import type { UpgradableEpisode, UpgradableSeries } from '@/types/missing'
import type { QualityProfile } from '@/types/quality-profile'
import { PREDEFINED_QUALITIES } from '@/types/quality-profile'

import type { EpisodeMonitoredParams } from './use-upgradable-series-list'

const qualityById = new Map(PREDEFINED_QUALITIES.map((q) => [q.id, q.name]))

type UpgradableEpisodeRowProps = {
  series: UpgradableSeries
  episode: UpgradableEpisode
  profile?: QualityProfile
  onMonitored: (params: EpisodeMonitoredParams) => void
  isDisabled: boolean
}

export function UpgradableEpisodeRow({ series, episode, profile, onMonitored, isDisabled }: UpgradableEpisodeRowProps) {
  const epLabel = `S${episode.seasonNumber.toString().padStart(2, '0')}E${episode.episodeNumber.toString().padStart(2, '0')}`
  const currentName = qualityById.get(episode.currentQualityId) ?? 'Unknown'
  const cutoffName = profile ? (qualityById.get(profile.cutoff) ?? 'Unknown') : 'Unknown'

  return (
    <div className="hover:bg-muted/50 flex items-center justify-between rounded px-1 py-1.5 text-sm">
      <EpisodeInfo episode={episode} currentName={currentName} cutoffName={cutoffName} />
      <div className="shrink-0">
        <MediaSearchMonitorControls
          mediaType="episode"
          episodeId={episode.id}
          seriesId={series.id}
          seriesTitle={series.title}
          seasonNumber={episode.seasonNumber}
          episodeNumber={episode.episodeNumber}
          title={epLabel}
          theme="tv"
          size="xs"
          monitored
          onMonitoredChange={(m) => onMonitored({ seriesId: series.id, episodeId: episode.id, label: epLabel, monitored: m })}
          monitorDisabled={isDisabled}
          qualityProfileId={series.qualityProfileId}
          tvdbId={series.tvdbId}
          tmdbId={series.tmdbId}
          imdbId={series.imdbId}
        />
      </div>
    </div>
  )
}

type EpisodeInfoProps = {
  episode: UpgradableEpisode
  currentName: string
  cutoffName: string
}

function EpisodeInfo({ episode, currentName, cutoffName }: EpisodeInfoProps) {
  return (
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
      <Badge variant="secondary" className="shrink-0 gap-1 px-1.5 py-0 text-[10px]">
        <span className="text-yellow-500">{currentName}</span>
        <ArrowRight className="text-muted-foreground size-2.5" />
        <span className="text-green-500">{cutoffName}</span>
      </Badge>
      {episode.airDate ? (
        <span className="text-muted-foreground flex shrink-0 items-center gap-1 text-xs">
          <Calendar className="size-3" />
          {formatDate(episode.airDate)}
        </span>
      ) : null}
    </div>
  )
}
