import { cn } from '@/lib/utils'
import { SeriesCard } from './SeriesCard'
import type { Series } from '@/types'

interface SeriesGridProps {
  series: Series[]
  className?: string
  editMode?: boolean
  selectedIds?: Set<number>
  onToggleSelect?: (id: number) => void
}

export function SeriesGrid({ series, className, editMode, selectedIds, onToggleSelect }: SeriesGridProps) {
  return (
    <div
      className={cn(
        'grid gap-4 grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6',
        className
      )}
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
