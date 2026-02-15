import {
  ArrowDownCircle,
  ArrowUpCircle,
  Binoculars,
  CheckCircle,
  Clock,
  Eye,
  Film,
  XCircle,
} from 'lucide-react'

import { PageHeader } from '@/components/layout/page-header'
import { MediaDeleteDialog } from '@/components/media/media-delete-dialog'
import { MediaListContent } from '@/components/media/media-list-content'
import { MediaListFilters } from '@/components/media/media-list-filters'
import { MediaListToolbar } from '@/components/media/media-list-toolbar'
import { MediaPageActions } from '@/components/media/media-page-actions'
import { MovieCard } from '@/components/movies/movie-card'
import { Skeleton } from '@/components/ui/skeleton'
import { MOVIE_COLUMNS } from '@/lib/table-columns'

import type { FilterStatus, MovieListState, SortField } from './use-movie-list'

const FILTER_OPTIONS: { value: FilterStatus; label: string; icon: typeof Eye }[] = [
  { value: 'monitored', label: 'Monitored', icon: Eye },
  { value: 'unreleased', label: 'Unreleased', icon: Clock },
  { value: 'missing', label: 'Missing', icon: Binoculars },
  { value: 'downloading', label: 'Downloading', icon: ArrowDownCircle },
  { value: 'failed', label: 'Failed', icon: XCircle },
  { value: 'upgradable', label: 'Upgradable', icon: ArrowUpCircle },
  { value: 'available', label: 'Available', icon: CheckCircle },
]

const SORT_OPTIONS: { value: SortField; label: string }[] = [
  { value: 'title', label: 'Title' },
  { value: 'monitored', label: 'Monitored' },
  { value: 'qualityProfile', label: 'Quality Profile' },
  { value: 'releaseDate', label: 'Release Date' },
  { value: 'dateAdded', label: 'Date Added' },
  { value: 'rootFolder', label: 'Root Folder' },
  { value: 'sizeOnDisk', label: 'Size on Disk' },
]

export function MovieListLayout({ state: s }: { state: MovieListState }) {
  return (
    <div>
      <MovieListHeader state={s} />
      {s.editMode ? (
        <MediaListToolbar
          selectedCount={s.selectedIds.size} totalCount={s.filteredMovies.length}
          qualityProfiles={s.qualityProfiles} isBulkUpdating={s.bulkUpdateMutation.isPending}
          onSelectAll={s.handleSelectAll} onMonitor={s.handleBulkMonitor}
          onChangeQualityProfile={s.handleBulkChangeQualityProfile} onDelete={() => s.setShowDeleteDialog(true)}
          theme="movie"
        />
      ) : null}
      <MediaListFilters
        filterOptions={FILTER_OPTIONS} sortOptions={SORT_OPTIONS}
        statusFilters={s.statusFilters} sortField={s.sortField} view={s.moviesView}
        posterSize={s.posterSize} visibleColumnIds={s.movieTableColumns} columns={MOVIE_COLUMNS}
        isLoading={s.isLoading}
        onToggleFilter={s.handleToggleFilter} onResetFilters={s.handleResetFilters}
        onSortFieldChange={s.handleSortFieldChange} onViewChange={s.handleViewChange}
        onPosterSizeChange={s.handlePosterSizeChange} onTableColumnsChange={s.setMovieTableColumns}
        theme="movie"
      />
      <MediaListContent
        isLoading={s.isLoading} view={s.moviesView} items={s.sortedMovies}
        groups={s.groups} posterSize={s.posterSize} editMode={s.editMode}
        selectedIds={s.selectedIds} allFiltersSelected={s.allFiltersSelected}
        allColumns={s.allColumns} visibleColumnIds={s.movieTableColumns}
        renderContext={s.renderContext} sortField={s.sortField}
        sortDirection={s.sortDirection} onSort={s.handleColumnSort} onToggleSelect={s.handleToggleSelect}
        theme="movie"
        renderCard={(movie, opts) => <MovieCard key={movie.id} movie={movie} {...opts} />}
        emptyIcon={<Film className="text-movie-500 size-8" />}
        emptyTitle="No movies found"
      />
      <MediaDeleteDialog
        open={s.showDeleteDialog} onOpenChange={s.setShowDeleteDialog}
        selectedCount={s.selectedIds.size} deleteFiles={s.deleteFiles}
        onDeleteFilesChange={s.setDeleteFiles} onConfirm={s.handleBulkDelete} isPending={s.bulkDeleteMutation.isPending}
        mediaLabel="Movie" pluralMediaLabel="Movies"
      />
    </div>
  )
}

function MovieListHeader({ state: s }: { state: MovieListState }) {
  return (
    <PageHeader
      title="Movies"
      description={s.isLoading ? <Skeleton className="h-4 w-36" /> : `${s.movies?.length ?? 0} movies in library`}
      actions={
        <MediaPageActions
          isLoading={s.isLoading} editMode={s.editMode} isRefreshing={s.refreshAllMutation.isPending}
          onRefreshAll={s.handleRefreshAll} onEnterEdit={() => s.setEditMode(true)} onExitEdit={s.handleExitEditMode}
          theme="movie" addLabel="Add Movie"
        />
      }
    />
  )
}
