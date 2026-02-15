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
import type { Series } from '@/types'

function SortIcon({ direction }: { direction: 'asc' | 'desc' }) {
  if (direction === 'asc') {
    return <ChevronUp className="size-3.5" />
  }
  return <ChevronDown className="size-3.5" />
}

type SeriesTableProps = {
  series: Series[]
  columns: ColumnDef<Series>[]
  visibleColumnIds: string[]
  renderContext: ColumnRenderContext
  sortField?: string
  sortDirection?: 'asc' | 'desc'
  onSort?: (field: string) => void
  editMode?: boolean
  selectedIds?: Set<number>
  onToggleSelect?: (id: number) => void
}

export function SeriesTable({
  series,
  columns,
  visibleColumnIds,
  renderContext,
  sortField,
  sortDirection,
  onSort,
  editMode,
  selectedIds,
  onToggleSelect,
}: SeriesTableProps) {
  const visibleColumns = columns.filter((col) => !col.hideable || visibleColumnIds.includes(col.id))

  return (
    <div className="rounded-md border">
      <Table>
        <SeriesTableHeader
          columns={visibleColumns}
          editMode={editMode}
          sortField={sortField}
          sortDirection={sortDirection}
          onSort={onSort}
        />
        <TableBody>
          {series.map((s) => (
            <SeriesTableRow
              key={s.id}
              series={s}
              columns={visibleColumns}
              renderContext={renderContext}
              editMode={editMode}
              selected={selectedIds?.has(s.id)}
              onToggleSelect={onToggleSelect}
            />
          ))}
        </TableBody>
      </Table>
    </div>
  )
}

function SeriesTableHeader({
  columns,
  editMode,
  sortField,
  sortDirection,
  onSort,
}: {
  columns: ColumnDef<Series>[]
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
                <SortIcon direction={sortDirection ?? 'asc'} />
              ) : null}
            </span>
          </TableHead>
        ))}
      </TableRow>
    </TableHeader>
  )
}

function SeriesTableRow({
  series,
  columns,
  renderContext,
  editMode,
  selected,
  onToggleSelect,
}: {
  series: Series
  columns: ColumnDef<Series>[]
  renderContext: ColumnRenderContext
  editMode?: boolean
  selected?: boolean
  onToggleSelect?: (id: number) => void
}) {
  return (
    <TableRow
      data-state={selected ? 'selected' : undefined}
      className={cn(editMode && 'cursor-pointer')}
      onClick={editMode && onToggleSelect ? () => onToggleSelect(series.id) : undefined}
    >
      {editMode ? (
        <TableCell>
          <Checkbox
            checked={selected}
            onCheckedChange={() => onToggleSelect?.(series.id)}
            onClick={(e) => e.stopPropagation()}
            className="data-checked:bg-tv-500 data-checked:border-tv-500"
          />
        </TableCell>
      ) : null}
      {columns.map((col) => (
        <TableCell
          key={col.id}
          className={col.cellClassName}
          style={col.minWidth ? { minWidth: col.minWidth } : undefined}
        >
          {col.render(series, renderContext)}
        </TableCell>
      ))}
    </TableRow>
  )
}
