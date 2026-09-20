import { seriesToCell } from '@/components/media/library-cells'
import { LibraryScreen } from '@/components/media/library-screen'
import { SERIES_COLUMNS } from '@/lib/table-columns'
import { getModuleOrThrow } from '@/modules'
import type { Series } from '@/types'

import type { FilterStatus, SeriesListState } from './use-series-list'

const mod = getModuleOrThrow('tv')

export function SeriesListLayout({ state: s }: { state: SeriesListState }) {
  return (
    <LibraryScreen<Series>
      moduleId={mod.id} theme={mod.themeColor} title={mod.name}
      addLabel={`Add ${mod.singularName}`} mediaLabel={mod.singularName} pluralMediaLabel={mod.pluralName}
      isLoading={s.isLoading} items={s.sortedSeries} groups={s.groups}
      toCell={(series) => seriesToCell(series, s.profileNameMap)}
      emptyTitle={`No ${mod.pluralName.toLowerCase()} found`}
      filterOptions={mod.filterOptions} statusFilters={s.statusFilters} onToggleFilter={(value) => s.handleToggleFilter(value as FilterStatus)}
      sortOptions={mod.sortOptions} sortField={s.sortField} sortDirection={s.sortDirection}
      onSortFieldChange={s.handleSortFieldChange} onColumnSort={s.handleColumnSort}
      view={s.seriesView} onViewChange={s.setSeriesView}
      posterSize={s.posterSize} onPosterSizeChange={s.setPosterSize}
      allTableColumns={s.allColumns} staticColumns={SERIES_COLUMNS}
      visibleColumnIds={s.seriesTableColumns} onTableColumnsChange={s.setSeriesTableColumns}
      renderContext={s.renderContext}
      editMode={s.editMode} selectedIds={s.selectedIds} filteredCount={s.filteredSeries.length}
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
