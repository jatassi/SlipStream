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
import type { Movie } from '@/types'

type MovieTableProps = {
  movies: Movie[]
  columns: ColumnDef<Movie>[]
  visibleColumnIds: string[]
  renderContext: ColumnRenderContext
  sortField?: string
  sortDirection?: 'asc' | 'desc'
  onSort?: (field: string) => void
  editMode?: boolean
  selectedIds?: Set<number>
  onToggleSelect?: (id: number) => void
}

export function MovieTable({
  movies,
  columns,
  visibleColumnIds,
  renderContext,
  sortField,
  sortDirection,
  onSort,
  editMode,
  selectedIds,
  onToggleSelect,
}: MovieTableProps) {
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
          {movies.map((movie) => {
            const selected = selectedIds?.has(movie.id)
            return (
              <TableRow
                key={movie.id}
                data-state={selected ? 'selected' : undefined}
                className={cn(editMode && 'cursor-pointer')}
                onClick={editMode && onToggleSelect ? () => onToggleSelect(movie.id) : undefined}
              >
                {editMode ? (
                  <TableCell>
                    <Checkbox
                      checked={selected}
                      onCheckedChange={() => onToggleSelect?.(movie.id)}
                      onClick={(e) => e.stopPropagation()}
                      className="data-checked:bg-movie-500 data-checked:border-movie-500"
                    />
                  </TableCell>
                ) : null}
                {visibleColumns.map((col) => (
                  <TableCell
                    key={col.id}
                    className={col.cellClassName}
                    style={col.minWidth ? { minWidth: col.minWidth } : undefined}
                  >
                    {col.render(movie, renderContext)}
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
