import { Film } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import { LoadingState } from '@/components/data/loading-state'
import { GroupedMovieGrid } from '@/components/movies/grouped-movie-grid'
import { MovieGrid } from '@/components/movies/movie-grid'
import { MovieTable } from '@/components/movies/movie-table'
import type { MediaGroup } from '@/lib/grouping'
import type { ColumnDef, ColumnRenderContext } from '@/lib/table-columns'
import type { Movie } from '@/types'

type Props = {
  isLoading: boolean
  moviesView: 'grid' | 'table'
  sortedMovies: Movie[]
  groups: MediaGroup<Movie & { releaseDate?: string }>[] | null
  posterSize: number
  editMode: boolean
  selectedIds: Set<number>
  allFiltersSelected: boolean
  allColumns: ColumnDef<Movie>[]
  movieTableColumns: string[]
  renderContext: ColumnRenderContext
  sortField: string
  sortDirection: 'asc' | 'desc'
  onSort: (field: string) => void
  onToggleSelect: (id: number) => void
}

export function MovieListContent(props: Props) {
  if (props.isLoading) {
    return <LoadingState variant={props.moviesView === 'grid' ? 'card' : 'list'} posterSize={props.posterSize} theme="movie" />
  }
  if (props.sortedMovies.length === 0) {
    return <MovieEmptyState allFiltersSelected={props.allFiltersSelected} />
  }
  if (props.moviesView === 'table') {
    return <TableView {...props} />
  }
  if (props.groups) {
    return <GroupedMovieGrid groups={props.groups} posterSize={props.posterSize} editMode={props.editMode} selectedIds={props.selectedIds} onToggleSelect={props.onToggleSelect} />
  }
  return <MovieGrid movies={props.sortedMovies} posterSize={props.posterSize} editMode={props.editMode} selectedIds={props.selectedIds} onToggleSelect={props.onToggleSelect} />
}

function MovieEmptyState({ allFiltersSelected }: { allFiltersSelected: boolean }) {
  return (
    <EmptyState
      icon={<Film className="text-movie-500 size-8" />}
      title="No movies found"
      description={allFiltersSelected ? 'Add your first movie to get started' : 'Try adjusting your filters'}
    />
  )
}

function TableView(props: Props) {
  return (
    <MovieTable
      movies={props.sortedMovies}
      columns={props.allColumns}
      visibleColumnIds={props.movieTableColumns}
      renderContext={props.renderContext}
      sortField={props.sortField}
      sortDirection={props.sortDirection}
      onSort={props.onSort}
      editMode={props.editMode}
      selectedIds={props.selectedIds}
      onToggleSelect={props.onToggleSelect}
    />
  )
}
