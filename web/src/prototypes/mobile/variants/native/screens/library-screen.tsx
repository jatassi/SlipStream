import { useState } from 'react'

import { Plus } from 'lucide-react'

import { cn } from '@/lib/utils'

import { useMobileState } from '../../../shared/mobile-state-context'
import { Poster } from '../../../shared/poster'
import type { MediaItem, MediaKind } from '../../../shared/types'
import { StatusDot } from '../../../shared/ui'
import { NativeScreen } from '../native-screen'
import type { NativeNav } from '../use-native-nav'

function Segmented({ value, onChange }: { value: MediaKind; onChange: (kind: MediaKind) => void }) {
  return (
    <div className="relative grid h-8 grid-cols-2 rounded-[9px] bg-foreground/8 p-0.5" role="tablist">
      <span
        className="native-segment-thumb absolute top-0.5 bottom-0.5 left-0.5 w-[calc(50%-2px)] rounded-[7px] bg-card shadow-[0_1px_3px_rgba(0,0,0,0.4),0_0_0_0.5px_rgba(255,255,255,0.06)]"
        style={{ transform: value === 'series' ? 'translateX(100%)' : 'none' }}
        aria-hidden="true"
      />
      {(['movie', 'series'] as MediaKind[]).map((kind) => (
        <button
          key={kind}
          type="button"
          role="tab"
          aria-selected={value === kind}
          onClick={() => onChange(kind)}
          className={cn('relative z-10 text-m-footnote font-semibold transition-colors duration-150', value === kind ? 'text-foreground' : 'text-muted-foreground')}
        >
          {kind === 'movie' ? 'Movies' : 'Series'}
        </button>
      ))}
    </div>
  )
}

function PosterCell({ item, onOpen }: { item: MediaItem; onOpen: () => void }) {
  return (
    <button type="button" onClick={onOpen} className="m-press text-left">
      <Poster item={item} showTitle={false} className="shadow-[0_6px_16px_rgba(0,0,0,0.35)]" />
      <div className="mt-2 truncate text-m-footnote font-semibold">{item.title}</div>
      <div className="mt-0.5 flex items-center gap-1.5 text-m-caption text-muted-foreground">
        <StatusDot status={item.status} />
        <span className="m-nums">{item.year}</span>
        <span>·</span>
        <span className="truncate">{item.quality}</span>
      </div>
    </button>
  )
}

export function LibraryScreen({ nav }: { nav: NativeNav }) {
  const [kind, setKind] = useState<MediaKind>('movie')
  const { library } = useMobileState()
  const items = library.filter((m) => m.kind === kind)

  return (
    <NativeScreen
      title="Library"
      right={
        <button type="button" aria-label="Add" className="m-press flex size-tap items-center justify-center text-tv-400">
          <Plus className="size-6" />
        </button>
      }
    >
      <div className="px-screen pb-4">
        <Segmented value={kind} onChange={setKind} />
      </div>
      <div key={kind} className="m-stagger grid grid-cols-3 gap-x-3 gap-y-5 px-screen [&>button]:m-enter-fade-up">
        {items.map((item) => (
          <PosterCell key={item.id} item={item} onOpen={() => nav.push({ kind: 'detail', id: item.id })} />
        ))}
      </div>
    </NativeScreen>
  )
}
