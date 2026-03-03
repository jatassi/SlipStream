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

import { MediaListLayout } from '@/components/media/media-list-layout'
import { MovieCard } from '@/components/movies/movie-card'
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
    <MediaListLayout
      theme="movie" title="Movies" addLabel="Add Movie"
      mediaLabel="Movie" pluralMediaLabel="Movies" libraryCount={s.movies?.length}
      filterOptions={FILTER_OPTIONS} sortOptions={SORT_OPTIONS}
      allTableColumns={s.allColumns} staticColumns={MOVIE_COLUMNS}
      visibleColumnIds={s.movieTableColumns} onTableColumnsChange={s.setMovieTableColumns}
      isLoading={s.isLoading} editMode={s.editMode} selectedIds={s.selectedIds}
      filteredCount={s.filteredMovies.length} allFiltersSelected={s.allFiltersSelected}
      statusFilters={s.statusFilters} sortField={s.sortField}
      sortDirection={s.sortDirection} view={s.moviesView} posterSize={s.posterSize}
      items={s.sortedMovies} groups={s.groups} renderContext={s.renderContext}
      qualityProfiles={s.qualityProfiles} isBulkUpdating={s.bulkUpdateMutation.isPending}
      showDeleteDialog={s.showDeleteDialog} deleteFiles={s.deleteFiles}
      isBulkDeleting={s.bulkDeleteMutation.isPending} isRefreshing={s.refreshAllMutation.isPending}
      emptyIcon={<Film className="text-movie-500 size-8" />} emptyTitle="No movies found"
      renderCard={(movie, opts) => <MovieCard key={movie.id} movie={movie} {...opts} />}
      onToggleFilter={s.handleToggleFilter} onResetFilters={s.handleResetFilters}
      onSortFieldChange={s.handleSortFieldChange} onViewChange={s.handleViewChange}
      onPosterSizeChange={s.handlePosterSizeChange} onColumnSort={s.handleColumnSort}
      onToggleSelect={s.handleToggleSelect} onSelectAll={s.handleSelectAll}
      onBulkMonitor={s.handleBulkMonitor} onBulkChangeQualityProfile={s.handleBulkChangeQualityProfile}
      onDelete={() => s.setShowDeleteDialog(true)} onRefreshAll={s.handleRefreshAll}
      onEnterEdit={() => s.setEditMode(true)} onExitEdit={s.handleExitEditMode}
      onShowDeleteDialog={s.setShowDeleteDialog} onDeleteFilesChange={s.setDeleteFiles}
      onBulkDelete={s.handleBulkDelete}
    />
  )
}
