import type { ReactNode } from 'react'

type MediaGridProps<T extends { id: number }> = {
  items: T[]
  renderCard: (item: T, opts: { editMode?: boolean; selected?: boolean; onToggleSelect?: (id: number) => void }) => ReactNode
  posterSize?: number
  editMode?: boolean
  selectedIds?: Set<number>
  onToggleSelect?: (id: number) => void
}

export function MediaGrid<T extends { id: number }>({
  items,
  renderCard,
  posterSize = 150,
  editMode,
  selectedIds,
  onToggleSelect,
}: MediaGridProps<T>) {
  return (
    <div className="grid gap-4" style={{ gridTemplateColumns: `repeat(auto-fill, minmax(${posterSize}px, 1fr))` }}>
      {items.map((item) => renderCard(item, { editMode, selected: selectedIds?.has(item.id), onToggleSelect }))}
    </div>
  )
}
