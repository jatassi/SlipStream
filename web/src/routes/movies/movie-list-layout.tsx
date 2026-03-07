import { MediaListLayout } from '@/components/media/media-list-layout'
import { MovieCard } from '@/components/movies/movie-card'
import { MOVIE_COLUMNS } from '@/lib/table-columns'
import { getModuleOrThrow } from '@/modules'

import type { MovieListState } from './use-movie-list'

const mod = getModuleOrThrow('movie')
const ModIcon = mod.icon

export function MovieListLayout({ state: s }: { state: MovieListState }) {
  return (
    <MediaListLayout
      theme={mod.themeColor} title={mod.name} addLabel={`Add ${mod.singularName}`}
      mediaLabel={mod.singularName} pluralMediaLabel={mod.pluralName} libraryCount={s.movies?.length}
      filterOptions={mod.filterOptions as never} sortOptions={mod.sortOptions as never}
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
      emptyIcon={<ModIcon className="text-movie-500 size-8" />} emptyTitle={`No ${mod.pluralName.toLowerCase()} found`}
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
