import { movieToCell } from '@/components/media/library-cells'
import { LibraryScreen } from '@/components/media/library-screen'
import { MOVIE_COLUMNS } from '@/lib/table-columns'
import { getModuleOrThrow } from '@/modules'
import type { Movie } from '@/types'

import type { FilterStatus, MovieListState } from './use-movie-list'

const mod = getModuleOrThrow('movie')

export function MovieListLayout({ state: s }: { state: MovieListState }) {
  return (
    <LibraryScreen<Movie>
      moduleId={mod.id} theme={mod.themeColor} title={mod.name}
      addLabel={`Add ${mod.singularName}`} mediaLabel={mod.singularName} pluralMediaLabel={mod.pluralName}
      isLoading={s.isLoading} items={s.sortedMovies} groups={s.groups}
      toCell={(movie) => movieToCell(movie, s.profileNameMap)}
      emptyTitle={`No ${mod.pluralName.toLowerCase()} found`}
      filterOptions={mod.filterOptions} statusFilters={s.statusFilters} onToggleFilter={(value) => s.handleToggleFilter(value as FilterStatus)}
      sortOptions={mod.sortOptions} sortField={s.sortField} sortDirection={s.sortDirection}
      onSortFieldChange={s.handleSortFieldChange} onColumnSort={s.handleColumnSort}
      view={s.moviesView} onViewChange={s.setMoviesView}
      posterSize={s.posterSize} onPosterSizeChange={s.setPosterSize}
      allTableColumns={s.allColumns} staticColumns={MOVIE_COLUMNS}
      visibleColumnIds={s.movieTableColumns} onTableColumnsChange={s.setMovieTableColumns}
      renderContext={s.renderContext}
      editMode={s.editMode} selectedIds={s.selectedIds} filteredCount={s.filteredMovies.length}
      onEnterEdit={() => s.setEditMode(true)} onExitEdit={s.handleExitEditMode}
      onToggleSelect={s.handleToggleSelect} onSelectAll={s.handleSelectAll}
      qualityProfiles={s.qualityProfiles} isBulkUpdating={s.bulkUpdateMutation.isPending}
      isBulkDeleting={s.bulkDeleteMutation.isPending} isRefreshing={s.refreshAllMutation.isPending}
      onBulkMonitor={s.handleBulkMonitor} onBulkChangeQualityProfile={s.handleBulkChangeQualityProfile}
      onBulkDelete={s.handleBulkDelete} onRefreshAll={s.handleRefreshAll}
      showDeleteDialog={s.showDeleteDialog} onShowDeleteDialog={s.setShowDeleteDialog}
      deleteFiles={s.deleteFiles} onDeleteFilesChange={s.setDeleteFiles}
    />
  )
}
