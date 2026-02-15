import { ChevronDown, ChevronUp } from 'lucide-react'

import { Checkbox } from '@/components/ui/checkbox'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { ColumnDef, ColumnRenderContext } from '@/lib/table-columns'
import { cn } from '@/lib/utils'

function SortIcon({ direction }: { direction?: 'asc' | 'desc' }) {
  if (direction === 'asc') {
    return <ChevronUp className="size-3.5" />
  }
  return <ChevronDown className="size-3.5" />
}

type MediaTableProps<T extends { id: number }> = {
  items: T[]
  columns: ColumnDef<T>[]
  visibleColumnIds: string[]
  renderContext: ColumnRenderContext
  sortField?: string
  sortDirection?: 'asc' | 'desc'
  onSort?: (field: string) => void
  editMode?: boolean
  selectedIds?: Set<number>
  onToggleSelect?: (id: number) => void
  theme: 'movie' | 'tv'
}

export function MediaTable<T extends { id: number }>({
  items,
  columns,
  visibleColumnIds,
  renderContext,
  sortField,
  sortDirection,
  onSort,
  editMode,
  selectedIds,
  onToggleSelect,
  theme,
}: MediaTableProps<T>) {
  const visibleColumns = columns.filter((col) => !col.hideable || visibleColumnIds.includes(col.id))

  return (
    <div className="rounded-md border">
      <Table>
        <MediaTableHeader
          columns={visibleColumns}
          editMode={editMode}
          sortField={sortField}
          sortDirection={sortDirection}
          onSort={onSort}
        />
        <TableBody>
          {items.map((item) => (
            <MediaTableRow
              key={item.id}
              item={item}
              columns={visibleColumns}
              renderContext={renderContext}
              editMode={editMode}
              selected={selectedIds?.has(item.id)}
              onToggleSelect={onToggleSelect}
              theme={theme}
            />
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function MediaTableHeader<T extends { id: number }>({
  columns,
  editMode,
  sortField,
  sortDirection,
  onSort,
}: {
  columns: ColumnDef<T>[]
  editMode?: boolean
  sortField?: string
  sortDirection?: 'asc' | 'desc'
  onSort?: (field: string) => void
}) {
  return (
    <TableHeader>
      <TableRow>
        {editMode ? <TableHead className="w-[40px]" /> : null}
        {columns.map((col) => (
          <TableHead
            key={col.id}
            className={cn(
              col.headerClassName,
              col.sortField && 'hover:text-foreground cursor-pointer select-none',
            )}
            style={col.minWidth ? { minWidth: col.minWidth } : undefined}
            onClick={
              col.sortField && onSort ? () => col.sortField && onSort(col.sortField) : undefined
            }
          >
            <span className="inline-flex items-center gap-1">
              {col.label}
              {col.sortField && sortField === col.sortField ? (
                <SortIcon direction={sortDirection} />
              ) : null}
            </span>
          </TableHead>
        ))}
      </TableRow>
    </TableHeader>
  )
}

function MediaTableRow<T extends { id: number }>({
  item,
  columns,
  renderContext,
  editMode,
  selected,
  onToggleSelect,
  theme,
}: {
  item: T
  columns: ColumnDef<T>[]
  renderContext: ColumnRenderContext
  editMode?: boolean
  selected?: boolean
  onToggleSelect?: (id: number) => void
  theme: 'movie' | 'tv'
}) {
  const checkboxClassName =
    theme === 'movie'
      ? 'data-checked:bg-movie-500 data-checked:border-movie-500'
      : 'data-checked:bg-tv-500 data-checked:border-tv-500'

  return (
    <TableRow
      data-state={selected ? 'selected' : undefined}
      className={cn(editMode && 'cursor-pointer')}
      onClick={editMode && onToggleSelect ? () => onToggleSelect(item.id) : undefined}
    >
      {editMode ? (
        <TableCell>
          <Checkbox
            checked={selected}
            onCheckedChange={() => onToggleSelect?.(item.id)}
            onClick={(e) => e.stopPropagation()}
            className={checkboxClassName}
          />
        </TableCell>
      ) : null}
      {columns.map((col) => (
        <TableCell
          key={col.id}
          className={col.cellClassName}
          style={col.minWidth ? { minWidth: col.minWidth } : undefined}
        >
          {col.render(item, renderContext)}
        </TableCell>
      ))}
    </TableRow>
  )
}
