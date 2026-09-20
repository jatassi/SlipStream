import { useNavigate } from '@tanstack/react-router'
import { Plus } from 'lucide-react'

import { Group, IconTile, Row, RowSkeleton } from '@/components/grouped-list'
import type { PosterCellItem } from '@/components/media/poster-cell'
import { RowThumbnail } from '@/components/media/row-thumbnail'
import { StatusDot } from '@/components/media/status-dot'

import type { ExternalResult, ExternalSearch } from './use-search-page'

const KIND: Record<'movie' | 'series', { label: string; className: string }> = {
  movie: { label: 'Movie', className: 'text-movie-400' },
  series: { label: 'Series', className: 'text-tv-400' },
}

export const ADD_NEW = 'Add new'

function Caption({
  kind,
  year,
  children,
}: {
  kind: 'movie' | 'series'
  year: number | null
  children?: React.ReactNode
}) {
  return (
    <span className="flex items-center gap-1.5">
      {children}
      <span className="nums">{year ?? 'Unknown'}</span>
      <span aria-hidden="true">·</span>
      <span className={KIND[kind].className}>{KIND[kind].label}</span>
    </span>
  )
}

function resultsHeader(count: number): string {
  return `${count} result${count === 1 ? '' : 's'}`
}

export function LibraryResults({ results }: { results: PosterCellItem[] }) {
  return (
    <Group header={resultsHeader(results.length)}>
      {results.map((item) => (
        <Row
          key={`${item.posterType}-${item.id}`}
          href={item.href}
          chevron
          leading={
            <RowThumbnail
              tmdbId={item.tmdbId}
              tvdbId={item.tvdbId}
              type={item.posterType}
              version={item.version}
              alt=""
              size="sm"
            />
          }
          title={item.title}
          subtitle={
            <Caption kind={item.posterType} year={item.year ?? null}>
              <StatusDot status={item.status} />
            </Caption>
          }
        />
      ))}
    </Group>
  )
}

function ExternalRow({ result }: { result: ExternalResult }) {
  const navigate = useNavigate()
  return (
    <Row
      chevron
      onClick={() => void navigate({ to: result.addTo, search: { tmdbId: result.tmdbId } })}
      leading={<RowThumbnail url={result.posterUrl} type={result.kind} alt="" size="sm" />}
      title={result.title}
      subtitle={<Caption kind={result.kind} year={result.year} />}
      trailing={result.inLibrary ? 'In library' : undefined}
    />
  )
}

function ExternalBody({ external, query }: { external: ExternalSearch; query: string }) {
  if (external.isLoading) {
    return (
      <Group header={ADD_NEW}>
        <RowSkeleton leading="poster" />
        <RowSkeleton leading="poster" />
        <RowSkeleton leading="poster" />
      </Group>
    )
  }
  if (external.results.length === 0) {
    return (
      <Group header={ADD_NEW}>
        <Row title={`Nothing online matches “${query}”`} />
      </Group>
    )
  }
  return (
    <Group header={ADD_NEW}>
      {external.results.map((result) => (
        <ExternalRow key={`${result.kind}-${result.tmdbId}`} result={result} />
      ))}
    </Group>
  )
}

export function AddNew({ external, query }: { external: ExternalSearch; query: string }) {
  if (!external.open) {
    return (
      <Group>
        <Row
          chevron
          onClick={external.reveal}
          leading={
            <IconTile className="bg-primary">
              <Plus />
            </IconTile>
          }
          title={ADD_NEW}
          subtitle={`Search online for “${query}”`}
        />
      </Group>
    )
  }
  return <ExternalBody external={external} query={query} />
}
