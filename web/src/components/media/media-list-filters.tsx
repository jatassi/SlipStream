import type { LucideIcon } from 'lucide-react'
import { ArrowUpDown, Grid, List } from 'lucide-react'

import { ColumnConfigPopover } from '@/components/tables/column-config-popover'
import { FilterDropdown } from '@/components/ui/filter-dropdown'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
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
  theme: 'movie' | 'tv'
}

export function MediaListFilters<T, F extends string, S extends string>(props: Props<T, F, S>) {
  const { sortField, view, isLoading, theme } = props

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
      <SortSelect sortField={sortField} sortOptions={props.sortOptions} onChange={props.onSortFieldChange} disabled={isLoading} theme={theme} />
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

function SortSelect<S extends string>({ sortField, sortOptions, onChange, disabled, theme }: {
  sortField: S
  sortOptions: SortOption<S>[]
  onChange: (v: string) => void
  disabled: boolean
  theme: 'movie' | 'tv'
}) {
  const accentClass = theme === 'movie' ? 'text-movie-400' : 'text-tv-400'

  return (
    <Select value={sortField} onValueChange={(v) => v && onChange(v)} disabled={disabled}>
      <SelectTrigger className="gap-1.5">
        <ArrowUpDown className={cn('size-4 shrink-0', sortField === 'title' ? 'text-muted-foreground' : accentClass)} />
        <span className="hidden sm:inline">{sortOptions.find((o) => o.value === sortField)?.label}</span>
      </SelectTrigger>
      <SelectContent>
        {sortOptions.map((opt) => (
          <SelectItem key={opt.value} value={opt.value}>{opt.label}</SelectItem>
        ))}
      </SelectContent>
    </Select>
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
