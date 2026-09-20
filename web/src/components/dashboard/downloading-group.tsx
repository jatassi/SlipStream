import { Link } from '@tanstack/react-router'
import { ArrowDownToLine } from 'lucide-react'

import { Group, IconTile, ProgressLine, Row, RowSkeleton } from '@/components/grouped-list'
import { RowThumbnail } from '@/components/media/row-thumbnail'
import { useMovie, useQueue, useSeriesDetail } from '@/hooks'
import { formatEta, formatSpeed } from '@/lib/formatters'
import { useUIStore } from '@/stores'
import type { QueueItem } from '@/types/queue'

const SKELETONS = ['dl-a', 'dl-b', 'dl-c'] as const

function queueHref(item: QueueItem): string | undefined {
  if (item.movieId) {
    return `/movies/${item.movieId}`
  }
  if (item.seriesId) {
    return `/series/${item.seriesId}`
  }
  return undefined
}

function MovieThumbnail({ movieId, title }: { movieId?: number; title: string }) {
  const { data } = useMovie(movieId ?? 0)
  return <RowThumbnail tmdbId={data?.tmdbId} alt={title} type="movie" size="sm" />
}

function SeriesThumbnail({ seriesId, title }: { seriesId?: number; title: string }) {
  const { data } = useSeriesDetail(seriesId ?? 0)
  return (
    <RowThumbnail tmdbId={data?.tmdbId} tvdbId={data?.tvdbId} alt={title} type="series" size="sm" />
  )
}

function DownloadThumbnail({ item }: { item: QueueItem }) {
  if (item.mediaType === 'movie') {
    return <MovieThumbnail movieId={item.movieId} title={item.title} />
  }
  return <SeriesThumbnail seriesId={item.seriesId} title={item.title} />
}

function DownloadStats({ item }: { item: QueueItem }) {
  return (
    <span className="flex items-center gap-2">
      <ProgressLine
        value={item.progress}
        kind={item.mediaType === 'movie' ? 'movie' : 'series'}
        className="w-24"
      />
      <span className="nums">
        {item.progress.toFixed(0)}% · {formatSpeed(item.downloadSpeed)} · {formatEta(item.eta)}
      </span>
    </span>
  )
}

function DownloadRow({ item }: { item: QueueItem }) {
  return (
    <Row
      href={queueHref(item)}
      leading={<DownloadThumbnail item={item} />}
      title={item.title}
      subtitle={<DownloadStats item={item} />}
      chevron
    />
  )
}

function EmptyQueue() {
  return (
    <Row
      leading={
        <IconTile className="bg-zinc-600">
          <ArrowDownToLine />
        </IconTile>
      }
      title="Nothing downloading"
      subtitle="The queue is empty"
    />
  )
}

function SeeAll() {
  return (
    <Link to="/downloads" className="press-dim text-footnote font-medium text-tv-400">
      See all
    </Link>
  )
}

export function DownloadingGroup() {
  const globalLoading = useUIStore((s) => s.globalLoading)
  const { data, isLoading } = useQueue()

  if (isLoading || globalLoading) {
    return (
      <Group header="Downloading" action={<SeeAll />} inset={false} className="mb-0">
        {SKELETONS.map((id) => (
          <RowSkeleton key={id} leading="poster" progress chevron />
        ))}
      </Group>
    )
  }

  const active = (data?.items.filter((item) => item.status === 'downloading') ?? []).slice(0, 3)
  return (
    <Group header="Downloading" action={<SeeAll />} inset={false} className="mb-0">
      {active.length === 0 ? (
        <EmptyQueue />
      ) : (
        active.map((item) => <DownloadRow key={`${item.clientId}-${item.id}`} item={item} />)
      )}
    </Group>
  )
}
