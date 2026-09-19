import { Pause, Play, X } from 'lucide-react'

import { cn } from '@/lib/utils'

import { formatEta, formatGb, formatSpeed, QUEUE_LABEL } from '../../../shared/format'
import { useMobileState } from '../../../shared/mobile-state-context'
import { Backdrop, Poster } from '../../../shared/poster'
import type { MediaItem, QueueItem } from '../../../shared/types'
import { ProgressLine } from '../../../shared/ui'

function GlassButton({ onClick, label, children, className }: { onClick: () => void; label: string; children: React.ReactNode; className?: string }) {
  return (
    <button
      type="button"
      onClick={(e) => {
        e.stopPropagation()
        onClick()
      }}
      aria-label={label}
      className={cn('m-material-chip m-press flex size-tap items-center justify-center rounded-full text-white', className)}
    >
      {children}
    </button>
  )
}

function stats(q: QueueItem): string {
  if (q.state === 'downloading') {
    return `${formatSpeed(q.speedMbps)} · ${formatEta(q.etaMin)} left · ${formatGb(q.sizeGb)}`
  }
  return `${QUEUE_LABEL[q.state]} · ${formatGb(q.sizeGb)}`
}

function QueueCard({ q, item, onOpen }: { q: QueueItem; item: MediaItem; onOpen: () => void }) {
  const { togglePause, removeFromQueue } = useMobileState()
  const canToggle = q.state === 'downloading' || q.state === 'paused'
  return (
    <div className="relative overflow-hidden rounded-[18px] shadow-[0_14px_40px_rgba(0,0,0,0.45)]">
      <Backdrop item={item} className="absolute inset-0" />
      <div className="absolute inset-0 bg-black/45" />
      <button type="button" onClick={onOpen} className="relative flex w-full items-center gap-4 p-4 text-left">
        <Poster item={item} showTitle={false} className="w-16 shrink-0 shadow-[0_8px_20px_rgba(0,0,0,0.5)]" radius="rounded-[8px]" />
        <div className="min-w-0 flex-1 text-white">
          <div className="truncate text-m-title font-bold">{item.title}</div>
          {q.episode !== undefined && <div className="truncate text-m-footnote text-white/75">{q.episode}</div>}
          <div className="m-nums mt-2 flex items-baseline justify-between text-m-caption text-white/70">
            <span>{stats(q)}</span>
            <span className="font-semibold text-white">{q.progress.toFixed(1)}%</span>
          </div>
          <ProgressLine value={q.progress} kind={item.kind} muted={q.state === 'paused'} className={cn('mt-1.5 h-1.5 bg-white/15', q.state === 'downloading' && (item.kind === 'movie' ? 'cine-glow-movie' : 'cine-glow-tv'))} />
        </div>
      </button>
      <div className="absolute top-3 right-3 flex gap-2">
        {canToggle ? (
          <GlassButton onClick={() => togglePause(q.id)} label={q.state === 'paused' ? 'Resume' : 'Pause'}>
            {q.state === 'paused' ? <Play className="size-4 fill-current" /> : <Pause className="size-4 fill-current" />}
          </GlassButton>
        ) : null}
        <GlassButton onClick={() => removeFromQueue(q.id)} label="Remove" className="size-9 text-white/70"><X className="size-4" /></GlassButton>
      </div>
    </div>
  )
}

export function ActivityView({ onOpen }: { onOpen: (id: number) => void }) {
  const { queue, media } = useMobileState()
  const speed = queue.filter((q) => q.state === 'downloading').reduce((s, q) => s + q.speedMbps, 0)

  return (
    <div className="m-scroll h-full pb-32" style={{ paddingTop: 'calc(var(--safe-top) + 8px)' }}>
      <div className="px-screen pb-4">
        <h1 className="text-m-display font-extrabold tracking-[-0.03em]">Activity</h1>
        <div className="m-nums text-m-footnote text-muted-foreground">{queue.length} in queue · {formatSpeed(speed)} total</div>
      </div>
      <div className="m-stagger space-y-3 px-3 [&>div]:m-enter-fade-up">
        {queue.map((q) => (
          <QueueCard key={q.id} q={q} item={media(q.mediaId)} onOpen={() => onOpen(q.mediaId)} />
        ))}
        {queue.length === 0 && <p className="px-2 py-10 text-center text-m-body text-muted-foreground">Nothing in the queue.</p>}
      </div>
    </div>
  )
}
