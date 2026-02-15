import { Plus } from 'lucide-react'

import { ExpandableMediaGrid, ExternalMediaCard } from '@/components/search'
import type { PortalMovieSearchResult, PortalSeriesSearchResult } from '@/types'

import { convertToMovieSearchResult, convertToSeriesSearchResult } from './search-utils'

const ACTION_ICON = <Plus className="mr-1 size-3 md:mr-2 md:size-4" />

type MovieGridProps = {
  items: PortalMovieSearchResult[]
  currentUserId?: number
  collapsible?: boolean
  isRequested?: (tmdbId: number) => boolean
  onAction?: (movie: PortalMovieSearchResult) => void
  onViewRequest: (id: number) => void
}

export function MovieGrid({
  items,
  currentUserId,
  collapsible,
  isRequested,
  onAction,
  onViewRequest,
}: MovieGridProps) {
  return (
    <ExpandableMediaGrid
      items={items}
      getKey={(movie) => movie.tmdbId}
      label="Movies"
      icon="movie"
      collapsible={collapsible}
      renderItem={(movie) => (
        <ExternalMediaCard
          media={convertToMovieSearchResult(movie)}
          mediaType="movie"
          availability={movie.availability}
          requested={isRequested?.(movie.tmdbId || movie.id)}
          currentUserId={currentUserId}
          onAction={onAction ? () => onAction(movie) : undefined}
          onViewRequest={onViewRequest}
          actionLabel="Request"
          actionIcon={ACTION_ICON}
          disabledLabel="In Library"
        />
      )}
    />
  )
}

type SeriesGridProps = {
  items: PortalSeriesSearchResult[]
  currentUserId?: number
  collapsible?: boolean
  isRequested?: (tmdbId: number) => boolean
  onAction?: (item: PortalSeriesSearchResult) => void
  onViewRequest: (id: number) => void
}

export function SeriesGrid({
  items,
  currentUserId,
  collapsible,
  isRequested,
  onAction,
  onViewRequest,
}: SeriesGridProps) {
  return (
    <ExpandableMediaGrid
      items={items}
      getKey={(s) => s.tmdbId}
      label="Series"
      icon="series"
      collapsible={collapsible}
      renderItem={(item) => (
        <ExternalMediaCard
          media={convertToSeriesSearchResult(item)}
          mediaType="series"
          availability={item.availability}
          requested={isRequested?.(item.tmdbId || item.id)}
          currentUserId={currentUserId}
          onAction={onAction ? () => onAction(item) : undefined}
          onViewRequest={onViewRequest}
          actionLabel="Request"
          actionIcon={ACTION_ICON}
          disabledLabel="In Library"
        />
      )}
    />
  )
}
