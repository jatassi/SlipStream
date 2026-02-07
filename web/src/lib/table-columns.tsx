import type { ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import { MoreHorizontal, Search, Trash2, RefreshCw, Eye } from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { MediaStatusBadge } from '@/components/media/MediaStatusBadge'
import { ProgressBar } from '@/components/media/ProgressBar'
import { NetworkLogo } from '@/components/media/NetworkLogo'
import { formatBytes, formatDate, formatRuntime } from '@/lib/formatters'
import type { Movie } from '@/types/movie'
import type { Series, StatusCounts } from '@/types/series'

export interface ColumnDef<T> {
  id: string
  label: string
  sortField?: string
  defaultVisible: boolean
  hideable: boolean
  minWidth?: string
  render: (item: T, ctx: ColumnRenderContext) => ReactNode
  headerClassName?: string
  cellClassName?: string
}

export interface ColumnRenderContext {
  qualityProfileNames: Map<number, string>
  rootFolderNames: Map<number, string>
}

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
    render: (movie) => <>{movie.year || '-'}</>,
  },
  {
    id: 'studio',
    label: 'Studio',
    defaultVisible: true,
    hideable: true,
    render: (movie) => (
      <span className="text-muted-foreground">{movie.studio || '-'}</span>
    ),
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
        {ctx.qualityProfileNames.get(movie.qualityProfileId) || '-'}
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
        {ctx.rootFolderNames.get(movie.rootFolderId || 0) || '-'}
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
    render: (movie) => (
      <span className="text-muted-foreground">{formatDate(movie.addedAt)}</span>
    ),
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
      <span className="text-muted-foreground text-xs font-mono truncate max-w-[300px] block">
        {movie.path || '-'}
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
        <DropdownMenuTrigger className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring hover:bg-accent hover:text-accent-foreground size-8">
          <MoreHorizontal className="size-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <Link to="/movies/$id" params={{ id: String(movie.id) }}>
            <DropdownMenuItem>
              <Eye className="size-4 mr-2" />
              View details
            </DropdownMenuItem>
          </Link>
          {callbacks.onSearch && (
            <DropdownMenuItem onClick={() => callbacks.onSearch!(movie.id)}>
              <Search className="size-4 mr-2" />
              Search
            </DropdownMenuItem>
          )}
          {callbacks.onRefresh && (
            <DropdownMenuItem onClick={() => callbacks.onRefresh!(movie.id)}>
              <RefreshCw className="size-4 mr-2" />
              Refresh
            </DropdownMenuItem>
          )}
          {callbacks.onDelete && (
            <DropdownMenuItem
              onClick={() => callbacks.onDelete!(movie.id)}
              className="text-destructive"
            >
              <Trash2 className="size-4 mr-2" />
              Delete
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    ),
  }
}

// --- Series columns ---

export function getAggregateStatus(
  counts: StatusCounts,
): 'downloading' | 'failed' | 'missing' | 'upgradable' | 'available' | 'unreleased' {
  if (counts.downloading > 0) return 'downloading'
  if (counts.failed > 0) return 'failed'
  if (counts.missing > 0) return 'missing'
  if (counts.upgradable > 0) return 'upgradable'
  if (counts.available > 0) return 'available'
  return 'unreleased'
}

export const SERIES_COLUMNS: ColumnDef<Series>[] = [
  {
    id: 'title',
    label: 'Title',
    sortField: 'title',
    defaultVisible: true,
    hideable: false,
    render: (series) => (
      <Link
        to="/series/$id"
        params={{ id: String(series.id) }}
        className="font-medium hover:underline"
      >
        {series.title}
      </Link>
    ),
  },
  {
    id: 'network',
    label: 'Network',
    defaultVisible: true,
    hideable: true,
    render: (series) =>
      series.networkLogoUrl ? (
        <NetworkLogo
          logoUrl={series.networkLogoUrl}
          network={series.network}
          className="inline-flex"
        />
      ) : (
        <span className="text-muted-foreground">{series.network || '-'}</span>
      ),
  },
  {
    id: 'seasons',
    label: 'Seasons',
    defaultVisible: true,
    hideable: true,
    render: (series) => <>{series.seasons?.length ?? '-'}</>,
  },
  {
    id: 'episodes',
    label: 'Episodes',
    defaultVisible: true,
    hideable: true,
    minWidth: '120px',
    render: (series) => {
      const counts = series.statusCounts
      const available = counts.available + counts.upgradable
      return (
        <div className="flex items-center gap-2">
          <span className="text-xs tabular-nums whitespace-nowrap">
            {available}/{counts.total}
          </span>
          <ProgressBar
            value={available}
            max={counts.total || 1}
            variant="tv"
            size="sm"
            className="flex-1 min-w-[60px]"
          />
        </div>
      )
    },
  },
  {
    id: 'productionStatus',
    label: 'Production',
    defaultVisible: true,
    hideable: true,
    render: (series) => (
      <span className="text-muted-foreground capitalize">{series.productionStatus}</span>
    ),
  },
  {
    id: 'qualityProfile',
    label: 'Quality Profile',
    sortField: 'qualityProfile',
    defaultVisible: true,
    hideable: true,
    render: (series, ctx) => (
      <span className="text-muted-foreground">
        {ctx.qualityProfileNames.get(series.qualityProfileId) || '-'}
      </span>
    ),
  },
  {
    id: 'rootFolder',
    label: 'Root Folder',
    sortField: 'rootFolder',
    defaultVisible: false,
    hideable: true,
    render: (series, ctx) => (
      <span className="text-muted-foreground">
        {ctx.rootFolderNames.get(series.rootFolderId || 0) || '-'}
      </span>
    ),
  },
  {
    id: 'nextAiring',
    label: 'Next Airing',
    sortField: 'nextAirDate',
    defaultVisible: true,
    hideable: true,
    render: (series) => (
      <span className="text-muted-foreground">
        {series.nextAiring ? formatDate(series.nextAiring) : '-'}
      </span>
    ),
  },
  {
    id: 'dateAdded',
    label: 'Added',
    sortField: 'dateAdded',
    defaultVisible: true,
    hideable: true,
    render: (series) => (
      <span className="text-muted-foreground">{formatDate(series.addedAt)}</span>
    ),
  },
  {
    id: 'sizeOnDisk',
    label: 'Size',
    sortField: 'sizeOnDisk',
    defaultVisible: true,
    hideable: true,
    cellClassName: 'tabular-nums',
    render: (series) => (
      <span className="text-muted-foreground">
        {series.sizeOnDisk ? formatBytes(series.sizeOnDisk) : '-'}
      </span>
    ),
  },
  {
    id: 'path',
    label: 'Path',
    defaultVisible: false,
    hideable: true,
    minWidth: '200px',
    render: (series) => (
      <span className="text-muted-foreground text-xs font-mono truncate max-w-[300px] block">
        {series.path || '-'}
      </span>
    ),
  },
]

export function createSeriesActionsColumn(callbacks: {
  onDelete?: (id: number) => void
}): ColumnDef<Series> {
  return {
    id: 'actions',
    label: '',
    defaultVisible: true,
    hideable: false,
    headerClassName: 'w-[50px]',
    render: (series) => (
      <DropdownMenu>
        <DropdownMenuTrigger className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring hover:bg-accent hover:text-accent-foreground size-8">
          <MoreHorizontal className="size-4" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <Link to="/series/$id" params={{ id: String(series.id) }}>
            <DropdownMenuItem>
              <Eye className="size-4 mr-2" />
              View details
            </DropdownMenuItem>
          </Link>
          {callbacks.onDelete && (
            <DropdownMenuItem
              onClick={() => callbacks.onDelete!(series.id)}
              className="text-destructive"
            >
              <Trash2 className="size-4 mr-2" />
              Delete
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    ),
  }
}

export function getDefaultVisibleColumns<T>(columns: ColumnDef<T>[]): string[] {
  return columns.filter((c) => c.defaultVisible).map((c) => c.id)
}

export const DEFAULT_SORT_DIRECTIONS: Record<string, 'asc' | 'desc'> = {
  title: 'asc',
  monitored: 'desc',
  qualityProfile: 'asc',
  releaseDate: 'desc',
  dateAdded: 'desc',
  nextAirDate: 'desc',
  rootFolder: 'asc',
  sizeOnDisk: 'desc',
}
