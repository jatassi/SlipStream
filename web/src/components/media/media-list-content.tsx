import type { ReactNode } from 'react'

import { EmptyState } from '@/components/data/empty-state'
import { LoadingState } from '@/components/data/loading-state'
import { GroupedMediaGrid } from '@/components/media/grouped-media-grid'
import { MediaGrid } from '@/components/media/media-grid'
import { MediaTable } from '@/components/media/media-table'
import type { MediaGroup } from '@/lib/grouping'
import type { ColumnDef, ColumnRenderContext } from '@/lib/table-columns'

type Props<T> = {
  isLoading: boolean
  view: 'grid' | 'table'
  items: T[]
  groups: MediaGroup<T>[] | null
  posterSize: number
  editMode: boolean
  selectedIds: Set<number>
  allFiltersSelected: boolean
  allColumns: ColumnDef<T>[]
  visibleColumnIds: string[]
  renderContext: ColumnRenderContext
  sortField: string
  sortDirection: 'asc' | 'desc'
  onSort: (field: string) => void
  onToggleSelect: (id: number) => void
  theme: 'movie' | 'tv'
  renderCard: (item: T, opts: { editMode?: boolean; selected?: boolean; onToggleSelect?: (id: number) => void }) => ReactNode
  emptyIcon: ReactNode
  emptyTitle: string
  emptyAction?: { label: string; onClick: () => void }
}

export function MediaListContent<T extends { id: number }>(props: Props<T>) {
  if (props.isLoading) {
    return <LoadingState variant={props.view === 'grid' ? 'card' : 'list'} posterSize={props.posterSize} theme={props.theme} />
  }
  if (props.items.length === 0) {
    return (
      <EmptyState
        icon={props.emptyIcon}
        title={props.emptyTitle}
        description={props.allFiltersSelected ? `Add your first ${props.theme === 'movie' ? 'movie' : 'series'} to get started` : 'Try adjusting your filters'}
        action={props.allFiltersSelected ? props.emptyAction : undefined}
      />
    )
  }
  if (props.view === 'table') {
    return (
      <MediaTable
        items={props.items}
        columns={props.allColumns}
        visibleColumnIds={props.visibleColumnIds}
        renderContext={props.renderContext}
        sortField={props.sortField}
        sortDirection={props.sortDirection}
        onSort={props.onSort}
        editMode={props.editMode}
        selectedIds={props.selectedIds}
        onToggleSelect={props.onToggleSelect}
        theme={props.theme}
      />
    )
  }
  if (props.groups) {
    return <GroupedMediaGrid groups={props.groups} renderGrid={(items) => <MediaGrid items={items} renderCard={props.renderCard} posterSize={props.posterSize} editMode={props.editMode} selectedIds={props.selectedIds} onToggleSelect={props.onToggleSelect} />} />
  }
  return <MediaGrid items={props.items} renderCard={props.renderCard} posterSize={props.posterSize} editMode={props.editMode} selectedIds={props.selectedIds} onToggleSelect={props.onToggleSelect} />
}
