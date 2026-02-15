import { Link } from '@tanstack/react-router'
import { Eye, MoreHorizontal, RefreshCw, Search, Trash2 } from 'lucide-react'

import { MediaStatusBadge } from '@/components/media/media-status-badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { formatBytes, formatDate, formatRuntime } from '@/lib/formatters'
import type { Movie } from '@/types/movie'

import type { ColumnDef } from './column-types'

export const MOVIE_COLUMNS: ColumnDef<Movie>[] = [
  {
    id: 'title',
    label: 'Title',
    sortField: 'title',
    defaultVisible: true,
    hideable: false,
    render: (movie) => (
      <Link
        to="/movies/$id"
        params={{ id: String(movie.id) }}
        className="font-medium hover:underline"
      >
        {movie.title}
      </Link>
    ),
  },
  {
    id: 'year',
    label: 'Year',
    defaultVisible: true,
    hideable: true,
    render: (movie) => movie.year ?? '-',
  },
  {
    id: 'studio',
    label: 'Studio',
    defaultVisible: true,
    hideable: true,
    render: (movie) => <span className="text-muted-foreground">{movie.studio ?? '-'}</span>,
  },
  {
    id: 'status',
    label: 'Status',
    defaultVisible: true,
    hideable: true,
    render: (movie) => <MediaStatusBadge status={movie.status} />,
  },
  {
    id: 'qualityProfile',
    label: 'Quality Profile',
    sortField: 'qualityProfile',
    defaultVisible: true,
    hideable: true,
    render: (movie, ctx) => (
      <span className="text-muted-foreground">
        {ctx.qualityProfileNames.get(movie.qualityProfileId) ?? '-'}
      </span>
    ),
  },
  {
    id: 'rootFolder',
    label: 'Root Folder',
    sortField: 'rootFolder',
    defaultVisible: false,
    hideable: true,
    render: (movie, ctx) => (
      <span className="text-muted-foreground">
        {ctx.rootFolderNames.get(movie.rootFolderId ?? 0) ?? '-'}
      </span>
    ),
  },
  {
    id: 'releaseDate',
    label: 'Release Date',
    sortField: 'releaseDate',
    defaultVisible: true,
    hideable: true,
    render: (movie) => {
      const date = movie.releaseDate ?? movie.physicalReleaseDate ?? movie.theatricalReleaseDate
      return <span className="text-muted-foreground">{date ? formatDate(date) : '-'}</span>
    },
  },
  {
    id: 'dateAdded',
    label: 'Added',
    sortField: 'dateAdded',
    defaultVisible: true,
    hideable: true,
    render: (movie) => <span className="text-muted-foreground">{formatDate(movie.addedAt)}</span>,
  },
  {
    id: 'sizeOnDisk',
    label: 'Size',
    sortField: 'sizeOnDisk',
    defaultVisible: true,
    hideable: true,
    cellClassName: 'tabular-nums',
    render: (movie) => (
      <span className="text-muted-foreground">
        {movie.sizeOnDisk ? formatBytes(movie.sizeOnDisk) : '-'}
      </span>
    ),
  },
  {
    id: 'runtime',
    label: 'Runtime',
    defaultVisible: false,
    hideable: true,
    render: (movie) => (
      <span className="text-muted-foreground">
        {movie.runtime ? formatRuntime(movie.runtime) : '-'}
      </span>
    ),
  },
  {
    id: 'path',
    label: 'Path',
    defaultVisible: false,
    hideable: true,
    minWidth: '200px',
    render: (movie) => (
      <span className="text-muted-foreground block max-w-[300px] truncate font-mono text-xs">
        {movie.path ?? '-'}
      </span>
    ),
  },
]

export function createMovieActionsColumn(callbacks: {
  onSearch?: (id: number) => void
  onRefresh?: (id: number) => void
  onDelete?: (id: number) => void
}): ColumnDef<Movie> {
  return {
    id: 'actions',
    label: '',
    defaultVisible: true,
    hideable: false,
    headerClassName: 'w-[50px]',
    render: (movie) => (
      <DropdownMenu>
        <DropdownMenuTrigger className="focus-visible:ring-ring hover:bg-accent hover:text-accent-foreground inline-flex size-8 items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:ring-1 focus-visible:outline-none">
          <MoreHorizontal className="size-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <Link to="/movies/$id" params={{ id: String(movie.id) }}>
            <DropdownMenuItem>
              <Eye className="mr-2 size-4" />
              View details
            </DropdownMenuItem>
          </Link>
          {callbacks.onSearch ? (
            <DropdownMenuItem onClick={() => callbacks.onSearch?.(movie.id)}>
              <Search className="mr-2 size-4" />
              Search
            </DropdownMenuItem>
          ) : null}
          {callbacks.onRefresh ? (
            <DropdownMenuItem onClick={() => callbacks.onRefresh?.(movie.id)}>
              <RefreshCw className="mr-2 size-4" />
              Refresh
            </DropdownMenuItem>
          ) : null}
          {callbacks.onDelete ? (
            <DropdownMenuItem
              onClick={() => callbacks.onDelete?.(movie.id)}
              className="text-destructive"
            >
              <Trash2 className="mr-2 size-4" />
              Delete
            </DropdownMenuItem>
          ) : null}
        </DropdownMenuContent>
      </DropdownMenu>
    ),
  }
}
