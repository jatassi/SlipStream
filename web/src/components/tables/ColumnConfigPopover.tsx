import { Columns3 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import type { ColumnDef } from '@/lib/table-columns'

interface ColumnConfigPopoverProps<T> {
  columns: ColumnDef<T>[]
  visibleColumnIds: string[]
  onVisibleColumnsChange: (ids: string[]) => void
  theme: 'movie' | 'tv'
}

export function ColumnConfigPopover<T>({
  columns,
  visibleColumnIds,
  onVisibleColumnsChange,
  theme,
}: ColumnConfigPopoverProps<T>) {
  const hideableColumns = columns.filter((c) => c.hideable)
  const accentClass = theme === 'movie' ? 'text-movie-400' : 'text-tv-400'

  const toggleColumn = (id: string) => {
    if (visibleColumnIds.includes(id)) {
      onVisibleColumnsChange(visibleColumnIds.filter((c) => c !== id))
    } else {
      onVisibleColumnsChange([...visibleColumnIds, id])
    }
  }

  return (
    <Popover>
      <PopoverTrigger render={<Button variant="outline" size="sm" />}>
        <Columns3 className={cn('size-4', accentClass)} />
        <span className="hidden sm:inline ml-1.5">Columns</span>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-48 p-2">
        <div className="space-y-1">
          {hideableColumns.map((col) => (
            <label
              key={col.id}
              className="flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-muted cursor-pointer text-sm"
            >
              <Checkbox
                checked={visibleColumnIds.includes(col.id)}
                onCheckedChange={() => toggleColumn(col.id)}
              />
              {col.label}
            </label>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}
