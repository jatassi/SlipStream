import type { ReactNode } from 'react'

import { Link, useNavigate } from '@tanstack/react-router'
import { Plus } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import { LibraryOptions } from '@/components/media/library-options'
import { MediaDeleteDialog } from '@/components/media/media-delete-dialog'
import { MediaListToolbar } from '@/components/media/media-list-toolbar'
import { MediaTable } from '@/components/media/media-table'
import type { PosterCellItem } from '@/components/media/poster-cell'
import { PosterCell, PosterCellSkeleton } from '@/components/media/poster-cell'
import { PosterGrid, PosterGridItem } from '@/components/media/poster-grid'
import { Screen } from '@/components/screen/screen'
import { ChipRow } from '@/components/ui/chip-row'
import { Segmented } from '@/components/ui/segmented'
import { useViewport } from '@/hooks/use-viewport'
import type { MediaGroup } from '@/lib/grouping'
import type { ColumnDef, ColumnRenderContext } from '@/lib/table-columns'
import { getEnabledModules, getModuleOrThrow } from '@/modules'
import type { ModuleFilterOption, ModuleSortOption } from '@/modules/types'
import type { QualityProfile } from '@/types'

const SKELETON_KEYS = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i'] as const

export type LibraryScreenProps<T extends { id: number }> = {
  moduleId: string
  theme: string
  title: string
  addLabel: string
  mediaLabel: string
  pluralMediaLabel: string

  isLoading: boolean
  items: T[]
  groups: MediaGroup<T>[] | null
  toCell: (item: T) => PosterCellItem
  emptyTitle: string

  filterOptions: ModuleFilterOption[]
  statusFilters: string[]
  onToggleFilter: (value: string) => void

  sortOptions: ModuleSortOption[]
  sortField: string
  sortDirection: 'asc' | 'desc'
  onSortFieldChange: (value: string) => void
  onColumnSort: (field: string) => void

  view: 'grid' | 'table'
  onViewChange: (view: 'grid' | 'table') => void
  posterSize: number
  onPosterSizeChange: (size: number) => void

  allTableColumns: ColumnDef<T>[]
  staticColumns: ColumnDef<T>[]
  visibleColumnIds: string[]
  onTableColumnsChange: (ids: string[]) => void
  renderContext: ColumnRenderContext

  editMode: boolean
  selectedIds: Set<number>
  filteredCount: number
  onEnterEdit: () => void
  onExitEdit: () => void
  onToggleSelect: (id: number) => void
  onSelectAll: () => void

  qualityProfiles: QualityProfile[] | undefined
  isBulkUpdating: boolean
  isBulkDeleting: boolean
  isRefreshing: boolean
  onBulkMonitor: (monitored: boolean) => void
  onBulkChangeQualityProfile: (id: number) => void
  onBulkDelete: () => void
  onRefreshAll: () => void

  showDeleteDialog: boolean
  onShowDeleteDialog: (open: boolean) => void
  deleteFiles: boolean
  onDeleteFilesChange: (checked: boolean) => void
}

function ModuleSegmented({ moduleId }: { moduleId: string }) {
  const navigate = useNavigate()
  const modules = getEnabledModules()

  if (modules.length < 2) {
    return null
  }

  return (
    <div className="px-screen pb-4">
      <Segmented
        label="Library"
        value={moduleId}
        options={modules.map((mod) => ({ value: mod.id, label: mod.name }))}
        onChange={(next) => {
          void navigate({ to: getModuleOrThrow(next).basePath })
        }}
      />
    </div>
  )
}

function AddLink({ label }: { label: string }) {
  return (
    <Link
      to="/search"
      search={{ q: '' }}
      aria-label={label}
      className="press-dim focus-visible:ring-ring text-primary flex size-11 items-center justify-center rounded-full outline-none focus-visible:ring-[3px]"
    >
      <Plus className="size-6" />
    </Link>
  )
}

function LibraryTrailing<T extends { id: number }>(props: LibraryScreenProps<T>) {
  return (
    <div className="flex items-center">
      <AddLink label={props.addLabel} />
      <LibraryOptions
        pluralMediaLabel={props.pluralMediaLabel}
        sortOptions={props.sortOptions}
        sortField={props.sortField}
        onSortFieldChange={props.onSortFieldChange}
        view={props.view}
        onViewChange={props.onViewChange}
        posterSize={props.posterSize}
        onPosterSizeChange={props.onPosterSizeChange}
        columns={props.staticColumns}
        visibleColumnIds={props.visibleColumnIds}
        onTableColumnsChange={props.onTableColumnsChange}
        editMode={props.editMode}
        onEnterEdit={props.onEnterEdit}
        onExitEdit={props.onExitEdit}
        isRefreshing={props.isRefreshing}
        onRefreshAll={props.onRefreshAll}
      />
    </div>
  )
}

type GridProps<T> = {
  items: T[]
  label: string
  posterSize: number
  toCell: (item: T) => PosterCellItem
  editMode: boolean
  selectedIds: Set<number>
  onToggleSelect: (id: number) => void
}

function LibraryGrid<T extends { id: number }>({
  items,
  label,
  posterSize,
  toCell,
  editMode,
  selectedIds,
  onToggleSelect,
}: GridProps<T>) {
  return (
    <PosterGrid label={label} posterSize={posterSize}>
      {items.map((item) => (
        <PosterGridItem key={item.id}>
          <PosterCell
            item={toCell(item)}
            editMode={editMode}
            selected={selectedIds.has(item.id)}
            onToggleSelect={onToggleSelect}
          />
        </PosterGridItem>
      ))}
    </PosterGrid>
  )
}

function gridProps<T extends { id: number }>(
  props: LibraryScreenProps<T>,
  items: T[],
): GridProps<T> {
  return {
    items,
    label: props.title,
    posterSize: props.posterSize,
    toCell: props.toCell,
    editMode: props.editMode,
    selectedIds: props.selectedIds,
    onToggleSelect: props.onToggleSelect,
  }
}

function GroupHeading({ label, count }: { label: string; count: number }) {
  return (
    <div className="px-screen text-footnote text-muted-foreground flex items-center gap-2 pt-2 pb-3 font-semibold">
      <span>{label}</span>
      <span className="nums">{count}</span>
    </div>
  )
}

function LibraryEmpty<T extends { id: number }>(props: LibraryScreenProps<T>) {
  return (
    <EmptyState
      title={props.emptyTitle}
      description={
        props.statusFilters.length === 0
          ? `Add ${props.pluralMediaLabel.toLowerCase()} to get started`
          : 'Try adjusting your filters'
      }
    />
  )
}

function LibraryTable<T extends { id: number }>(props: LibraryScreenProps<T>) {
  return (
    <div className="px-screen">
      <MediaTable
        items={props.items}
        columns={props.allTableColumns}
        visibleColumnIds={props.visibleColumnIds}
        renderContext={props.renderContext}
        sortField={props.sortField}
        sortDirection={props.sortDirection}
        onSort={props.onColumnSort}
        editMode={props.editMode}
        selectedIds={props.selectedIds}
        onToggleSelect={props.onToggleSelect}
        theme={props.theme}
      />
    </div>
  )
}

function LibraryEditToolbar<T extends { id: number }>(props: LibraryScreenProps<T>) {
  if (!props.editMode) {
    return null
  }
  return (
    <div className="px-screen pb-4">
      <MediaListToolbar
        selectedCount={props.selectedIds.size}
        totalCount={props.filteredCount}
        qualityProfiles={props.qualityProfiles}
        isBulkUpdating={props.isBulkUpdating}
        onSelectAll={props.onSelectAll}
        onMonitor={props.onBulkMonitor}
        onChangeQualityProfile={props.onBulkChangeQualityProfile}
        onDelete={() => {
          props.onShowDeleteDialog(true)
        }}
        theme={props.theme}
      />
    </div>
  )
}

function LibraryContent<T extends { id: number }>(props: LibraryScreenProps<T>): ReactNode {
  if (props.isLoading) {
    return (
      <PosterGrid label={props.title} posterSize={props.posterSize}>
        {SKELETON_KEYS.map((key) => (
          <PosterGridItem key={key}>
            <PosterCellSkeleton />
          </PosterGridItem>
        ))}
      </PosterGrid>
    )
  }

  if (props.items.length === 0) {
    return <LibraryEmpty {...props} />
  }

  if (props.view === 'table') {
    return <LibraryTable {...props} />
  }

  if (props.groups) {
    return (
      <div className="space-y-4">
        {props.groups.map((group) => (
          <div key={group.key}>
            <GroupHeading label={group.label} count={group.items.length} />
            <LibraryGrid {...gridProps(props, group.items)} />
          </div>
        ))}
      </div>
    )
  }

  return <LibraryGrid {...gridProps(props, props.items)} />
}

export function LibraryScreen<T extends { id: number }>(props: LibraryScreenProps<T>) {
  const shell = useViewport()
  const view = shell === 'phone' ? 'grid' : props.view
  const resolved = { ...props, view }

  return (
    <Screen title={props.title} trailing={<LibraryTrailing {...resolved} />}>
      <ModuleSegmented moduleId={props.moduleId} />
      <ChipRow
        label={`Filter ${props.pluralMediaLabel.toLowerCase()}`}
        options={props.filterOptions}
        selected={props.statusFilters}
        onToggle={props.onToggleFilter}
      />
      <LibraryEditToolbar {...props} />
      <LibraryContent {...resolved} />
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
    </Screen>
  )
}
