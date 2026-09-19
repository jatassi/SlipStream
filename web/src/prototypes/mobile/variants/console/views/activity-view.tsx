import { useState } from 'react'

import { Ban, Pause, Play, Trash2 } from 'lucide-react'

import { formatEta, formatGb, formatSpeed, kindColor, QUEUE_LABEL } from '../../../shared/format'
import { useMobileState } from '../../../shared/mobile-state-context'
import { Poster } from '../../../shared/poster'
import type { QueueItem } from '../../../shared/types'
import { DenseRow, InlineButton, SectionLabel, Tag } from '../dense'

function meta(q: QueueItem): string {
  if (q.state === 'downloading') {
    return `${formatSpeed(q.speedMbps)} · ${formatEta(q.etaMin)} · ${formatGb(q.sizeGb)}`
  }
  return `${QUEUE_LABEL[q.state]} · ${formatGb(q.sizeGb)} · ${q.client}`
}

function ToggleButton({ q, onToggle }: { q: QueueItem; onToggle: () => void }) {
  if (q.state !== 'downloading' && q.state !== 'paused') {
    return <Tag>{QUEUE_LABEL[q.state]}</Tag>
  }
  const Icon = q.state === 'paused' ? Play : Pause
  return (
    <button
      type="button"
      aria-label={q.state === 'paused' ? 'Resume' : 'Pause'}
      onClick={(e) => {
        e.stopPropagation()
        onToggle()
      }}
      className="m-press flex size-9 items-center justify-center rounded-[8px] bg-foreground/8"
    >
      <Icon className="size-3.5 fill-current" />
    </button>
  )
}

function Expanded({ q, onOpen }: { q: QueueItem; onOpen: () => void }) {
  const { removeFromQueue } = useMobileState()
  return (
    <div className="con-row-detail bg-foreground/3 px-3 pt-1 pb-3">
      <div className="font-mono text-[11px] leading-[16px] break-all text-muted-foreground">{q.release}</div>
      <div className="m-nums mt-1 font-mono text-[11px] text-muted-foreground">{q.protocol} · {q.client} · {formatGb(q.sizeGb)}</div>
      <div className="mt-2.5 flex gap-1.5">
        <InlineButton tone="accent" onClick={onOpen}>Open</InlineButton>
        <InlineButton tone="danger" onClick={() => removeFromQueue(q.id)}><Trash2 className="size-3.5" />Remove</InlineButton>
        <InlineButton tone="danger" onClick={() => removeFromQueue(q.id)}><Ban className="size-3.5" />Blocklist</InlineButton>
      </div>
    </div>
  )
}

export function QueueRows({ items, onOpen, compact = false }: { items: QueueItem[]; onOpen: (id: number) => void; compact?: boolean }) {
  const { media, togglePause } = useMobileState()
  const [expanded, setExpanded] = useState<string | null>(null)

  return (
    <div className="con-hairline">
      {items.map((q) => {
        const item = media(q.mediaId)
        const open = expanded === q.id
        return (
          <DenseRow
            key={q.id}
            leading={<Poster item={item} showTitle={false} className="w-7" radius="rounded-[3px]" />}
            title={<>{item.title}{q.episode !== undefined && <span className="ml-1.5 font-mono text-[11px] text-muted-foreground">{q.episode.split(' · ')[0]}</span>}</>}
            meta={meta(q)}
            trailing={<><span className="font-mono text-foreground">{q.progress.toFixed(1)}%</span><ToggleButton q={q} onToggle={() => togglePause(q.id)} /></>}
            onClick={() => (compact ? onOpen(item.id) : setExpanded(open ? null : q.id))}
          >
            {open ? <Expanded q={q} onOpen={() => onOpen(item.id)} /> : null}
            <div className="con-progress"><span style={{ width: `${q.progress}%`, background: q.state === 'paused' ? 'var(--muted-foreground)' : kindColor(item.kind) }} /></div>
          </DenseRow>
        )
      })}
      {items.length === 0 && <p className="px-3 py-6 font-mono text-[12px] text-muted-foreground">queue empty</p>}
    </div>
  )
}

export function ActivityView({ onOpen, query }: { onOpen: (id: number) => void; query: string }) {
  const { queue, media } = useMobileState()
  const q = query.trim().toLowerCase()
  const items = q.length === 0 ? queue : queue.filter((x) => `${media(x.mediaId).title} ${x.release}`.toLowerCase().includes(q))
  const speed = queue.filter((x) => x.state === 'downloading').reduce((s, x) => s + x.speedMbps, 0)

  return (
    <>
      <SectionLabel trailing={`${items.length} items · ${formatSpeed(speed)}`}>Activity</SectionLabel>
      <QueueRows items={items} onOpen={onOpen} />
    </>
  )
}
