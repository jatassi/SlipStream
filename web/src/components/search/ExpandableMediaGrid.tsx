import { type ReactNode, useState, useSyncExternalStore } from 'react'

import { ChevronDown, ChevronRight, Film, Tv } from 'lucide-react'

const BREAKPOINTS = [
  { minWidth: 1280, columns: 8 }, // xl
  { minWidth: 1024, columns: 7 }, // lg
  { minWidth: 768, columns: 5 }, // md
  { minWidth: 640, columns: 4 }, // sm
  { minWidth: 0, columns: 3 }, // default
]

function getColumns(): number {
  if (typeof window === 'undefined') {
    return 3
  }
  const width = window.innerWidth
  for (const bp of BREAKPOINTS) {
    if (width >= bp.minWidth) {
      return bp.columns
    }
  }
  return 3
}

function subscribe(callback: () => void): () => void {
  window.addEventListener('resize', callback)
  return () => window.removeEventListener('resize', callback)
}

function useGridColumns(): number {
  return useSyncExternalStore(subscribe, getColumns, () => 3)
}

type ExpandableMediaGridProps<T> = {
  items: T[]
  renderItem: (item: T, index: number) => ReactNode
  getKey: (item: T) => string | number
  label: string
  icon?: 'movie' | 'series'
  showHeader?: boolean
  collapsible?: boolean
}

export function ExpandableMediaGrid<T>({
  items,
  renderItem,
  getKey,
  label,
  icon,
  showHeader = true,
  collapsible = true,
}: ExpandableMediaGridProps<T>) {
  const [expanded, setExpanded] = useState(false)
  const [prevItemsLength, setPrevItemsLength] = useState(items.length)
  const columns = useGridColumns()
  // Show all items if they fit in one row, otherwise leave room for "Show more" card
  const initialCount = items.length <= columns ? items.length : columns - 1

  // Reset expansion when items change significantly (React-recommended pattern)
  if (Math.abs(prevItemsLength - items.length) > 2) {
    setExpanded(false)
  }
  if (items.length !== prevItemsLength) {
    setPrevItemsLength(items.length)
  }

  if (items.length === 0) {
    return null
  }

  const visibleItems = expanded ? items : items.slice(0, initialCount)
  const hasMore = items.length > initialCount
  const Icon = icon === 'movie' ? Film : icon === 'series' ? Tv : null
  const themeColor = icon === 'movie' ? 'text-movie-400' : icon === 'series' ? 'text-tv-400' : ''

  return (
    <div className="space-y-3">
      {showHeader ? (
        collapsible && hasMore ? (
          <button
            type="button"
            className={`flex items-center gap-2 text-sm ${themeColor || 'text-muted-foreground'} cursor-pointer hover:brightness-125 transition-all duration-200`}
            onClick={() => setExpanded(!expanded)}
          >
            <ChevronRight
              className={`size-4 transition-transform duration-200 ${expanded ? 'rotate-90' : ''}`}
            />
            <span>
              {label} ({items.length})
            </span>
          </button>
        ) : (
          <div className={`flex items-center gap-2 text-sm ${themeColor || 'text-muted-foreground'} transition-all duration-200`}>
            {Icon ? (
              <Icon
                className={`size-4 ${icon === 'movie' ? 'icon-glow-movie' : icon === 'series' ? 'icon-glow-tv' : ''}`}
              />
            ) : null}
            <span>
              {label} ({items.length})
            </span>
          </div>
        )
      ) : null}

      <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-7 xl:grid-cols-8">
        {visibleItems.map((item, index) => (
          <div key={getKey(item)}>{renderItem(item, index)}</div>
        ))}

        {/* Show more card */}
        {!expanded && hasMore ? (
          <button
            type="button"
            className="border-border bg-card/50 hover:bg-card/80 cursor-pointer rounded-lg border-2 border-dashed transition-colors w-full"
            onClick={() => setExpanded(true)}
          >
            <div className="flex aspect-[2/3] items-center justify-center">
              <div className="p-4 text-center">
                <ChevronDown className="text-muted-foreground mx-auto mb-2 size-6" />
                <span className="text-muted-foreground text-sm">
                  Show {items.length - initialCount} more...
                </span>
              </div>
            </div>
            <div className="p-2">
              <div className="h-8" />
            </div>
          </button>
        ) : null}
      </div>

      {/* Show less button */}
      {expanded && hasMore && collapsible ? (
        <button
          type="button"
          className="hover:text-foreground flex cursor-pointer justify-center pt-2 transition-all duration-200 w-full"
          onClick={() => setExpanded(false)}
        >
          <div className="text-muted-foreground flex items-center gap-1 text-sm">
            <ChevronDown className="size-4 rotate-180" />
            <span>Show less</span>
          </div>
        </button>
      ) : null}
    </div>
  )
}
