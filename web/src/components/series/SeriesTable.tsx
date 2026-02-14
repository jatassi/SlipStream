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
        <TableHeader>
          <TableRow>
            {editMode ? <TableHead className="w-[40px]" /> : null}
            {visibleColumns.map((col) => (
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
                    sortDirection === 'asc' ? (
                      <ChevronUp className="size-3.5" />
                    ) : (
                      <ChevronDown className="size-3.5" />
                    )
                  ) : null}
                </span>
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {series.map((s) => {
            const selected = selectedIds?.has(s.id)
            return (
              <TableRow
                key={s.id}
                data-state={selected ? 'selected' : undefined}
                className={cn(editMode && 'cursor-pointer')}
                onClick={editMode && onToggleSelect ? () => onToggleSelect(s.id) : undefined}
              >
                {editMode ? (
                  <TableCell>
                    <Checkbox
                      checked={selected}
                      onCheckedChange={() => onToggleSelect?.(s.id)}
                      onClick={(e) => e.stopPropagation()}
                      className="data-checked:bg-tv-500 data-checked:border-tv-500"
                    />
                  </TableCell>
                ) : null}
                {visibleColumns.map((col) => (
                  <TableCell
                    key={col.id}
                    className={col.cellClassName}
                    style={col.minWidth ? { minWidth: col.minWidth } : undefined}
                  >
                    {col.render(s, renderContext)}
                  </TableCell>
                ))}
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </div>
  )
}
