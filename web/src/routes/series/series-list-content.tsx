import { Tv } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import { LoadingState } from '@/components/data/loading-state'
import { GroupedSeriesGrid } from '@/components/series/grouped-series-grid'
import { SeriesGrid } from '@/components/series/series-grid'
import { SeriesTable } from '@/components/series/series-table'
import type { MediaGroup } from '@/lib/grouping'
import type { ColumnDef, ColumnRenderContext } from '@/lib/table-columns'
import type { Series } from '@/types'

type Props = {
  isLoading: boolean
  seriesView: 'grid' | 'table'
  sortedSeries: Series[]
  groups: MediaGroup<Series>[] | null
  posterSize: number
  editMode: boolean
  selectedIds: Set<number>
  allFiltersSelected: boolean
  allColumns: ColumnDef<Series>[]
  seriesTableColumns: string[]
  renderContext: ColumnRenderContext
  sortField: string
  sortDirection: 'asc' | 'desc'
  onSort: (field: string) => void
  onToggleSelect: (id: number) => void
}

export function SeriesListContent(props: Props) {
  if (props.isLoading) {
    return <LoadingState variant={props.seriesView === 'grid' ? 'card' : 'list'} posterSize={props.posterSize} theme="tv" />
  }
  if (props.sortedSeries.length === 0) {
    return <SeriesEmptyState allFiltersSelected={props.allFiltersSelected} />
  }
  if (props.seriesView === 'table') {
    return <TableView {...props} />
  }
  if (props.groups) {
    return <GroupedSeriesGrid groups={props.groups} posterSize={props.posterSize} editMode={props.editMode} selectedIds={props.selectedIds} onToggleSelect={props.onToggleSelect} />
  }
  return <SeriesGrid series={props.sortedSeries} posterSize={props.posterSize} editMode={props.editMode} selectedIds={props.selectedIds} onToggleSelect={props.onToggleSelect} />
}

function SeriesEmptyState({ allFiltersSelected }: { allFiltersSelected: boolean }) {
  return (
    <EmptyState
      icon={<Tv className="text-tv-500 size-8" />}
      title="No series found"
      description={allFiltersSelected ? 'Add your first series to get started' : 'Try adjusting your filters'}
      action={allFiltersSelected ? { label: 'Add Series', onClick: () => document.getElementById('global-search')?.focus() } : undefined}
    />
  )
}

function TableView(props: Props) {
  return (
    <SeriesTable
      series={props.sortedSeries}
      columns={props.allColumns}
      visibleColumnIds={props.seriesTableColumns}
      renderContext={props.renderContext}
      sortField={props.sortField}
      sortDirection={props.sortDirection}
      onSort={props.onSort}
      editMode={props.editMode}
      selectedIds={props.selectedIds}
      onToggleSelect={props.onToggleSelect}
    />
  )
}
