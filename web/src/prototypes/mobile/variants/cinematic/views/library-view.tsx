import { useState } from 'react'

import { useMobileState } from '../../../shared/mobile-state-context'
import { Poster } from '../../../shared/poster'
import type { MediaItem } from '../../../shared/types'
import { Chip, StatusPill } from '../../../shared/ui'

type Filter = 'all' | 'movie' | 'series' | 'missing' | 'downloading'

const FILTERS: { id: Filter; label: string }[] = [
  { id: 'all', label: 'All' },
  { id: 'movie', label: 'Movies' },
  { id: 'series', label: 'Series' },
  { id: 'missing', label: 'Missing' },
  { id: 'downloading', label: 'Downloading' },
]

function applyFilter(items: MediaItem[], filter: Filter): MediaItem[] {
  if (filter === 'all') {
    return items
  }
  if (filter === 'movie' || filter === 'series') {
    return items.filter((m) => m.kind === filter)
  }
  return items.filter((m) => m.status === filter)
}

export function LibraryView({ onOpen }: { onOpen: (id: number) => void }) {
  const [filter, setFilter] = useState<Filter>('all')
  const { library } = useMobileState()
  const items = applyFilter(library, filter)

  return (
    <div className="relative h-full">
      <div className="m-material absolute inset-x-0 top-0 z-10 m-safe-top">
        <div className="flex items-baseline justify-between px-screen pt-2 pb-3">
          <h1 className="text-m-heading font-extrabold tracking-[-0.03em]">Library</h1>
          <span className="m-nums text-m-footnote text-muted-foreground">{items.length} titles</span>
        </div>
        <div className="m-scroll-x flex gap-2 px-screen pb-3 [scroll-snap-type:none]">
          {FILTERS.map((f) => (
            <Chip key={f.id} active={filter === f.id} onClick={() => setFilter(f.id)}>{f.label}</Chip>
          ))}
        </div>
      </div>
      <div className="m-scroll h-full pb-32" style={{ paddingTop: 'calc(var(--safe-top) + 104px)' }}>
        <div key={filter} className="m-stagger grid grid-cols-2 gap-3 px-3 [&>button]:m-enter-scale">
          {items.map((item) => (
            <button key={item.id} type="button" onClick={() => onOpen(item.id)} className="m-press relative text-left">
              <Poster item={item} radius="rounded-[14px]" className="shadow-[0_12px_30px_rgba(0,0,0,0.45)]" />
              {item.status !== 'available' && <StatusPill status={item.status} className="m-material-chip absolute top-2.5 left-2.5" />}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}
