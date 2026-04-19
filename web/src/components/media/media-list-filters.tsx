import type { LucideIcon } from 'lucide-react'
import { ArrowDown, ArrowUp, ChevronDown, Grid, List } from 'lucide-react'

import { ColumnConfigPopover } from '@/components/tables/column-config-popover'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { FilterDropdown } from '@/components/ui/filter-dropdown'
import { Slider } from '@/components/ui/slider'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import type { ColumnDef } from '@/lib/table-columns'
import { cn } from '@/lib/utils'

type FilterOption<F extends string> = { value: F; label: string; icon: LucideIcon }
type SortOption<S extends string> = { value: S; label: string }

type Props<T, F extends string, S extends string> = {
  filterOptions: FilterOption<F>[]
  sortOptions: SortOption<S>[]
  statusFilters: F[]
  sortField: S
  sortDirection: 'asc' | 'desc'
  view: 'grid' | 'table'
  posterSize: number
  visibleColumnIds: string[]
  columns: ColumnDef<T>[]
  isLoading: boolean
  onToggleFilter: (v: F) => void
  onResetFilters: () => void
  onSortFieldChange: (v: string) => void
  onViewChange: (v: string[]) => void
  onPosterSizeChange: (v: number | readonly number[]) => void
  onTableColumnsChange: (cols: string[]) => void
  theme: string
}

export function MediaListFilters<T, F extends string, S extends string>(props: Props<T, F, S>) {
  const { sortField, sortDirection, view, isLoading, theme } = props

  return (
    <div className="mb-6 flex flex-wrap items-center gap-2">
      <FilterDropdown
        options={props.filterOptions}
        selected={props.statusFilters}
        onToggle={props.onToggleFilter}
        onReset={props.onResetFilters}
        label="Statuses"
        theme={theme}
        disabled={isLoading}
      />
      <SortSelect
        sortField={sortField}
        sortDirection={sortDirection}
        sortOptions={props.sortOptions}
        onChange={props.onSortFieldChange}
        disabled={isLoading}
        theme={theme}
      />
      <div className="ml-auto flex items-center gap-4">
        <ViewOptions {...props} />
        <ToggleGroup value={[view]} onValueChange={props.onViewChange} disabled={isLoading}>
          <ToggleGroupItem value="grid" aria-label="Grid view"><Grid className="size-4" /></ToggleGroupItem>
          <ToggleGroupItem value="table" aria-label="Table view"><List className="size-4" /></ToggleGroupItem>
        </ToggleGroup>
      </div>
    </div>
  )
}

function SortSelect<S extends string>({ sortField, sortDirection, sortOptions, onChange, disabled, theme }: {
  sortField: S
  sortDirection: 'asc' | 'desc'
  sortOptions: SortOption<S>[]
  onChange: (v: string) => void
  disabled: boolean
  theme: string
}) {
  const accentMap: Record<string, string> = { movie: 'text-movie-400', tv: 'text-tv-400' }
  const accentClass = accentMap[theme] ?? 'text-primary'
  const DirectionIcon = sortDirection === 'asc' ? ArrowUp : ArrowDown
  const selectedLabel = sortOptions.find((o) => o.value === sortField)?.label

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        disabled={disabled}
        className={cn(
          'border-input dark:bg-input/30 dark:hover:bg-input/50 focus-visible:border-ring focus-visible:ring-ring/50 flex h-8 w-fit items-center gap-1.5 rounded-lg border bg-transparent py-2 pr-2 pl-2.5 text-sm whitespace-nowrap transition-colors outline-none select-none focus-visible:ring-[3px]',
          disabled && 'pointer-events-none opacity-50',
        )}
      >
        <DirectionIcon className={cn('size-4 shrink-0', sortField === 'title' ? 'text-muted-foreground' : accentClass)} />
        <span className="hidden sm:inline">{selectedLabel}</span>
        <ChevronDown className="text-muted-foreground size-4 shrink-0" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-auto min-w-(--anchor-width)">
        {sortOptions.map((opt) => {
          const selected = opt.value === sortField
          return (
            <DropdownMenuItem key={opt.value} onClick={() => onChange(opt.value)}>
              <span className="flex-1 pr-4">{opt.label}</span>
              {selected ? <DirectionIcon className={cn('size-4 shrink-0', accentClass)} /> : null}
            </DropdownMenuItem>
          )
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function ViewOptions<T, F extends string, S extends string>(props: Props<T, F, S>) {
  if (props.view === 'grid') {
    return (
      <div className="flex items-center gap-2">
        <span className="text-muted-foreground text-xs">Size</span>
        <Slider
          value={[props.posterSize]}
          onValueChange={props.onPosterSizeChange}
          min={100}
          max={250}
          step={10}
          className="w-24"
          disabled={props.isLoading}
        />
      </div>
    )
  }
  return (
    <ColumnConfigPopover
      columns={props.columns}
      visibleColumnIds={props.visibleColumnIds}
      onVisibleColumnsChange={props.onTableColumnsChange}
      theme={props.theme}
    />
  )
}
