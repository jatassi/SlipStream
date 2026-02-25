import { Film, Trash2, Tv } from 'lucide-react'

import { ConfirmDialog } from '@/components/forms/confirm-dialog'
import { Button } from '@/components/ui/button'
import { FilterDropdown } from '@/components/ui/filter-dropdown'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { filterableEventTypes } from '@/lib/history-utils'
import { cn } from '@/lib/utils'
import type { HistoryEventType } from '@/types'

import type { DatePreset, MediaFilter } from './history-utils'
import { DATE_PRESETS } from './history-utils'

export function ClearHistoryAction({
  isLoading,
  onConfirm,
}: {
  isLoading: boolean
  onConfirm: () => Promise<void>
}) {
  if (isLoading) {
    return (
      <Button variant="destructive" disabled>
        <Trash2 className="mr-2 size-4" />
        Clear History
      </Button>
    )
  }

  return (
    <ConfirmDialog
      trigger={
        <Button variant="destructive">
          <Trash2 className="mr-2 size-4" />
          Clear History
        </Button>
      }
      title="Clear history"
      description="Are you sure you want to clear all history? This action cannot be undone."
      confirmLabel="Clear"
      variant="destructive"
      onConfirm={onConfirm}
    />
  )
}

const TAB_ACTIVE = 'data-active:bg-white data-active:text-black'

function MediaTypeTabs({
  value,
  onChange,
}: {
  value: MediaFilter
  onChange: (v: string) => void
}) {
  return (
    <Tabs value={value} onValueChange={onChange}>
      <TabsList>
        <TabsTrigger value="all" className={`data-active:glow-media-sm px-4 ${TAB_ACTIVE}`}>
          All
        </TabsTrigger>
        <TabsTrigger value="movie" className={`data-active:glow-movie ${TAB_ACTIVE}`}>
          <Film className="mr-1.5 size-4" />
          Movies
        </TabsTrigger>
        <TabsTrigger value="episode" className={`data-active:glow-tv ${TAB_ACTIVE}`}>
          <Tv className="mr-1.5 size-4" />
          Series
        </TabsTrigger>
      </TabsList>
    </Tabs>
  )
}

type HistoryFiltersProps = {
  mediaType: MediaFilter
  datePreset: DatePreset
  eventTypes: HistoryEventType[]
  isLoading: boolean
  onMediaTypeChange: (v: string) => void
  onDatePresetChange: (v: string | null) => void
  onToggleEventType: (value: HistoryEventType) => void
  onResetEventTypes: () => void
}

export function HistoryFilters({
  mediaType,
  datePreset,
  eventTypes,
  isLoading,
  onMediaTypeChange,
  onDatePresetChange,
  onToggleEventType,
  onResetEventTypes,
}: HistoryFiltersProps) {
  return (
    <div
      className={cn(
        'mb-4 flex items-center justify-between',
        isLoading && 'pointer-events-none opacity-50',
      )}
    >
      <MediaTypeTabs value={mediaType} onChange={onMediaTypeChange} />

      <div className="flex items-center gap-3">
        <FilterDropdown
          options={filterableEventTypes}
          selected={eventTypes}
          onToggle={onToggleEventType}
          onReset={onResetEventTypes}
          label="Events"
        />

        <Select value={datePreset} onValueChange={onDatePresetChange}>
          <SelectTrigger className="w-auto">
            {DATE_PRESETS.find((p) => p.value === datePreset)?.label}
          </SelectTrigger>
          <SelectContent>
            {DATE_PRESETS.map((preset) => (
              <SelectItem key={preset.value} value={preset.value}>
                {preset.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}
