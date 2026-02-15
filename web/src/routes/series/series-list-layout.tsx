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

import { PageHeader } from '@/components/layout/page-header'
import { MediaDeleteDialog } from '@/components/media/media-delete-dialog'
import { MediaListContent } from '@/components/media/media-list-content'
import { MediaListFilters } from '@/components/media/media-list-filters'
import { MediaListToolbar } from '@/components/media/media-list-toolbar'
import { MediaPageActions } from '@/components/media/media-page-actions'
import { SeriesCard } from '@/components/series/series-card'
import { Skeleton } from '@/components/ui/skeleton'
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

export function SeriesListLayout({ state: s }: { state: SeriesListState }) {
  return (
    <div>
      <SeriesListHeader state={s} />
      {s.editMode ? (
        <MediaListToolbar
          selectedCount={s.selectedIds.size} totalCount={s.filteredSeries.length}
          qualityProfiles={s.qualityProfiles} isBulkUpdating={s.bulkUpdateMutation.isPending}
          onSelectAll={s.handleSelectAll} onMonitor={s.handleBulkMonitor}
          onChangeQualityProfile={s.handleBulkChangeQualityProfile} onDelete={() => s.setShowDeleteDialog(true)}
          theme="tv"
        />
      ) : null}
      <MediaListFilters
        filterOptions={FILTER_OPTIONS} sortOptions={SORT_OPTIONS}
        statusFilters={s.statusFilters} sortField={s.sortField} view={s.seriesView}
        posterSize={s.posterSize} visibleColumnIds={s.seriesTableColumns} columns={SERIES_COLUMNS}
        isLoading={s.isLoading}
        onToggleFilter={s.handleToggleFilter} onResetFilters={s.handleResetFilters}
        onSortFieldChange={s.handleSortFieldChange} onViewChange={s.handleViewChange}
        onPosterSizeChange={s.handlePosterSizeChange} onTableColumnsChange={s.setSeriesTableColumns}
        theme="tv"
      />
      <MediaListContent
        isLoading={s.isLoading} view={s.seriesView} items={s.sortedSeries}
        groups={s.groups} posterSize={s.posterSize} editMode={s.editMode}
        selectedIds={s.selectedIds} allFiltersSelected={s.allFiltersSelected}
        allColumns={s.allColumns} visibleColumnIds={s.seriesTableColumns}
        renderContext={s.renderContext} sortField={s.sortField}
        sortDirection={s.sortDirection} onSort={s.handleColumnSort} onToggleSelect={s.handleToggleSelect}
        theme="tv"
        renderCard={(series, opts) => <SeriesCard key={series.id} series={series} {...opts} />}
        emptyIcon={<Tv className="text-tv-500 size-8" />}
        emptyTitle="No series found"
        emptyAction={{ label: 'Add Series', onClick: () => document.getElementById('global-search')?.focus() }}
      />
      <MediaDeleteDialog
        open={s.showDeleteDialog} onOpenChange={s.setShowDeleteDialog}
        selectedCount={s.selectedIds.size} deleteFiles={s.deleteFiles}
        onDeleteFilesChange={s.setDeleteFiles} onConfirm={s.handleBulkDelete} isPending={s.bulkDeleteMutation.isPending}
        mediaLabel="Series" pluralMediaLabel="Series"
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
        <MediaPageActions
          isLoading={s.isLoading} editMode={s.editMode} isRefreshing={s.refreshAllMutation.isPending}
          onRefreshAll={s.handleRefreshAll} onEnterEdit={() => s.setEditMode(true)} onExitEdit={s.handleExitEditMode}
          theme="tv" addLabel="Add Series"
        />
      }
    />
  )
}
