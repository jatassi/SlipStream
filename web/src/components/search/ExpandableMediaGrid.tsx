import { useState, useRef, useEffect, useSyncExternalStore, type ReactNode } from 'react'
import { ChevronRight, ChevronDown, Film, Tv } from 'lucide-react'

const BREAKPOINTS = [
  { minWidth: 1280, columns: 8 }, // xl
  { minWidth: 1024, columns: 7 }, // lg
  { minWidth: 768, columns: 5 },  // md
  { minWidth: 640, columns: 4 },  // sm
  { minWidth: 0, columns: 3 },    // default
]

function getColumns(): number {
  if (typeof window === 'undefined') return 3
  const width = window.innerWidth
  for (const bp of BREAKPOINTS) {
    if (width >= bp.minWidth) return bp.columns
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

interface ExpandableMediaGridProps<T> {
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
  const prevItemsLengthRef = useRef(items.length)
  const columns = useGridColumns()
  // Show all items if they fit in one row, otherwise leave room for "Show more" card
  const initialCount = items.length <= columns ? items.length : columns - 1

  // Reset expansion when items change significantly
  useEffect(() => {
    if (Math.abs(prevItemsLengthRef.current - items.length) > 2) {
      setExpanded(false)
    }
    prevItemsLengthRef.current = items.length
  }, [items.length])

  if (items.length === 0) return null

  const visibleItems = expanded ? items : items.slice(0, initialCount)
  const hasMore = items.length > initialCount
  const Icon = icon === 'movie' ? Film : icon === 'series' ? Tv : null

  return (
    <div className="space-y-3">
      {showHeader && (
        <div
          className={`flex items-center gap-2 text-sm text-muted-foreground ${
            collapsible && hasMore ? 'cursor-pointer hover:text-foreground' : ''
          } transition-all duration-200`}
          onClick={collapsible && hasMore ? () => setExpanded(!expanded) : undefined}
        >
          {collapsible && hasMore ? (
            <ChevronRight
              className={`size-4 transition-transform duration-200 ${expanded ? 'rotate-90' : ''}`}
            />
          ) : Icon ? (
            <Icon className="size-4" />
          ) : null}
          <span>
            {label} ({items.length})
          </span>
        </div>
      )}

      <div className="grid gap-3 grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-7 xl:grid-cols-8">
        {visibleItems.map((item, index) => (
          <div key={getKey(item)}>{renderItem(item, index)}</div>
        ))}

        {/* Show more card */}
        {!expanded && hasMore && (
          <div
            className="rounded-lg border-2 border-dashed border-border bg-card/50 cursor-pointer hover:bg-card/80 transition-colors"
            onClick={() => setExpanded(true)}
          >
            <div className="aspect-[2/3] flex items-center justify-center">
              <div className="text-center p-4">
                <ChevronDown className="size-6 mx-auto mb-2 text-muted-foreground" />
                <span className="text-sm text-muted-foreground">
                  Show {items.length - initialCount} more...
                </span>
              </div>
            </div>
            <div className="p-2">
              <div className="h-8" />
            </div>
          </div>
        )}
      </div>

      {/* Show less button */}
      {expanded && hasMore && collapsible && (
        <div
          className="flex justify-center pt-2 cursor-pointer hover:text-foreground transition-all duration-200"
          onClick={() => setExpanded(false)}
        >
          <div className="flex items-center gap-1 text-sm text-muted-foreground">
            <ChevronDown className="size-4 rotate-180" />
            <span>Show less</span>
          </div>
        </div>
      )}
    </div>
  )
}
