import type { MediaGroup } from '@/lib/grouping'
import type { Series } from '@/types'

import { SeriesGrid } from './SeriesGrid'

type GroupedSeriesGridProps = {
  groups: MediaGroup<Series>[]
  posterSize?: number
  editMode?: boolean
  selectedIds?: Set<number>
  onToggleSelect?: (id: number) => void
}

export function GroupedSeriesGrid({
  groups,
  posterSize,
  editMode,
  selectedIds,
  onToggleSelect,
}: GroupedSeriesGridProps) {
  return (
    <div className="space-y-0">
      {groups.map((group) => (
        <div key={group.key}>
          <div className="bg-background/80 border-border/50 sticky top-0 z-10 mb-4 flex items-center gap-2 border-b px-1 py-2 backdrop-blur-md">
            <span className="text-sm font-medium">{group.label}</span>
            <span className="text-muted-foreground text-xs">({group.items.length})</span>
          </div>
          <div className="mb-6">
            <SeriesGrid
              series={group.items}
              posterSize={posterSize}
              editMode={editMode}
              selectedIds={selectedIds}
              onToggleSelect={onToggleSelect}
            />
          </div>
        </div>
      ))}
    </div>
  )
}
