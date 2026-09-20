import { Fragment, useMemo, useState } from 'react'

import { ChevronRight } from 'lucide-react'

import { Group, Row } from '@/components/grouped-list'
import { aggregateMediaStatus } from '@/components/media/media-status'
import { StatusDot } from '@/components/media/status-dot'
import { StatusPill } from '@/components/media/status-pill'
import { MediaSearchMonitorControls } from '@/components/search'
import { formatDate } from '@/lib/formatters'
import { cn } from '@/lib/utils'
import type { Episode, Season, Slot } from '@/types'

import { EpisodeSlotStatusContent } from './episode-slot-status-content'
import type { SeriesInfo } from './series-context'
import { SeriesContext, useSeriesInfo } from './series-context'

type SeasonListProps = SeriesInfo & {
  seasons: Season[]
  episodes?: Episode[]
  onSeasonMonitoredChange?: (seasonNumber: number, monitored: boolean) => void
  onEpisodeMonitoredChange?: (episode: Episode, monitored: boolean) => void
  isMultiVersionEnabled?: boolean
  enabledSlots?: Slot[]
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

function seasonLabelFor(seasonNumber: number): string {
  return seasonNumber === 0 ? 'Specials' : `Season ${seasonNumber}`
}

function buildSlotQualityMap(slots: Slot[]): Record<number, number> {
  const map: Record<number, number> = {}
  for (const slot of slots) {
    if (slot.qualityProfileId !== null) {
      map[slot.id] = slot.qualityProfileId
    }
  }
  return map
}

export function SeasonList(props: SeasonListProps) {
  const { seasons, episodes = [], enabledSlots = [], isMultiVersionEnabled = false } = props
  const [expanded, setExpanded] = useState<number | null>(null)
  const episodesBySeason = useMemo(() => groupEpisodesBySeason(episodes), [episodes])
  const sortedSeasons = useMemo(() => sortSeasons(seasons), [seasons])
  const slotQualityProfiles = useMemo(() => buildSlotQualityMap(enabledSlots), [enabledSlots])

  const seriesInfo: SeriesInfo = {
    seriesId: props.seriesId,
    seriesTitle: props.seriesTitle,
    qualityProfileId: props.qualityProfileId,
    tvdbId: props.tvdbId,
    tmdbId: props.tmdbId,
    imdbId: props.imdbId,
  }

  return (
    <SeriesContext.Provider value={seriesInfo}>
      <Group header="Seasons">
        {sortedSeasons.map((season) => (
          <Fragment key={season.id}>
            <SeasonRow
              season={season}
              expanded={expanded === season.seasonNumber}
              onToggle={() =>
                setExpanded((prev) => (prev === season.seasonNumber ? null : season.seasonNumber))
              }
              onSeasonMonitoredChange={props.onSeasonMonitoredChange}
            />
            {expanded === season.seasonNumber && (
              <SeasonEpisodes
                episodes={episodesBySeason[season.seasonNumber] ?? []}
                onEpisodeMonitoredChange={props.onEpisodeMonitoredChange}
                isMultiVersionEnabled={isMultiVersionEnabled}
                slotQualityProfiles={slotQualityProfiles}
              />
            )}
          </Fragment>
        ))}
      </Group>
    </SeriesContext.Provider>
  )
}

function SeasonRow({
  season,
  expanded,
  onToggle,
  onSeasonMonitoredChange,
}: {
  season: Season
  expanded: boolean
  onToggle: () => void
  onSeasonMonitoredChange?: (seasonNumber: number, monitored: boolean) => void
}) {
  const label = seasonLabelFor(season.seasonNumber)
  const fileCount = season.statusCounts.available + season.statusCounts.upgradable
  const totalCount = season.statusCounts.total - season.statusCounts.unreleased

  return (
    <div className="flex items-center gap-2 pr-4">
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={expanded}
        className="press-row min-h-tap focus-visible:ring-ring flex min-w-0 flex-1 items-center gap-3 px-4 py-2.5 text-left outline-none focus-visible:ring-[3px]"
      >
        <ChevronRight
          className={cn(
            'text-muted-foreground/60 size-4 shrink-0 transition-transform duration-[var(--dur-fast)]',
            expanded && 'rotate-90',
          )}
        />
        <span className="min-w-0 flex-1">
          <span className="text-body block truncate font-medium">{label}</span>
          <span className="text-footnote text-muted-foreground nums mt-0.5 block">
            {fileCount}/{totalCount} episodes
          </span>
        </span>
        <StatusPill status={aggregateMediaStatus(season.statusCounts)} />
      </button>
      <SeasonActions season={season} label={label} onSeasonMonitoredChange={onSeasonMonitoredChange} />
    </div>
  )
}

function SeasonActions({
  season,
  label,
  onSeasonMonitoredChange,
}: {
  season: Season
  label: string
  onSeasonMonitoredChange?: (seasonNumber: number, monitored: boolean) => void
}) {
  const info = useSeriesInfo()
  return (
    <MediaSearchMonitorControls
      mediaType="season"
      seriesId={info.seriesId}
      seriesTitle={info.seriesTitle}
      seasonNumber={season.seasonNumber}
      title={label}
      theme="tv"
      monitored={season.monitored}
      onMonitoredChange={(m) => onSeasonMonitoredChange?.(season.seasonNumber, m)}
      monitorDisabled={onSeasonMonitoredChange === undefined}
      qualityProfileId={info.qualityProfileId}
      tvdbId={info.tvdbId}
      tmdbId={info.tmdbId}
      imdbId={info.imdbId}
    />
  )
}

function SeasonEpisodes({
  episodes,
  onEpisodeMonitoredChange,
  isMultiVersionEnabled,
  slotQualityProfiles,
}: {
  episodes: Episode[]
  onEpisodeMonitoredChange?: (episode: Episode, monitored: boolean) => void
  isMultiVersionEnabled: boolean
  slotQualityProfiles: Record<number, number>
}) {
  const sorted = useMemo(
    () => episodes.toSorted((a, b) => a.episodeNumber - b.episodeNumber),
    [episodes],
  )

  if (sorted.length === 0) {
    return <Row title="No episodes found" />
  }

  return (
    <>
      {sorted.map((episode) => (
        <EpisodeRow
          key={episode.id}
          episode={episode}
          onMonitoredChange={onEpisodeMonitoredChange}
          isMultiVersionEnabled={isMultiVersionEnabled}
          slotQualityProfiles={slotQualityProfiles}
        />
      ))}
    </>
  )
}

function episodeCode(episode: Episode): string {
  return `S${episode.seasonNumber.toString().padStart(2, '0')}E${episode.episodeNumber.toString().padStart(2, '0')}`
}

function episodeSubtitle(episode: Episode): string {
  const parts = [episode.airDate ? formatDate(episode.airDate) : 'No air date']
  if (episode.episodeFile?.quality) {
    parts.push(episode.episodeFile.quality)
  }
  return parts.join(' · ')
}

type EpisodeRowProps = {
  episode: Episode
  onMonitoredChange?: (episode: Episode, monitored: boolean) => void
  isMultiVersionEnabled: boolean
  slotQualityProfiles: Record<number, number>
}

function EpisodeRow({
  episode,
  onMonitoredChange,
  isMultiVersionEnabled,
  slotQualityProfiles,
}: EpisodeRowProps) {
  const [slotsOpen, setSlotsOpen] = useState(false)
  const slotsVisible = isMultiVersionEnabled && slotsOpen

  return (
    <>
      <Row
        className="bg-foreground/[0.03]"
        leading={<StatusDot status={episode.status} className="ml-1" />}
        title={`${episode.episodeNumber}. ${episode.title}`}
        subtitle={episodeSubtitle(episode)}
        trailing={
          <div className="flex items-center gap-1.5">
            <SlotsToggle
              episode={episode}
              enabled={isMultiVersionEnabled}
              open={slotsOpen}
              onToggle={() => setSlotsOpen((prev) => !prev)}
            />
            <EpisodeActions episode={episode} onMonitoredChange={onMonitoredChange} />
          </div>
        }
      />
      <EpisodeSlots
        episode={episode}
        open={slotsVisible}
        slotQualityProfiles={slotQualityProfiles}
      />
    </>
  )
}

function SlotsToggle({
  episode,
  enabled,
  open,
  onToggle,
}: {
  episode: Episode
  enabled: boolean
  open: boolean
  onToggle: () => void
}) {
  if (!enabled) {
    return null
  }
  return (
    <button
      type="button"
      aria-label={`Version slots for ${episodeCode(episode)}`}
      aria-expanded={open}
      onClick={onToggle}
      className="press focus-visible:ring-ring flex size-8 items-center justify-center rounded-md outline-none focus-visible:ring-[3px]"
    >
      <ChevronRight className={cn('size-4 transition-transform', open && 'rotate-90')} />
    </button>
  )
}

function EpisodeSlots({
  episode,
  open,
  slotQualityProfiles,
}: {
  episode: Episode
  open: boolean
  slotQualityProfiles: Record<number, number>
}) {
  if (!open) {
    return null
  }
  return (
    <div className="px-4 py-2">
      <EpisodeSlotStatusContent episode={episode} slotQualityProfiles={slotQualityProfiles} />
    </div>
  )
}

function EpisodeActions({
  episode,
  onMonitoredChange,
}: {
  episode: Episode
  onMonitoredChange?: (episode: Episode, monitored: boolean) => void
}) {
  const info = useSeriesInfo()
  return (
    <MediaSearchMonitorControls
      mediaType="episode"
      episodeId={episode.id}
      seriesId={info.seriesId}
      seriesTitle={info.seriesTitle}
      seasonNumber={episode.seasonNumber}
      episodeNumber={episode.episodeNumber}
      title={episodeCode(episode)}
      theme="tv"
      monitored={episode.monitored}
      onMonitoredChange={(m) => onMonitoredChange?.(episode, m)}
      monitorDisabled={onMonitoredChange === undefined}
      qualityProfileId={info.qualityProfileId}
      tvdbId={info.tvdbId}
      tmdbId={info.tmdbId}
      imdbId={info.imdbId}
    />
  )
}
