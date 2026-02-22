import type { SearchModal } from './search-modal'

export type MediaTheme = 'movie' | 'tv'
type ControlSize = 'lg' | 'sm' | 'xs' | 'responsive'
export type ResolvedSize = 'lg' | 'sm' | 'xs'

type BaseProps = {
  title: string
  theme: MediaTheme
  size: ControlSize
  monitored: boolean
  onMonitoredChange: (monitored: boolean) => void
  monitorDisabled?: boolean
  qualityProfileId: number
  className?: string
}

type MovieProps = {
  mediaType: 'movie'
  movieId: number
  tmdbId?: number
  imdbId?: string
  year?: number
} & BaseProps

type SeriesProps = {
  mediaType: 'series'
  seriesId: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
} & BaseProps

type SeasonProps = {
  mediaType: 'season'
  seriesId: number
  seriesTitle: string
  seasonNumber: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
} & BaseProps

type EpisodeProps = {
  mediaType: 'episode'
  episodeId: number
  seriesId: number
  seriesTitle: string
  seasonNumber: number
  episodeNumber: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
} & BaseProps

type MovieSlotProps = {
  mediaType: 'movie-slot'
  movieId: number
  slotId: number
  tmdbId?: number
  imdbId?: string
  year?: number
} & BaseProps

type EpisodeSlotProps = {
  mediaType: 'episode-slot'
  episodeId: number
  slotId: number
  seriesId: number
  seriesTitle: string
  seasonNumber: number
  episodeNumber: number
  tvdbId?: number
  tmdbId?: number
  imdbId?: string
} & BaseProps

export type MediaSearchMonitorControlsProps =
  | MovieProps
  | SeriesProps
  | SeasonProps
  | EpisodeProps
  | MovieSlotProps
  | EpisodeSlotProps

export type ControlState =
  | { type: 'default' }
  | { type: 'searching'; mode: 'manual' | 'auto' }
  | { type: 'progress' }
  | { type: 'completed' }
  | { type: 'error'; message: string }

export type SearchModalExternalProps = Omit<
  React.ComponentProps<typeof SearchModal>,
  'open' | 'onOpenChange' | 'onGrabSuccess'
>
