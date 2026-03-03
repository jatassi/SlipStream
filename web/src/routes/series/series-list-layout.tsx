import {
  ArrowDownCircle,
  ArrowUpCircle,
  Binoculars,
  CheckCircle,
  CircleStop,
  Clock,
  Eye,
  Play,
  Tv,
  XCircle,
} from 'lucide-react'

import { MediaListLayout } from '@/components/media/media-list-layout'
import { SeriesCard } from '@/components/series/series-card'
import { SERIES_COLUMNS } from '@/lib/table-columns'

import type { FilterStatus, SeriesListState, SortField } from './use-series-list'

const FILTER_OPTIONS: { value: FilterStatus; label: string; icon: typeof Eye }[] = [
  { value: 'monitored', label: 'Monitored', icon: Eye },
  { value: 'continuing', label: 'Continuing', icon: Play },
  { value: 'ended', label: 'Ended', icon: CircleStop },
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
  { value: 'nextAirDate', label: 'Next Air Date' },
  { value: 'dateAdded', label: 'Date Added' },
  { value: 'rootFolder', label: 'Root Folder' },
  { value: 'sizeOnDisk', label: 'Size on Disk' },
]

const EMPTY_ACTION = { label: 'Add Series', onClick: () => document.getElementById('global-search')?.focus() }

export function SeriesListLayout({ state: s }: { state: SeriesListState }) {
  return (
    <MediaListLayout
      theme="tv" title="Series" addLabel="Add Series"
      mediaLabel="Series" pluralMediaLabel="Series" libraryCount={s.seriesList?.length}
      filterOptions={FILTER_OPTIONS} sortOptions={SORT_OPTIONS}
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
      emptyIcon={<Tv className="text-tv-500 size-8" />} emptyTitle="No series found"
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
