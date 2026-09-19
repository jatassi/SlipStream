import { useLayoutEffect, useRef, useState } from 'react'

import type { LucideIcon } from 'lucide-react'
import { ArrowDownToLine, LayoutGrid, Search, Sparkles } from 'lucide-react'

import { afterPaint } from '../../shared/after-paint'
import { useMobileState } from '../../shared/mobile-state-context'

export type CineView = 'home' | 'library' | 'activity' | 'search'

const ITEMS: { id: CineView; label: string; icon: LucideIcon }[] = [
  { id: 'home', label: 'Now', icon: Sparkles },
  { id: 'library', label: 'Library', icon: LayoutGrid },
  { id: 'activity', label: 'Activity', icon: ArrowDownToLine },
  { id: 'search', label: 'Search', icon: Search },
]

function useSlidingHighlight(active: number) {
  const navRef = useRef<HTMLElement>(null)
  const highlightRef = useRef<HTMLSpanElement>(null)
  const [ready, setReady] = useState(false)

  useLayoutEffect(() => {
    const items = navRef.current?.querySelectorAll<HTMLButtonElement>('.cine-dock-item')
    const el = items?.[active]
    const highlight = highlightRef.current
    if (el && highlight) {
      highlight.style.width = `${el.offsetWidth}px`
      highlight.style.transform = `translateX(${el.offsetLeft}px)`
    }
  }, [active])

  useLayoutEffect(() => afterPaint(() => setReady(true)), [])

  return { navRef, highlightRef, ready }
}

export function Dock({ view, onChange }: { view: CineView; onChange: (view: CineView) => void }) {
  const activeIndex = ITEMS.findIndex((i) => i.id === view)
  const { navRef, highlightRef, ready } = useSlidingHighlight(activeIndex)
  const { queue } = useMobileState()
  const downloading = queue.filter((q) => q.state === 'downloading').length

  return (
    <nav ref={navRef} className="cine-dock" aria-label="Primary" data-ready={ready ? '' : undefined}>
      <span ref={highlightRef} className="cine-dock-highlight" aria-hidden="true" />
      {ITEMS.map(({ id, label, icon: Icon }) => {
        const active = id === view
        return (
          <button
            key={id}
            type="button"
            className="cine-dock-item m-press"
            data-active={active ? '' : undefined}
            aria-current={active ? 'page' : undefined}
            aria-label={label}
            onClick={() => onChange(id)}
          >
            <Icon className="size-[22px]" strokeWidth={active ? 2.4 : 2} />
            {active ? <span className="ml-2 text-m-footnote font-semibold">{label}</span> : null}
            {id === 'activity' && !active && downloading > 0 ? (
              <span className="absolute top-2.5 right-2.5 size-2 rounded-full bg-movie-500 shadow-[0_0_8px_var(--movie-500)]" />
            ) : null}
          </button>
        )
      })}
    </nav>
  )
}
