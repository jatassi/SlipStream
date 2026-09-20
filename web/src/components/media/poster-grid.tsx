import type { ReactNode } from 'react'

import { useViewport } from '@/hooks/use-viewport'

type PosterGridProps = {
  label: string
  posterSize: number
  children: ReactNode
}

export function PosterGrid({ label, posterSize, children }: PosterGridProps) {
  const shell = useViewport()

  if (shell === 'phone') {
    return (
      <div role="list" aria-label={label} className="px-screen grid grid-cols-3 gap-x-3 gap-y-5">
        {children}
      </div>
    )
  }

  return (
    <div
      role="list"
      aria-label={label}
      className="px-screen grid gap-x-4 gap-y-6"
      style={{ gridTemplateColumns: `repeat(auto-fill, minmax(${posterSize}px, 1fr))` }}
    >
      {children}
    </div>
  )
}

export function PosterGridItem({ children }: { children: ReactNode }) {
  return <div role="listitem">{children}</div>
}
