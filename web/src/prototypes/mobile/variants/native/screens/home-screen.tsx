import { AlertTriangle, ArrowDownToLine, CheckCircle2, HardDrive, RefreshCw, XCircle } from 'lucide-react'

import { formatEta, formatSpeed } from '../../../shared/format'
import { useMobileState } from '../../../shared/mobile-state-context'
import { HEALTH, HISTORY, STORAGE } from '../../../shared/mock-data'
import { Poster } from '../../../shared/poster'
import type { HistoryEvent } from '../../../shared/types'
import { ProgressLine } from '../../../shared/ui'
import { Group, IconTile, Row } from '../grouped-list'
import { NativeScreen } from '../native-screen'
import type { NativeNav } from '../use-native-nav'

function HealthGroup({ onOpen }: { onOpen: () => void }) {
  return (
    <Group header="Health">
      {HEALTH.map((issue) => (
        <Row
          key={issue.id}
          leading={<IconTile className="bg-amber-500"><AlertTriangle /></IconTile>}
          title={issue.source}
          subtitle={issue.message}
          chevron
          onClick={onOpen}
        />
      ))}
    </Group>
  )
}

function StorageGroup() {
  const pct = (STORAGE.usedTb / STORAGE.totalTb) * 100
  return (
    <Group header="Storage">
      <div className="px-4 py-3">
        <div className="flex items-center gap-3">
          <IconTile className="bg-zinc-600"><HardDrive /></IconTile>
          <div className="min-w-0 flex-1">
            <div className="flex items-baseline justify-between">
              <span className="text-m-body font-medium">{STORAGE.usedTb.toFixed(2)} TB used</span>
              <span className="m-nums text-m-footnote text-muted-foreground">of {STORAGE.totalTb} TB</span>
            </div>
            <div className="mt-2 flex h-1.5 w-full gap-0.5 overflow-hidden rounded-full bg-foreground/10">
              <div className="h-full rounded-l-full bg-movie-500" style={{ width: `${(STORAGE.folders[0].usedTb / STORAGE.totalTb) * 100}%` }} />
              <div className="h-full rounded-r-full bg-tv-500" style={{ width: `${(STORAGE.folders[1].usedTb / STORAGE.totalTb) * 100}%` }} />
            </div>
            <div className="mt-1.5 flex gap-4 text-m-caption text-muted-foreground">
              <span><span className="text-movie-400">●</span> Movies {STORAGE.folders[0].usedTb} TB</span>
              <span><span className="text-tv-400">●</span> Series {STORAGE.folders[1].usedTb} TB</span>
              <span className="m-nums ml-auto">{pct.toFixed(0)}%</span>
            </div>
          </div>
        </div>
      </div>
    </Group>
  )
}

function DownloadingGroup({ nav }: { nav: NativeNav }) {
  const { queue, media } = useMobileState()
  const active = queue.filter((q) => q.state === 'downloading').slice(0, 3)
  return (
    <Group
      header="Downloading"
      action={<button type="button" className="m-press-dim text-m-footnote font-medium text-tv-400" onClick={() => nav.setTab('activity')}>See all</button>}
    >
      {active.length === 0 && <Row title="Nothing downloading" subtitle="The queue is empty" leading={<IconTile className="bg-zinc-600"><ArrowDownToLine /></IconTile>} />}
      {active.map((q) => {
        const item = media(q.mediaId)
        return (
          <Row
            key={q.id}
            leading={<Poster item={item} showTitle={false} className="w-8" radius="rounded-[4px]" />}
            title={item.title}
            subtitle={
              <span className="flex items-center gap-2">
                <ProgressLine value={q.progress} kind={item.kind} className="w-24" />
                <span className="m-nums">{q.progress.toFixed(0)}% · {formatSpeed(q.speedMbps)} · {formatEta(q.etaMin)}</span>
              </span>
            }
            onClick={() => nav.push({ kind: 'detail', id: item.id })}
            chevron
          />
        )
      })}
    </Group>
  )
}

const EVENT_TILE: Record<HistoryEvent['event'], { className: string; icon: typeof CheckCircle2 }> = {
  Imported: { className: 'bg-emerald-600', icon: CheckCircle2 },
  Grabbed: { className: 'bg-tv-600', icon: ArrowDownToLine },
  Upgraded: { className: 'bg-movie-600', icon: RefreshCw },
  Failed: { className: 'bg-red-600', icon: XCircle },
}

function RecentGroup({ nav }: { nav: NativeNav }) {
  const { media } = useMobileState()
  return (
    <Group header="Recent">
      {HISTORY.slice(0, 5).map((event) => {
        const tile = EVENT_TILE[event.event]
        const item = media(event.mediaId)
        return (
          <Row
            key={event.id}
            leading={<IconTile className={tile.className}><tile.icon /></IconTile>}
            title={item.title}
            subtitle={`${event.event} · ${event.detail}`}
            trailing={<span className="m-nums text-m-footnote">{event.ago}</span>}
            onClick={() => nav.push({ kind: 'detail', id: item.id })}
          />
        )
      })}
    </Group>
  )
}

export function HomeScreen({ nav }: { nav: NativeNav }) {
  return (
    <NativeScreen title="Dashboard">
      <div className="m-stagger [&>section]:m-enter-fade-up">
        <HealthGroup onOpen={() => nav.push({ kind: 'system' })} />
        <StorageGroup />
        <DownloadingGroup nav={nav} />
        <RecentGroup nav={nav} />
      </div>
    </NativeScreen>
  )
}
