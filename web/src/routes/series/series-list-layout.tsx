import { PageHeader } from '@/components/layout/page-header'
import { Skeleton } from '@/components/ui/skeleton'

import { SeriesDeleteDialog } from './series-delete-dialog'
import { SeriesListContent } from './series-list-content'
import { SeriesListFilters } from './series-list-filters'
import { SeriesListToolbar } from './series-list-toolbar'
import { SeriesPageActions } from './series-page-actions'
import type { SeriesListState } from './use-series-list'

export function SeriesListLayout({ state: s }: { state: SeriesListState }) {
  return (
    <div>
      <SeriesListHeader state={s} />
      {s.editMode ? (
        <SeriesListToolbar
          selectedCount={s.selectedIds.size} totalCount={s.filteredSeries.length}
          qualityProfiles={s.qualityProfiles} isBulkUpdating={s.bulkUpdateMutation.isPending}
          onSelectAll={s.handleSelectAll} onMonitor={s.handleBulkMonitor}
          onChangeQualityProfile={s.handleBulkChangeQualityProfile} onDelete={() => s.setShowDeleteDialog(true)}
        />
      ) : null}
      <SeriesListFilters
        statusFilters={s.statusFilters} sortField={s.sortField} seriesView={s.seriesView}
        posterSize={s.posterSize} seriesTableColumns={s.seriesTableColumns} isLoading={s.isLoading}
        onToggleFilter={s.handleToggleFilter} onResetFilters={s.handleResetFilters}
        onSortFieldChange={s.handleSortFieldChange} onViewChange={s.handleViewChange}
        onPosterSizeChange={s.handlePosterSizeChange} onTableColumnsChange={s.setSeriesTableColumns}
      />
      <SeriesListContent
        isLoading={s.isLoading} seriesView={s.seriesView} sortedSeries={s.sortedSeries}
        groups={s.groups} posterSize={s.posterSize} editMode={s.editMode}
        selectedIds={s.selectedIds} allFiltersSelected={s.allFiltersSelected}
        allColumns={s.allColumns} seriesTableColumns={s.seriesTableColumns}
        renderContext={s.renderContext} sortField={s.sortField}
        sortDirection={s.sortDirection} onSort={s.handleColumnSort} onToggleSelect={s.handleToggleSelect}
      />
      <SeriesDeleteDialog
        open={s.showDeleteDialog} onOpenChange={s.setShowDeleteDialog}
        selectedCount={s.selectedIds.size} deleteFiles={s.deleteFiles}
        onDeleteFilesChange={s.setDeleteFiles} onConfirm={s.handleBulkDelete} isPending={s.bulkDeleteMutation.isPending}
      />
    </div>
  )
}

function SeriesListHeader({ state: s }: { state: SeriesListState }) {
  return (
    <PageHeader
      title="Series"
      description={s.isLoading ? <Skeleton className="h-4 w-36" /> : `${s.seriesList?.length ?? 0} series in library`}
      actions={
        <SeriesPageActions
          isLoading={s.isLoading} editMode={s.editMode} isRefreshing={s.refreshAllMutation.isPending}
          onRefreshAll={s.handleRefreshAll} onEnterEdit={() => s.setEditMode(true)} onExitEdit={s.handleExitEditMode}
        />
      }
    />
  )
}
