import { SeriesCard } from './SeriesCard'
import type { Series } from '@/types'

interface SeriesGridProps {
  series: Series[]
  posterSize?: number
  editMode?: boolean
  selectedIds?: Set<number>
  onToggleSelect?: (id: number) => void
}

export function SeriesGrid({ series, posterSize = 150, editMode, selectedIds, onToggleSelect }: SeriesGridProps) {
  return (
    <div
      className="grid gap-4"
      style={{
        gridTemplateColumns: `repeat(auto-fill, minmax(${posterSize}px, 1fr))`,
      }}
    >
      {series.map((s) => (
        <SeriesCard
          key={s.id}
          series={s}
          editMode={editMode}
          selected={selectedIds?.has(s.id)}
          onToggleSelect={onToggleSelect}
        />
      ))}
    </div>
  )
}
