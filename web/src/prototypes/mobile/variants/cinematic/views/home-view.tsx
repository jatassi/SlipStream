import { AlertTriangle, ChevronRight } from 'lucide-react'

import { cn } from '@/lib/utils'

import { formatEta, formatSpeed, KIND_LABEL, KIND_TEXT } from '../../../shared/format'
import { useMobileState } from '../../../shared/mobile-state-context'
import { HEALTH, HISTORY } from '../../../shared/mock-data'
import { Backdrop, Poster } from '../../../shared/poster'
import type { MediaItem, QueueItem } from '../../../shared/types'
import { ProgressLine, StatusDot } from '../../../shared/ui'

function HeroCard({ q, item, onOpen }: { q: QueueItem; item: MediaItem; onOpen: () => void }) {
  return (
    <button type="button" onClick={onOpen} className="m-press relative w-[calc(100cqw-32px)] overflow-hidden rounded-[22px] text-left shadow-[0_20px_50px_rgba(0,0,0,0.5)]">
      <Backdrop item={item} className="aspect-[4/5]" />
      <div className="absolute inset-0 bg-gradient-to-t from-black/85 via-black/25 to-transparent" />
      <div className="absolute inset-x-0 bottom-0 p-5">
        <div className={cn('text-m-caption font-bold tracking-[0.12em] uppercase', KIND_TEXT[item.kind])}>{KIND_LABEL[item.kind]} · Downloading</div>
        <h2 className="mt-1 text-[30px] leading-[1.02] font-extrabold tracking-[-0.03em] text-balance text-white">{item.title}</h2>
        {q.episode !== undefined && <div className="mt-1 text-m-body font-medium text-white/80">{q.episode}</div>}
        <div className="m-nums mt-3 flex items-baseline justify-between text-m-footnote text-white/75">
          <span className="font-mono text-[11px] text-white/60">{q.release.split(/[.-]/).slice(3, 7).join(' · ')}</span>
          <span className="font-semibold text-white">{q.progress.toFixed(0)}%</span>
        </div>
        <ProgressLine value={q.progress} kind={item.kind} className={cn('mt-2 h-1.5', item.kind === 'movie' ? 'cine-glow-movie' : 'cine-glow-tv')} />
        <div className="m-nums mt-2 text-m-caption text-white/60">{formatSpeed(q.speedMbps)} · {formatEta(q.etaMin)} remaining</div>
      </div>
    </button>
  )
}

function HealthChip() {
  if (HEALTH.length === 0) {
    return null
  }
  return (
    <button type="button" className="m-material-chip m-press inline-flex h-8 items-center gap-1.5 rounded-full pr-3 pl-2 text-m-footnote font-semibold text-amber-300">
      <AlertTriangle className="size-3.5" />
      {HEALTH.length} warnings
    </button>
  )
}

function Rail({ title, items, onOpen, meta }: { title: string; items: MediaItem[]; onOpen: (id: number) => void; meta: (item: MediaItem) => string }) {
  return (
    <section className="mt-8">
      <div className="mb-3 flex items-center justify-between px-screen">
        <h3 className="text-m-title font-bold">{title}</h3>
        <ChevronRight className="size-5 text-muted-foreground" />
      </div>
      <div className="m-scroll-x cine-rail pb-2">
        {items.map((item) => (
          <button key={item.id} type="button" onClick={() => onOpen(item.id)} className="m-press w-[124px] text-left">
            <Poster item={item} className="shadow-[0_10px_30px_rgba(0,0,0,0.45)]" radius="rounded-[12px]" />
            <div className="mt-2 flex items-center gap-1.5 text-m-caption text-muted-foreground">
              <StatusDot status={item.status} />
              <span className="truncate">{meta(item)}</span>
            </div>
          </button>
        ))}
      </div>
    </section>
  )
}

export function HomeView({ onOpen }: { onOpen: (id: number) => void }) {
  const { queue, media, library } = useMobileState()
  const active = queue.filter((q) => q.state === 'downloading')
  const recent = HISTORY.filter((h) => h.event === 'Imported' || h.event === 'Upgraded').map((h) => media(h.mediaId))
  const missing = library.filter((m) => m.status === 'missing')

  return (
    <div className="m-scroll h-full pb-32" style={{ paddingTop: 'calc(var(--safe-top) + 8px)' }}>
      <div className="flex items-end justify-between px-screen pb-4">
        <div>
          <div className="text-m-footnote font-semibold text-muted-foreground">Saturday</div>
          <h1 className="text-m-display font-extrabold tracking-[-0.03em]">Now</h1>
        </div>
        <HealthChip />
      </div>
      <div className="m-scroll-x cine-rail m-enter-fade-up">
        {active.map((q) => (
          <HeroCard key={q.id} q={q} item={media(q.mediaId)} onOpen={() => onOpen(q.mediaId)} />
        ))}
      </div>
      <Rail title="Recently added" items={recent} onOpen={onOpen} meta={(m) => m.quality} />
      <Rail title="Missing" items={missing} onOpen={onOpen} meta={(m) => `${m.year} · wanted`} />
    </div>
  )
}
