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
  const width = globalThis.window.innerWidth
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

const ICON_MAP = {
  movie: Film,
  series: Tv,
} as const

const THEME_COLOR_MAP = {
  movie: 'text-movie-400',
  series: 'text-tv-400',
} as const

const ICON_GLOW_MAP = {
  movie: 'icon-glow-movie',
  series: 'icon-glow-tv',
} as const

function getThemeColor(icon?: 'movie' | 'series'): string {
  if (!icon) {
    return ''
  }
  return THEME_COLOR_MAP[icon]
}

function getIconGlow(icon?: 'movie' | 'series'): string {
  if (!icon) {
    return ''
  }
  return ICON_GLOW_MAP[icon]
}

type HeaderProps = {
  label: string
  count: number
  icon?: 'movie' | 'series'
  collapsible: boolean
  hasMore: boolean
  expanded: boolean
  onToggle: () => void
}

function GridHeader({
  label,
  count,
  icon,
  collapsible,
  hasMore,
  expanded,
  onToggle,
}: HeaderProps) {
  const themeColor = getThemeColor(icon)
  const colorClass = themeColor || 'text-muted-foreground'

  if (collapsible && hasMore) {
    return (
      <button
        type="button"
        className={`flex items-center gap-2 text-sm ${colorClass} cursor-pointer hover:brightness-125 transition-all duration-200`}
        onClick={onToggle}
      >
        <ChevronRight
          className={`size-4 transition-transform duration-200 ${expanded ? 'rotate-90' : ''}`}
        />
        <span>
          {label} ({count})
        </span>
      </button>
    )
  }

  const Icon = icon ? ICON_MAP[icon] : null

  return (
    <div
      className={`flex items-center gap-2 text-sm ${colorClass} transition-all duration-200`}
    >
      {Icon ? <Icon className={`size-4 ${getIconGlow(icon)}`} /> : null}
      <span>
        {label} ({count})
      </span>
    </div>
  )
}

function ShowMoreCard({
  remainingCount,
  onExpand,
}: {
  remainingCount: number
  onExpand: () => void
}) {
  return (
    <button
      type="button"
      className="border-border bg-card/50 hover:bg-card/80 cursor-pointer rounded-lg border-2 border-dashed transition-colors w-full"
      onClick={onExpand}
    >
      <div className="flex aspect-[2/3] items-center justify-center">
        <div className="p-4 text-center">
          <ChevronDown className="text-muted-foreground mx-auto mb-2 size-6" />
          <span className="text-muted-foreground text-sm">
            Show {remainingCount} more...
          </span>
        </div>
      </div>
      <div className="p-2">
        <div className="h-8" />
      </div>
    </button>
  )
}

function ShowLessButton({ onCollapse }: { onCollapse: () => void }) {
  return (
    <button
      type="button"
      className="hover:text-foreground flex cursor-pointer justify-center pt-2 transition-all duration-200 w-full"
      onClick={onCollapse}
    >
      <div className="text-muted-foreground flex items-center gap-1 text-sm">
        <ChevronDown className="size-4 rotate-180" />
        <span>Show less</span>
      </div>
    </button>
  )
}

const GRID_CLASS = 'grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-7 xl:grid-cols-8'

type GridBodyProps<T> = {
  visibleItems: T[]
  renderItem: (item: T, index: number) => ReactNode
  getKey: (item: T) => string | number
  expanded: boolean
  hasMore: boolean
  collapsible: boolean
  remainingCount: number
  onExpand: () => void
  onCollapse: () => void
}

function GridBody<T>({
  visibleItems, renderItem, getKey,
  expanded, hasMore, collapsible, remainingCount,
  onExpand, onCollapse,
}: GridBodyProps<T>) {
  return (
    <>
      <div className={GRID_CLASS}>
        {visibleItems.map((item, index) => (
          <div key={getKey(item)}>{renderItem(item, index)}</div>
        ))}
        {!expanded && hasMore ? (
          <ShowMoreCard remainingCount={remainingCount} onExpand={onExpand} />
        ) : null}
      </div>
      {expanded && hasMore && collapsible ? (
        <ShowLessButton onCollapse={onCollapse} />
      ) : null}
    </>
  )
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
  items, renderItem, getKey, label, icon,
  showHeader = true, collapsible = true,
}: ExpandableMediaGridProps<T>) {
  const [expanded, setExpanded] = useState(false)
  const [prevItemsLength, setPrevItemsLength] = useState(items.length)
  const columns = useGridColumns()
  const initialCount = items.length <= columns ? items.length : columns - 1
  if (items.length !== prevItemsLength) {
    if (Math.abs(prevItemsLength - items.length) > 2) {
      setExpanded(false)
    }
    setPrevItemsLength(items.length)
  }
  if (items.length === 0) {
    return null
  }

  const hasMore = items.length > initialCount
  const visibleItems = expanded ? items : items.slice(0, initialCount)
  return (
    <div className="space-y-3">
      {showHeader ? (
        <GridHeader
          label={label}
          count={items.length}
          icon={icon}
          collapsible={collapsible}
          hasMore={hasMore}
          expanded={expanded}
          onToggle={() => setExpanded(!expanded)}
        />
      ) : null}
      <GridBody
        visibleItems={visibleItems}
        renderItem={renderItem}
        getKey={getKey}
        expanded={expanded}
        hasMore={hasMore}
        collapsible={collapsible}
        remainingCount={items.length - initialCount}
        onExpand={() => setExpanded(true)}
        onCollapse={() => setExpanded(false)}
      />
    </div>
  )
}
