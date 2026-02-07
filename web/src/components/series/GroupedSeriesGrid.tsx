import { SeriesGrid } from './SeriesGrid'
import type { MediaGroup } from '@/lib/grouping'
import type { Series } from '@/types'

interface GroupedSeriesGridProps {
  groups: MediaGroup<Series>[]
  posterSize?: number
  editMode?: boolean
  selectedIds?: Set<number>
  onToggleSelect?: (id: number) => void
}

export function GroupedSeriesGrid({ groups, posterSize, editMode, selectedIds, onToggleSelect }: GroupedSeriesGridProps) {
  return (
    <div className="space-y-0">
      {groups.map((group) => (
        <div key={group.key}>
          <div className="sticky top-0 z-10 backdrop-blur-md bg-background/80 border-b border-border/50 px-1 py-2 mb-4 flex items-center gap-2">
            <span className="text-sm font-medium">{group.label}</span>
            <span className="text-xs text-muted-foreground">({group.items.length})</span>
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
