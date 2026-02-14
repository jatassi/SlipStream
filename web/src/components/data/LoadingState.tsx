import { useLayoutEffect, useRef, useState } from 'react'

import { Skeleton } from '@/components/ui/skeleton'

type LoadingStateProps = {
  variant?: 'card' | 'list' | 'detail'
  count?: number
  posterSize?: number
  theme?: 'movie' | 'tv'
}

const GAP = 16

function useFillCount(
  ref: React.RefObject<HTMLDivElement | null>,
  posterSize: number | undefined,
  variant: string,
) {
  const [count, setCount] = useState(0)

  useLayoutEffect(() => {
    const el = ref.current
    if (!el) {
      return
    }

    function calc() {
      if (!el) {
        return
      }
      const rect = el.getBoundingClientRect()
      const availableWidth = rect.width
      const availableHeight = window.innerHeight - rect.top

      if (variant === 'list') {
        const rowHeight = 48 + 8 // h-12 + space-y-2
        setCount(Math.ceil(availableHeight / rowHeight))
        return
      }

      const minWidth = posterSize || 150
      const cols = Math.max(1, Math.floor((availableWidth + GAP) / (minWidth + GAP)))
      const colWidth = (availableWidth - GAP * (cols - 1)) / cols
      const cardHeight = colWidth * 1.5 // aspect-[2/3]
      const rows = Math.max(1, Math.ceil(availableHeight / (cardHeight + GAP)))
      setCount(cols * rows)
    }

    calc()
    const observer = new ResizeObserver(calc)
    observer.observe(el)
    return () => observer.disconnect()
  }, [ref, posterSize, variant])

  return count
}

export function LoadingState({ variant = 'card', count, posterSize, theme }: LoadingStateProps) {
  const gridRef = useRef<HTMLDivElement>(null)
  const autoCount = useFillCount(gridRef, posterSize, variant)
  const finalCount = count ?? autoCount

  if (variant === 'list') {
    return (
      <div ref={gridRef} className="space-y-2">
        {finalCount > 0 &&
          Array.from({ length: finalCount }, (_, i) => i).map((i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
      </div>
    )
  }

  if (variant === 'detail') {
    return (
      <div className="space-y-4">
        <Skeleton className="h-48 w-full" />
        <Skeleton className="h-8 w-3/4" />
        <Skeleton className="h-4 w-1/2" />
        <Skeleton className="h-4 w-2/3" />
      </div>
    )
  }

  const gridStyle = posterSize
    ? { gridTemplateColumns: `repeat(auto-fill, minmax(${posterSize}px, 1fr))` }
    : undefined

  const gridClassName = posterSize
    ? 'grid gap-4'
    : 'grid gap-4 grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6'

  return (
    <div ref={gridRef} className={gridClassName} style={gridStyle}>
      {finalCount > 0 &&
        Array.from({ length: finalCount }, (_, i) => i).map((i) => (
          <div key={i} className="border-border bg-card overflow-hidden rounded-lg border">
            <div className="relative aspect-[2/3]">
              <Skeleton className="absolute inset-0 rounded-none" />
              <div className="absolute top-2 right-2 z-10">
                <Skeleton className="h-5 w-16 rounded-full" />
              </div>
              {theme === 'tv' && (
                <div className="absolute top-2 left-2 z-10">
                  <Skeleton className="h-5 w-10 rounded-full" />
                </div>
              )}
              <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/80 via-black/40 to-transparent p-3 pt-8">
                <Skeleton className="h-4 w-3/4 bg-white/10" />
                <Skeleton className="mt-1.5 h-3 w-1/3 bg-white/10" />
              </div>
            </div>
          </div>
        ))}
    </div>
  )
}
