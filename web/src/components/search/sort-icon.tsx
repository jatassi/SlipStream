import { ArrowDown, ArrowUp, ArrowUpDown } from 'lucide-react'

import type { SortColumn, SortDirection } from './search-modal-types'

export function SortIcon({
  column,
  sortColumn,
  sortDirection,
}: {
  column: SortColumn
  sortColumn: SortColumn
  sortDirection: SortDirection
}) {
  if (sortColumn !== column) {
    return <ArrowUpDown className="text-muted-foreground ml-1 size-3" />
  }
  return sortDirection === 'asc' ? (
    <ArrowUp className="ml-1 size-3" />
  ) : (
    <ArrowDown className="ml-1 size-3" />
  )
}
