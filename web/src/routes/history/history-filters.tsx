import { FilterDropdown } from '@/components/ui/filter-dropdown'
import { Segmented, type SegmentedOption } from '@/components/ui/segmented'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { filterableEventTypes } from '@/lib/history-utils'
import type { HistoryEventType } from '@/types'

import type { DatePreset, MediaFilter } from './history-utils'
import { DATE_PRESETS } from './history-utils'

const MEDIA_OPTIONS: SegmentedOption<MediaFilter>[] = [
  { value: 'all', label: 'All' },
  { value: 'movie', label: 'Movies' },
  { value: 'episode', label: 'Series' },
]

type HistoryFiltersProps = {
  mediaType: MediaFilter
  datePreset: DatePreset
  eventTypes: HistoryEventType[]
  onMediaTypeChange: (value: MediaFilter) => void
  onDatePresetChange: (value: string | null) => void
  onToggleEventType: (value: HistoryEventType) => void
  onResetEventTypes: () => void
}

export function HistoryFilters({
  mediaType,
  datePreset,
  eventTypes,
  onMediaTypeChange,
  onDatePresetChange,
  onToggleEventType,
  onResetEventTypes,
}: HistoryFiltersProps) {
  return (
    <div className="px-screen space-y-3 pb-5">
      <Segmented
        label="History media type"
        value={mediaType}
        onChange={onMediaTypeChange}
        options={MEDIA_OPTIONS}
      />
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
