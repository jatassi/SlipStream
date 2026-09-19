import { useState } from 'react'

import { Hourglass,PackageCheck, Pause, Play } from 'lucide-react'

import { formatEta, formatGb, formatSpeed, QUEUE_LABEL } from '../../../shared/format'
import { useMobileState } from '../../../shared/mobile-state-context'
import { Poster } from '../../../shared/poster'
import type { MediaItem, QueueItem } from '../../../shared/types'
import { ProgressLine } from '../../../shared/ui'
import { ActionSheet } from '../action-sheet'
import { Group } from '../grouped-list'
import { NativeScreen } from '../native-screen'

function statsLine(q: QueueItem): string {
  if (q.state === 'downloading') {
    return `${q.progress.toFixed(1)}% · ${formatSpeed(q.speedMbps)} · ${formatEta(q.etaMin)} left`
  }
  if (q.state === 'paused') {
    return `${q.progress.toFixed(1)}% · Paused · ${formatGb(q.sizeGb)}`
  }
  return `${QUEUE_LABEL[q.state]} · ${formatGb(q.sizeGb)}`
}

function TrailingControl({ q, onToggle }: { q: QueueItem; onToggle: () => void }) {
  if (q.state === 'importing') {
    return <PackageCheck className="size-5 text-emerald-400" />
  }
  if (q.state === 'queued') {
    return <Hourglass className="size-5 text-muted-foreground" />
  }
  const Icon = q.state === 'paused' ? Play : Pause
  return (
    <button
      type="button"
      onClick={(e) => {
        e.stopPropagation()
        onToggle()
      }}
      aria-label={q.state === 'paused' ? 'Resume' : 'Pause'}
      className="m-press flex size-tap items-center justify-center rounded-full text-foreground"
    >
      <Icon className="size-5 fill-current" />
    </button>
  )
}

function QueueRow({ q, item, onOpen, onToggle }: { q: QueueItem; item: MediaItem; onOpen: () => void; onToggle: () => void }) {
  return (
    <div className="m-press-row flex items-center gap-3 px-4 py-3">
      <button type="button" onClick={onOpen} className="flex min-w-0 flex-1 items-center gap-3 text-left">
        <Poster item={item} showTitle={false} className="w-10" radius="rounded-[5px]" />
        <div className="min-w-0 flex-1">
          <div className="flex items-baseline gap-2">
            <span className="truncate text-m-body font-medium">{item.title}</span>
            {q.episode !== undefined && <span className="m-nums shrink-0 text-m-caption text-muted-foreground">{q.episode.split(' · ')[0]}</span>}
          </div>
          <div className="mt-0.5 truncate font-mono text-[11px] text-muted-foreground/80">{q.release}</div>
          <ProgressLine value={q.progress} kind={item.kind} muted={q.state === 'paused'} className="mt-2" />
          <div className="m-nums mt-1.5 text-m-caption text-muted-foreground">{statsLine(q)}</div>
        </div>
      </button>
      <TrailingControl q={q} onToggle={onToggle} />
    </div>
  )
}

export function ActivityScreen() {
  const { queue, media, togglePause, removeFromQueue } = useMobileState()
  const [selected, setSelected] = useState<QueueItem | null>(null)
  const active = queue.filter((q) => q.state === 'downloading')
  const speed = active.reduce((sum, q) => sum + q.speedMbps, 0)

  return (
    <NativeScreen title="Activity">
      <p className="m-nums -mt-2 px-screen pb-4 text-m-footnote text-muted-foreground">
        {active.length} downloading · {formatSpeed(speed)} · {queue.length} in queue
      </p>
      <Group>
        {queue.map((q) => (
          <QueueRow key={q.id} q={q} item={media(q.mediaId)} onOpen={() => setSelected(q)} onToggle={() => togglePause(q.id)} />
        ))}
        {queue.length === 0 && <div className="px-4 py-6 text-center text-m-body text-muted-foreground">Queue is empty</div>}
      </Group>
      {selected !== null && (
        <ActionSheet
          title={media(selected.mediaId).title}
          subtitle={selected.release}
          onClose={() => setSelected(null)}
          actions={[
            { label: selected.state === 'paused' ? 'Resume' : 'Pause', onClick: () => togglePause(selected.id) },
            { label: 'Remove from queue', onClick: () => removeFromQueue(selected.id), destructive: true },
            { label: 'Remove and blocklist release', onClick: () => removeFromQueue(selected.id), destructive: true },
          ]}
        />
      )}
    </NativeScreen>
  )
}
