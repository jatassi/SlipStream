import { useState } from 'react'

import { Search, X } from 'lucide-react'

import { KIND_LABEL, KIND_TEXT } from '../../../shared/format'
import { useMobileState } from '../../../shared/mobile-state-context'
import { Poster } from '../../../shared/poster'
import type { MediaItem } from '../../../shared/types'
import { StatusDot } from '../../../shared/ui'

function ResultList({ items, onOpen }: { items: MediaItem[]; onOpen: (id: number) => void }) {
  return (
    <div className="m-stagger space-y-1 [&>button]:m-enter-fade-up">
      {items.map((item) => (
        <button key={item.id} type="button" onClick={() => onOpen(item.id)} className="m-press-row flex w-full items-center gap-3 rounded-[12px] px-2 py-2 text-left">
          <Poster item={item} showTitle={false} className="w-10" radius="rounded-[5px]" />
          <div className="min-w-0 flex-1">
            <div className="truncate text-m-body font-semibold">{item.title}</div>
            <div className="flex items-center gap-1.5 text-m-caption text-muted-foreground">
              <StatusDot status={item.status} />
              <span className="m-nums">{item.year}</span>
              <span>·</span>
              <span className={KIND_TEXT[item.kind]}>{KIND_LABEL[item.kind]}</span>
            </div>
          </div>
        </button>
      ))}
    </div>
  )
}

export function SearchOverlay({ onClose, onOpen }: { onClose: () => void; onOpen: (id: number) => void }) {
  const [query, setQuery] = useState('')
  const [closing, setClosing] = useState(false)
  const { library } = useMobileState()
  const q = query.trim().toLowerCase()
  const results = q.length === 0 ? library.slice(0, 6) : library.filter((m) => m.title.toLowerCase().includes(q))

  return (
    <div
      className="cine-overlay m-material-heavy"
      data-closing={closing ? '' : undefined}
      onAnimationEnd={(e) => {
        if (closing && e.target === e.currentTarget) {
          onClose()
        }
      }}
    >
      <div className="flex items-center gap-2 px-screen pb-3" style={{ paddingTop: 'calc(var(--safe-top) + 8px)' }}>
        <label className="flex h-11 flex-1 items-center gap-2 rounded-full bg-foreground/10 px-4">
          <Search className="size-4 text-muted-foreground" />
          <input
            // eslint-disable-next-line jsx-a11y/no-autofocus -- the overlay exists to be typed into
            autoFocus
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Titles, releases, people"
            enterKeyHint="search"
            autoCapitalize="none"
            className="min-w-0 flex-1 bg-transparent text-[16px] outline-none placeholder:text-muted-foreground"
          />
        </label>
        <button type="button" aria-label="Close" onClick={() => setClosing(true)} className="m-press flex size-11 items-center justify-center rounded-full bg-foreground/10">
          <X className="size-5" />
        </button>
      </div>
      <div className="m-scroll h-full px-screen pb-32">
        <div className="pb-2 text-m-caption font-semibold tracking-[0.1em] text-muted-foreground uppercase">{q.length === 0 ? 'Suggested' : `${results.length} results`}</div>
        <ResultList items={results} onOpen={onOpen} />
      </div>
    </div>
  )
}
