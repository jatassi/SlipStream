import { useState } from 'react'

import { Eye, EyeOff, Star, UserSearch, Zap } from 'lucide-react'

import { cn } from '@/lib/utils'

import { formatGb, KIND_LABEL, KIND_TEXT } from '../../shared/format'
import { useMobileState } from '../../shared/mobile-state-context'
import { Backdrop, Poster } from '../../shared/poster'
import type { MediaItem } from '../../shared/types'
import { StatusPill } from '../../shared/ui'
import { Sheet } from './sheet'

function SheetHeader({ item }: { item: MediaItem }) {
  return (
    <div className="relative">
      <Backdrop item={item} className="h-52 rounded-t-sheet" />
      <div className="absolute inset-x-0 top-0 flex justify-center pt-2.5">
        <span className="h-1.5 w-10 rounded-full bg-white/50" />
      </div>
      <div className="relative -mt-20 flex items-end gap-4 px-5">
        <Poster item={item} className="w-24 shrink-0 shadow-[0_14px_40px_rgba(0,0,0,0.55)]" radius="rounded-[10px]" />
        <div className="min-w-0 flex-1 pb-1">
          <div className={cn('text-m-caption font-bold tracking-[0.12em] uppercase', KIND_TEXT[item.kind])}>{KIND_LABEL[item.kind]}</div>
          <h2 className="mt-0.5 text-[26px] leading-[1.05] font-extrabold tracking-[-0.03em] text-balance">{item.title}</h2>
          <div className="m-nums mt-1.5 text-m-footnote text-muted-foreground">{item.year} · {item.length}</div>
        </div>
      </div>
    </div>
  )
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[12px] bg-foreground/6 px-3 py-2.5">
      <div className="text-m-caption text-muted-foreground">{label}</div>
      <div className="mt-0.5 truncate text-m-footnote font-semibold">{value}</div>
    </div>
  )
}

function Action({ icon: Icon, label, active = false, onClick, tint }: { icon: typeof Zap; label: string; active?: boolean; onClick?: () => void; tint: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn('m-press flex h-12 flex-1 items-center justify-center gap-2 rounded-full text-m-footnote font-bold', active ? 'text-black' : 'bg-foreground/10 text-foreground')}
      style={active ? { background: tint, boxShadow: `0 0 20px color-mix(in oklch, ${tint} 50%, transparent)` } : undefined}
    >
      <Icon className="size-4" />
      {label}
    </button>
  )
}

export function DetailSheet({ id, onClose }: { id: number; onClose: () => void }) {
  const { media, toggleMonitored } = useMobileState()
  const [expanded, setExpanded] = useState(false)
  const item = media(id)
  const tint = item.kind === 'movie' ? 'var(--movie-500)' : 'var(--tv-500)'

  return (
    <Sheet onClose={onClose} header={<SheetHeader item={item} />}>
      <div className="px-5 pt-4 pb-16">
        <div className="flex items-center gap-2">
          <StatusPill status={item.status} />
          <span className="m-nums inline-flex items-center gap-1 text-m-caption font-semibold text-amber-400"><Star className="size-3 fill-current" />{item.rating.toFixed(1)}</span>
          <span className="text-m-caption text-muted-foreground">{item.genres.join(' · ')}</span>
        </div>
        <div className="mt-4 flex gap-2">
          <Action icon={UserSearch} label="Search" tint={tint} />
          <Action icon={Zap} label="Auto" tint={tint} />
          <Action icon={item.monitored ? Eye : EyeOff} label={item.monitored ? 'Monitored' : 'Off'} active={item.monitored} tint={tint} onClick={() => toggleMonitored(item.id)} />
        </div>
        <button type="button" onClick={() => setExpanded((v) => !v)} className="mt-5 w-full text-left">
          <span className={cn('block text-m-body leading-[22px] text-foreground/85', !expanded && 'line-clamp-3')}>{item.overview}</span>
        </button>
        <div className="mt-5 grid grid-cols-2 gap-2">
          <Fact label="Quality" value={item.quality} />
          <Fact label="Size" value={formatGb(item.sizeGb)} />
          <Fact label="Profile" value={item.profile} />
          <Fact label="Studio" value={item.studio} />
        </div>
      </div>
    </Sheet>
  )
}
