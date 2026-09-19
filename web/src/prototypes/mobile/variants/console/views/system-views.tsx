import { AlertTriangle, CheckCircle2, Loader2, Search, Zap } from 'lucide-react'

import { useMobileState } from '../../../shared/mobile-state-context'
import { HEALTH, SETTINGS, TASKS } from '../../../shared/mock-data'
import { Poster } from '../../../shared/poster'
import { DenseRow, InlineButton, SectionLabel, Tag } from '../dense'

export function MissingView({ onOpen, query }: { onOpen: (id: number) => void; query: string }) {
  const { library } = useMobileState()
  const q = query.trim().toLowerCase()
  const items = library.filter((m) => (m.status === 'missing' || m.status === 'upgradable') && m.title.toLowerCase().includes(q))

  return (
    <>
      <SectionLabel trailing={`${items.length} wanted`}>Missing & upgrades</SectionLabel>
      <div className="con-hairline">
        {items.map((item) => (
          <DenseRow
            key={item.id}
            leading={<Poster item={item} showTitle={false} className="w-7" radius="rounded-[3px]" />}
            title={item.title}
            meta={item.status === 'missing' ? `Missing · wants ${item.profile}` : `Upgradable · ${item.quality} → ${item.profile}`}
            trailing={
              <>
                <InlineButton onClick={() => onOpen(item.id)}><Search className="size-3.5" /></InlineButton>
                <InlineButton tone="accent" onClick={() => onOpen(item.id)}><Zap className="size-3.5" />Auto</InlineButton>
              </>
            }
            onClick={() => onOpen(item.id)}
          />
        ))}
      </div>
    </>
  )
}

export function SystemView({ query }: { query: string }) {
  const q = query.trim().toLowerCase()
  const settings = SETTINGS.flatMap((s) => s.items.map((i) => ({ ...i, section: s.title }))).filter((i) => `${i.section} ${i.title}`.toLowerCase().includes(q))

  return (
    <>
      <SectionLabel trailing={`${HEALTH.length} warnings`}>Health</SectionLabel>
      <div className="con-hairline">
        {HEALTH.map((issue) => (
          <DenseRow key={issue.id} leading={<AlertTriangle className="size-4 text-amber-400" />} title={issue.message} meta={issue.source} trailing={<InlineButton>Retest</InlineButton>} />
        ))}
        <DenseRow leading={<CheckCircle2 className="size-4 text-emerald-400" />} title="qBittorrent · SABnzbd reachable" meta="Download clients" />
      </div>
      <SectionLabel>Tasks</SectionLabel>
      <div className="con-hairline">
        {TASKS.map((t) => (
          <DenseRow
            key={t.id}
            leading={t.status === 'running' ? <Loader2 className="size-3.5 animate-spin text-tv-400" /> : <span className="con-led text-foreground/25" />}
            title={t.name}
            meta={`last ${t.last} · next ${t.next}`}
            trailing={<InlineButton>Run</InlineButton>}
          />
        ))}
      </div>
      <SectionLabel trailing={`${settings.length}`}>Settings</SectionLabel>
      <div className="con-hairline">
        {settings.map((item) => (
          <DenseRow key={`${item.section}-${item.title}`} leading={<Tag>{item.section}</Tag>} title={item.title} trailing={item.detail} onClick={() => undefined} />
        ))}
      </div>
      <p className="px-3 pt-6 font-mono text-[11px] text-muted-foreground">SlipStream 0.9.4 · go1.23 · sqlite wal · uptime 6d 4h</p>
    </>
  )
}
