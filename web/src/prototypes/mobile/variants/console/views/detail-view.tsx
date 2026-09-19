import { ChevronLeft, Eye, EyeOff, Search, Zap } from 'lucide-react'

import { formatGb, KIND_LABEL } from '../../../shared/format'
import { useMobileState } from '../../../shared/mobile-state-context'
import { Poster } from '../../../shared/poster'
import { DenseRow, InlineButton, SectionLabel, StatusTag, Tag } from '../dense'

function KeyValue({ k, v }: { k: string; v: string }) {
  return (
    <div className="flex items-baseline justify-between gap-4 px-3 py-2">
      <span className="text-[12px] text-muted-foreground">{k}</span>
      <span className="m-nums truncate text-right font-mono text-[12px]">{v}</span>
    </div>
  )
}

export function DetailView({ id, onBack }: { id: number; onBack: () => void }) {
  const { media, toggleMonitored, queue } = useMobileState()
  const item = media(id)
  const inQueue = queue.filter((q) => q.mediaId === id)

  return (
    <div className="m-scroll h-full" style={{ paddingTop: 'calc(var(--safe-top) + 40px)', paddingBottom: 'calc(var(--safe-bottom) + 120px)' }}>
      <button type="button" onClick={onBack} className="m-press-dim flex h-10 items-center gap-0.5 px-2 font-mono text-[12px] text-tv-400">
        <ChevronLeft className="size-4" /> back
      </button>
      <div className="flex gap-3 px-3">
        <Poster item={item} showTitle={false} className="w-20 shrink-0" radius="rounded-[6px]" />
        <div className="min-w-0 flex-1">
          <div className="font-mono text-[11px] tracking-[0.08em] text-muted-foreground uppercase">{KIND_LABEL[item.kind]} · {item.year}</div>
          <h1 className="mt-0.5 text-[20px] leading-[24px] font-bold tracking-[-0.015em]">{item.title}</h1>
          <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
            <StatusTag status={item.status} />
            <Tag>{item.quality}</Tag>
            <Tag>{item.profile}</Tag>
          </div>
          <div className="mt-3 flex gap-1.5">
            <InlineButton><Search className="size-3.5" />Search</InlineButton>
            <InlineButton tone="accent"><Zap className="size-3.5" />Auto</InlineButton>
            <InlineButton onClick={() => toggleMonitored(item.id)}>{item.monitored ? <Eye className="size-3.5" /> : <EyeOff className="size-3.5" />}{item.monitored ? 'Monitored' : 'Off'}</InlineButton>
          </div>
        </div>
      </div>
      <p className="px-3 pt-4 text-[13px] leading-[19px] text-foreground/80">{item.overview}</p>
      <SectionLabel>File</SectionLabel>
      <div className="con-hairline">
        <KeyValue k="Path" v={`/mnt/media/${item.kind === 'movie' ? 'movies' : 'tv'}/${item.title} (${item.year})`} />
        <KeyValue k="Size" v={formatGb(item.sizeGb)} />
        <KeyValue k="Length" v={item.length} />
        <KeyValue k="Studio" v={item.studio} />
        <KeyValue k="Rating" v={item.rating.toFixed(1)} />
        <KeyValue k="Added" v={item.added} />
      </div>
      {inQueue.length > 0 && (
        <>
          <SectionLabel>In queue</SectionLabel>
          <div className="con-hairline">
            {inQueue.map((q) => (
              <DenseRow key={q.id} title={<span className="font-mono text-[11px] break-all">{q.release}</span>} meta={`${q.state} · ${q.client}`} trailing={`${q.progress.toFixed(1)}%`} />
            ))}
          </div>
        </>
      )}
    </div>
  )
}
