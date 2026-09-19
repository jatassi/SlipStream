import { AlertTriangle, Loader2 } from 'lucide-react'

import { formatEta, formatSpeed } from '../../../shared/format'
import { useMobileState } from '../../../shared/mobile-state-context'
import { COUNTS, HEALTH, HISTORY, STORAGE, TASKS } from '../../../shared/mock-data'
import { DenseRow, SectionLabel, Tag } from '../dense'
import { QueueRows } from './activity-view'

function Stat({ label, value, tone }: { label: string; value: string; tone?: string }) {
  return (
    <div className="rounded-[8px] bg-foreground/5 px-2.5 py-2">
      <div className={`m-nums font-mono text-[18px] leading-[22px] font-semibold ${tone ?? ''}`}>{value}</div>
      <div className="text-[11px] text-muted-foreground">{label}</div>
    </div>
  )
}

function Stats() {
  const { queue } = useMobileState()
  const pct = ((STORAGE.usedTb / STORAGE.totalTb) * 100).toFixed(0)
  return (
    <div className="grid grid-cols-4 gap-1.5 px-3 pt-3">
      <Stat label="Movies" value={String(COUNTS.movies)} tone="text-movie-400" />
      <Stat label="Series" value={String(COUNTS.series)} tone="text-tv-400" />
      <Stat label="Missing" value={String(COUNTS.missingMovies + COUNTS.missingEpisodes)} tone="text-amber-400" />
      <Stat label={`Queue · ${pct}% disk`} value={String(queue.length)} />
    </div>
  )
}

export function BoardView({ onOpen, onScope }: { onOpen: (id: number) => void; onScope: (scope: 'activity' | 'system') => void }) {
  const { queue, media } = useMobileState()
  const active = queue.filter((q) => q.state === 'downloading')
  const speed = active.reduce((s, q) => s + q.speedMbps, 0)
  const eta = Math.max(0, ...active.map((q) => q.etaMin))

  return (
    <>
      <Stats />
      <SectionLabel trailing={`${HEALTH.length} open`}>Health</SectionLabel>
      <div className="con-hairline">
        {HEALTH.map((issue) => (
          <DenseRow key={issue.id} leading={<AlertTriangle className="size-4 text-amber-400" />} title={issue.message} meta={issue.source} onClick={() => onScope('system')} />
        ))}
      </div>
      <SectionLabel trailing={`${formatSpeed(speed)} · ${formatEta(eta)}`}>Queue</SectionLabel>
      <QueueRows items={queue.slice(0, 4)} onOpen={onOpen} compact />
      {queue.length > 4 && (
        <button type="button" onClick={() => onScope('activity')} className="m-press-row w-full px-3 py-2 text-left font-mono text-[12px] text-tv-400">
          + {queue.length - 4} more in Activity
        </button>
      )}
      <SectionLabel>Recent</SectionLabel>
      <div className="con-hairline">
        {HISTORY.slice(0, 5).map((h) => (
          <DenseRow
            key={h.id}
            leading={<Tag className={h.event === 'Failed' ? 'text-red-400' : undefined}>{h.event}</Tag>}
            title={media(h.mediaId).title}
            meta={h.detail}
            trailing={h.ago}
            onClick={() => onOpen(h.mediaId)}
          />
        ))}
      </div>
      <SectionLabel>Tasks</SectionLabel>
      <div className="con-hairline">
        {TASKS.map((t) => (
          <DenseRow
            key={t.id}
            leading={t.status === 'running' ? <Loader2 className="size-3.5 animate-spin text-tv-400" /> : <span className="con-led text-foreground/25" />}
            title={t.name}
            meta={`last ${t.last} · next ${t.next}`}
            onClick={() => onScope('system')}
          />
        ))}
      </div>
    </>
  )
}
