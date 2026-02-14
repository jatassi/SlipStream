import type { MediaGroup } from '@/lib/grouping'
import type { Movie } from '@/types'

import { MovieGrid } from './MovieGrid'

type GroupedMovieGridProps = {
  groups: MediaGroup<Movie>[]
  posterSize?: number
  editMode?: boolean
  selectedIds?: Set<number>
  onToggleSelect?: (id: number) => void
}

export function GroupedMovieGrid({
  groups,
  posterSize,
  editMode,
  selectedIds,
  onToggleSelect,
}: GroupedMovieGridProps) {
  return (
    <div className="space-y-0">
      {groups.map((group) => (
        <div key={group.key}>
          <div className="bg-background/80 border-border/50 sticky top-0 z-10 mb-4 flex items-center gap-2 border-b px-1 py-2 backdrop-blur-md">
            <span className="text-sm font-medium">{group.label}</span>
            <span className="text-muted-foreground text-xs">({group.items.length})</span>
          </div>
          <div className="mb-6">
            <MovieGrid
              movies={group.items}
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
