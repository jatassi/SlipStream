import type { ReactNode } from 'react'

import type { LucideIcon } from 'lucide-react'

import { PageHeader } from '@/components/layout/page-header'
import { MediaDeleteDialog } from '@/components/media/media-delete-dialog'
import { MediaListContent } from '@/components/media/media-list-content'
import { MediaListFilters } from '@/components/media/media-list-filters'
import { MediaListToolbar } from '@/components/media/media-list-toolbar'
import { MediaPageActions } from '@/components/media/media-page-actions'
import { Skeleton } from '@/components/ui/skeleton'
import type { MediaGroup } from '@/lib/grouping'
import type { ColumnDef, ColumnRenderContext } from '@/lib/table-columns'
import type { QualityProfile } from '@/types'

type FilterOption<F extends string> = { value: F; label: string; icon: LucideIcon }
type SortOption<S extends string> = { value: S; label: string }

export type MediaListLayoutProps<T extends { id: number }, F extends string, S extends string> = {
  // theming & labels
  theme: string
  title: string
  addLabel: string
  mediaLabel: string
  pluralMediaLabel: string
  libraryCount: number | undefined

  // filter / sort config
  filterOptions: FilterOption<F>[]
  sortOptions: SortOption<S>[]

  // table columns
  allTableColumns: ColumnDef<T>[]
  staticColumns: ColumnDef<T>[]
  visibleColumnIds: string[]
  onTableColumnsChange: (cols: string[]) => void

  // loading / state
  isLoading: boolean
  editMode: boolean
  selectedIds: Set<number>
  filteredCount: number
  allFiltersSelected: boolean

  // filter / sort state
  statusFilters: F[]
  sortField: S
  sortDirection: 'asc' | 'desc'
  view: 'grid' | 'table'
  posterSize: number

  // items
  items: T[]
  groups: MediaGroup<T>[] | null
  renderContext: ColumnRenderContext

  // quality profiles (for bulk toolbar)
  qualityProfiles: QualityProfile[] | undefined
  isBulkUpdating: boolean

  // delete dialog
  showDeleteDialog: boolean
  deleteFiles: boolean
  isBulkDeleting: boolean
  isRefreshing: boolean

  // empty state
  emptyIcon: ReactNode
  emptyTitle: string
  emptyAction?: { label: string; onClick: () => void }

  // card renderer
  renderCard: (item: T, opts: { editMode?: boolean; selected?: boolean; onToggleSelect?: (id: number) => void }) => ReactNode

  // callbacks
  onToggleFilter: (v: F) => void
  onResetFilters: () => void
  onSortFieldChange: (v: string) => void
  onViewChange: (v: string[]) => void
  onPosterSizeChange: (v: number | readonly number[]) => void
  onColumnSort: (field: string) => void
  onToggleSelect: (id: number) => void
  onSelectAll: () => void
  onBulkMonitor: (monitored: boolean) => void
  onBulkChangeQualityProfile: (id: number) => void
  onDelete: () => void
  onRefreshAll: () => void
  onEnterEdit: () => void
  onExitEdit: () => void
  onShowDeleteDialog: (open: boolean) => void
  onDeleteFilesChange: (checked: boolean) => void
  onBulkDelete: () => void
}

export function MediaListLayout<T extends { id: number }, F extends string, S extends string>(
  props: MediaListLayoutProps<T, F, S>,
) {
  return (
    <div>
      <MediaListHeader {...props} />
      <MediaListEditToolbar {...props} />
      <MediaListFiltersRow {...props} />
      <MediaListContentArea {...props} />
      <MediaListDeleteDialog {...props} />
    </div>
  )
}

function MediaListEditToolbar<T extends { id: number }, F extends string, S extends string>(
  props: MediaListLayoutProps<T, F, S>,
) {
  if (!props.editMode) { return null }
  return (
    <MediaListToolbar
      selectedCount={props.selectedIds.size}
      totalCount={props.filteredCount}
      qualityProfiles={props.qualityProfiles}
      isBulkUpdating={props.isBulkUpdating}
      onSelectAll={props.onSelectAll}
      onMonitor={props.onBulkMonitor}
      onChangeQualityProfile={props.onBulkChangeQualityProfile}
      onDelete={props.onDelete}
      theme={props.theme}
    />
  )
}

function MediaListFiltersRow<T extends { id: number }, F extends string, S extends string>(
  props: MediaListLayoutProps<T, F, S>,
) {
  return (
    <MediaListFilters
      filterOptions={props.filterOptions}
      sortOptions={props.sortOptions}
      statusFilters={props.statusFilters}
      sortField={props.sortField}
      sortDirection={props.sortDirection}
      view={props.view}
      posterSize={props.posterSize}
      visibleColumnIds={props.visibleColumnIds}
      columns={props.staticColumns}
      isLoading={props.isLoading}
      onToggleFilter={props.onToggleFilter}
      onResetFilters={props.onResetFilters}
      onSortFieldChange={props.onSortFieldChange}
      onViewChange={props.onViewChange}
      onPosterSizeChange={props.onPosterSizeChange}
      onTableColumnsChange={props.onTableColumnsChange}
      theme={props.theme}
    />
  )
}

function MediaListContentArea<T extends { id: number }, F extends string, S extends string>(
  props: MediaListLayoutProps<T, F, S>,
) {
  return (
    <MediaListContent
      isLoading={props.isLoading}
      view={props.view}
      items={props.items}
      groups={props.groups}
      posterSize={props.posterSize}
      editMode={props.editMode}
      selectedIds={props.selectedIds}
      allFiltersSelected={props.allFiltersSelected}
      allColumns={props.allTableColumns}
      visibleColumnIds={props.visibleColumnIds}
      renderContext={props.renderContext}
      sortField={props.sortField}
      sortDirection={props.sortDirection}
      onSort={props.onColumnSort}
      onToggleSelect={props.onToggleSelect}
      theme={props.theme}
      renderCard={props.renderCard}
      emptyIcon={props.emptyIcon}
      emptyTitle={props.emptyTitle}
      emptyAction={props.emptyAction}
    />
  )
}

function MediaListDeleteDialog<T extends { id: number }, F extends string, S extends string>(
  props: MediaListLayoutProps<T, F, S>,
) {
  return (
    <MediaDeleteDialog
      open={props.showDeleteDialog}
      onOpenChange={props.onShowDeleteDialog}
      selectedCount={props.selectedIds.size}
      deleteFiles={props.deleteFiles}
      onDeleteFilesChange={props.onDeleteFilesChange}
      onConfirm={props.onBulkDelete}
      isPending={props.isBulkDeleting}
      mediaLabel={props.mediaLabel}
      pluralMediaLabel={props.pluralMediaLabel}
    />
  )
}

function MediaListHeader<T extends { id: number }, F extends string, S extends string>({
  title,
  isLoading,
  libraryCount,
  editMode,
  isRefreshing,
  onRefreshAll,
  onEnterEdit,
  onExitEdit,
  theme,
  addLabel,
  pluralMediaLabel,
}: Pick<
  MediaListLayoutProps<T, F, S>,
  | 'title'
  | 'isLoading'
  | 'libraryCount'
  | 'editMode'
  | 'isRefreshing'
  | 'onRefreshAll'
  | 'onEnterEdit'
  | 'onExitEdit'
  | 'theme'
  | 'addLabel'
  | 'pluralMediaLabel'
>) {
  return (
    <PageHeader
      title={title}
      description={
        isLoading
          ? <Skeleton className="h-4 w-36" />
          : `${libraryCount ?? 0} ${pluralMediaLabel.toLowerCase()} in library`
      }
      actions={
        <MediaPageActions
          isLoading={isLoading}
          editMode={editMode}
          isRefreshing={isRefreshing}
          onRefreshAll={onRefreshAll}
          onEnterEdit={onEnterEdit}
          onExitEdit={onExitEdit}
          theme={theme}
          addLabel={addLabel}
        />
      }
    />
  )
}
