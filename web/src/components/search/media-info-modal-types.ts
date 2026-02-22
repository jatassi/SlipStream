import type {
  EnrichedSeason,
  ExtendedMovieResult,
  ExtendedSeriesResult,
  MovieSearchResult,
  PortalDownload,
  SeriesSearchResult,
} from '@/types'

export type MediaInfoModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  media: MovieSearchResult | SeriesSearchResult
  mediaType: 'movie' | 'series'
  inLibrary?: boolean
  onAction?: () => void
  actionLabel?: string
  actionIcon?: React.ReactNode
  disabledLabel?: string
}

export type MediaInfoState = {
  extendedData: ExtendedMovieResult | ExtendedSeriesResult | undefined
  isLoading: boolean
  isInLibrary: boolean
  isPending: boolean
  isApproved: boolean
  isAvailable: boolean
  hasActiveDownload: boolean
  activeDownload: PortalDownload | undefined
  director: string | undefined
  creators: { name: string }[] | undefined
  studio: string | undefined
  trailerUrl: string | undefined
  seasons: ExtendedSeriesResult['seasons'] | undefined
  enrichedSeasons: EnrichedSeason[] | undefined
  handleAdd: () => void
}
