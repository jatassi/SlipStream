import {
  ArrowDownCircle,
  ArrowUpCircle,
  ArrowUpDown,
  Binoculars,
  CheckCircle,
  Clock,
  Eye,
  Grid,
  List,
  XCircle,
} from 'lucide-react'

import { ColumnConfigPopover } from '@/components/tables/column-config-popover'
import { FilterDropdown } from '@/components/ui/filter-dropdown'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Slider } from '@/components/ui/slider'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { MOVIE_COLUMNS } from '@/lib/table-columns'
import { cn } from '@/lib/utils'

import type { FilterStatus, SortField } from './use-movie-list'

const FILTER_OPTIONS: { value: FilterStatus; label: string; icon: typeof Eye }[] = [
  { value: 'monitored', label: 'Monitored', icon: Eye },
  { value: 'unreleased', label: 'Unreleased', icon: Clock },
  { value: 'missing', label: 'Missing', icon: Binoculars },
  { value: 'downloading', label: 'Downloading', icon: ArrowDownCircle },
  { value: 'failed', label: 'Failed', icon: XCircle },
  { value: 'upgradable', label: 'Upgradable', icon: ArrowUpCircle },
  { value: 'available', label: 'Available', icon: CheckCircle },
]

const SORT_OPTIONS: { value: SortField; label: string }[] = [
  { value: 'title', label: 'Title' },
  { value: 'monitored', label: 'Monitored' },
  { value: 'qualityProfile', label: 'Quality Profile' },
  { value: 'releaseDate', label: 'Release Date' },
  { value: 'dateAdded', label: 'Date Added' },
  { value: 'rootFolder', label: 'Root Folder' },
  { value: 'sizeOnDisk', label: 'Size on Disk' },
]

type Props = {
  statusFilters: FilterStatus[]
  sortField: SortField
  moviesView: 'grid' | 'table'
  posterSize: number
  movieTableColumns: string[]
  isLoading: boolean
  onToggleFilter: (v: FilterStatus) => void
  onResetFilters: () => void
  onSortFieldChange: (v: string) => void
  onViewChange: (v: string[]) => void
  onPosterSizeChange: (v: number | readonly number[]) => void
  onTableColumnsChange: (cols: string[]) => void
}

export function MovieListFilters(props: Props) {
  const { sortField, moviesView, isLoading } = props

  return (
    <div className="mb-6 flex flex-wrap items-center gap-2">
      <FilterDropdown
        options={FILTER_OPTIONS}
        selected={props.statusFilters}
        onToggle={props.onToggleFilter}
        onReset={props.onResetFilters}
        label="Statuses"
        theme="movie"
        disabled={isLoading}
      />
      <SortSelect sortField={sortField} onChange={props.onSortFieldChange} disabled={isLoading} />
      <div className="ml-auto flex items-center gap-4">
        <ViewOptions {...props} />
        <ToggleGroup value={[moviesView]} onValueChange={props.onViewChange} disabled={isLoading}>
          <ToggleGroupItem value="grid" aria-label="Grid view"><Grid className="size-4" /></ToggleGroupItem>
          <ToggleGroupItem value="table" aria-label="Table view"><List className="size-4" /></ToggleGroupItem>
        </ToggleGroup>
      </div>
    </div>
  )
}

function SortSelect({ sortField, onChange, disabled }: { sortField: SortField; onChange: (v: string) => void; disabled: boolean }) {
  return (
    <Select value={sortField} onValueChange={(v) => v && onChange(v)} disabled={disabled}>
      <SelectTrigger className="gap-1.5">
        <ArrowUpDown className={cn('size-4 shrink-0', sortField === 'title' ? 'text-muted-foreground' : 'text-movie-400')} />
        <span className="hidden sm:inline">{SORT_OPTIONS.find((o) => o.value === sortField)?.label}</span>
      </SelectTrigger>
      <SelectContent>
        {SORT_OPTIONS.map((opt) => (
          <SelectItem key={opt.value} value={opt.value}>{opt.label}</SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

function ViewOptions(props: Props) {
  if (props.moviesView === 'grid') {
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
      columns={MOVIE_COLUMNS}
      visibleColumnIds={props.movieTableColumns}
      onVisibleColumnsChange={props.onTableColumnsChange}
      theme="movie"
    />
  )
}
