import type { ReactNode } from 'react'

import type { MediaGroup } from '@/lib/grouping'

type GroupedMediaGridProps<T> = {
  groups: MediaGroup<T>[]
  renderGrid: (items: T[]) => ReactNode
}

export function GroupedMediaGrid<T>({ groups, renderGrid }: GroupedMediaGridProps<T>) {
  return (
    <div className="space-y-0">
      {groups.map((group) => (
        <div key={group.key}>
          <div className="bg-background/80 border-border/50 sticky top-0 z-10 mb-4 flex items-center gap-2 border-b px-1 py-2 backdrop-blur-md">
            <span className="text-sm font-medium">{group.label}</span>
            <span className="text-muted-foreground text-xs">({group.items.length})</span>
          </div>
          <div className="mb-6">{renderGrid(group.items)}</div>
        </div>
      ))}
    </div>
  )
}
