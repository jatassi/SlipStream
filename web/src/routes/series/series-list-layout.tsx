import { MediaListLayout } from '@/components/media/media-list-layout'
import { SeriesCard } from '@/components/series/series-card'
import { SERIES_COLUMNS } from '@/lib/table-columns'
import { getModuleOrThrow } from '@/modules'

import type { SeriesListState } from './use-series-list'

const mod = getModuleOrThrow('tv')
const ModIcon = mod.icon

const EMPTY_ACTION = { label: `Add ${mod.singularName}`, onClick: () => document.getElementById('global-search')?.focus() }

export function SeriesListLayout({ state: s }: { state: SeriesListState }) {
  return (
    <MediaListLayout
      theme={mod.themeColor} title={mod.name} addLabel={`Add ${mod.singularName}`}
      mediaLabel={mod.singularName} pluralMediaLabel={mod.pluralName} libraryCount={s.seriesList?.length}
      filterOptions={mod.filterOptions as never} sortOptions={mod.sortOptions as never}
      allTableColumns={s.allColumns} staticColumns={SERIES_COLUMNS}
      visibleColumnIds={s.seriesTableColumns} onTableColumnsChange={s.setSeriesTableColumns}
      isLoading={s.isLoading} editMode={s.editMode} selectedIds={s.selectedIds}
      filteredCount={s.filteredSeries.length} allFiltersSelected={s.allFiltersSelected}
      statusFilters={s.statusFilters} sortField={s.sortField}
      sortDirection={s.sortDirection} view={s.seriesView} posterSize={s.posterSize}
      items={s.sortedSeries} groups={s.groups} renderContext={s.renderContext}
      qualityProfiles={s.qualityProfiles} isBulkUpdating={s.bulkUpdateMutation.isPending}
      showDeleteDialog={s.showDeleteDialog} deleteFiles={s.deleteFiles}
      isBulkDeleting={s.bulkDeleteMutation.isPending} isRefreshing={s.refreshAllMutation.isPending}
      emptyIcon={<ModIcon className="text-tv-500 size-8" />} emptyTitle={`No ${mod.pluralName.toLowerCase()} found`}
      emptyAction={EMPTY_ACTION}
      renderCard={(series, opts) => <SeriesCard key={series.id} series={series} {...opts} />}
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
