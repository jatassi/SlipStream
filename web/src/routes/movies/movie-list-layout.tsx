import { PageHeader } from '@/components/layout/page-header'
import { Skeleton } from '@/components/ui/skeleton'

import { MovieDeleteDialog } from './movie-delete-dialog'
import { MovieListContent } from './movie-list-content'
import { MovieListFilters } from './movie-list-filters'
import { MovieListToolbar } from './movie-list-toolbar'
import { MoviePageActions } from './movie-page-actions'
import type { MovieListState } from './use-movie-list'

export function MovieListLayout({ state: s }: { state: MovieListState }) {
  return (
    <div>
      <MovieListHeader state={s} />
      {s.editMode ? (
        <MovieListToolbar
          selectedCount={s.selectedIds.size} totalCount={s.filteredMovies.length}
          qualityProfiles={s.qualityProfiles} isBulkUpdating={s.bulkUpdateMutation.isPending}
          onSelectAll={s.handleSelectAll} onMonitor={s.handleBulkMonitor}
          onChangeQualityProfile={s.handleBulkChangeQualityProfile} onDelete={() => s.setShowDeleteDialog(true)}
        />
      ) : null}
      <MovieListFilters
        statusFilters={s.statusFilters} sortField={s.sortField} moviesView={s.moviesView}
        posterSize={s.posterSize} movieTableColumns={s.movieTableColumns} isLoading={s.isLoading}
        onToggleFilter={s.handleToggleFilter} onResetFilters={s.handleResetFilters}
        onSortFieldChange={s.handleSortFieldChange} onViewChange={s.handleViewChange}
        onPosterSizeChange={s.handlePosterSizeChange} onTableColumnsChange={s.setMovieTableColumns}
      />
      <MovieListContent
        isLoading={s.isLoading} moviesView={s.moviesView} sortedMovies={s.sortedMovies}
        groups={s.groups} posterSize={s.posterSize} editMode={s.editMode}
        selectedIds={s.selectedIds} allFiltersSelected={s.allFiltersSelected}
        allColumns={s.allColumns} movieTableColumns={s.movieTableColumns}
        renderContext={s.renderContext} sortField={s.sortField}
        sortDirection={s.sortDirection} onSort={s.handleColumnSort} onToggleSelect={s.handleToggleSelect}
      />
      <MovieDeleteDialog
        open={s.showDeleteDialog} onOpenChange={s.setShowDeleteDialog}
        selectedCount={s.selectedIds.size} deleteFiles={s.deleteFiles}
        onDeleteFilesChange={s.setDeleteFiles} onConfirm={s.handleBulkDelete} isPending={s.bulkDeleteMutation.isPending}
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
        <MoviePageActions
          isLoading={s.isLoading} editMode={s.editMode} isRefreshing={s.refreshAllMutation.isPending}
          onRefreshAll={s.handleRefreshAll} onEnterEdit={() => s.setEditMode(true)} onExitEdit={s.handleExitEditMode}
        />
      }
    />
  )
}
