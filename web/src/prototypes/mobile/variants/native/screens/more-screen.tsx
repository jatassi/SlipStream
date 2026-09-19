import { Binoculars, Calendar, Cog, FileInput, FolderOpen, History, LogOut, RotateCcw, Server, Users, Workflow } from 'lucide-react'

import { COUNTS, SETTINGS, TASKS } from '../../../shared/mock-data'
import { Group, IconTile, Row } from '../grouped-list'
import { NativeScreen } from '../native-screen'
import type { NativeNav } from '../use-native-nav'

const SETTINGS_ICON: Record<string, { icon: typeof Cog; className: string }> = {
  media: { icon: FolderOpen, className: 'bg-movie-600' },
  pipeline: { icon: Workflow, className: 'bg-tv-600' },
  general: { icon: Cog, className: 'bg-zinc-600' },
}

function MissingBadge() {
  return (
    <span className="m-nums flex items-center gap-1 text-m-footnote">
      <span className="text-movie-400">{COUNTS.missingMovies}</span>
      <span className="text-muted-foreground/50">|</span>
      <span className="text-tv-400">{COUNTS.missingEpisodes}</span>
    </span>
  )
}

export function MoreScreen({ nav }: { nav: NativeNav }) {
  const openSystem = () => nav.push({ kind: 'system' })
  return (
    <NativeScreen title="More">
      <Group header="Discover">
        <Row leading={<IconTile className="bg-rose-600"><Calendar /></IconTile>} title="Calendar" chevron onClick={openSystem} />
        <Row leading={<IconTile className="bg-violet-600"><Users /></IconTile>} title="Requests" trailing={<span className="m-nums">3</span>} chevron onClick={openSystem} />
        <Row leading={<IconTile className="bg-amber-600"><Binoculars /></IconTile>} title="Missing" trailing={<MissingBadge />} chevron onClick={openSystem} />
        <Row leading={<IconTile className="bg-sky-600"><FileInput /></IconTile>} title="Manual Import" chevron onClick={openSystem} />
        <Row leading={<IconTile className="bg-zinc-600"><History /></IconTile>} title="History" chevron onClick={openSystem} />
      </Group>
      <Group header="Settings">
        {SETTINGS.map((section) => {
          const meta = SETTINGS_ICON[section.id]
          return (
            <Row
              key={section.id}
              leading={<IconTile className={meta.className}><meta.icon /></IconTile>}
              title={section.title}
              trailing={`${section.items.length}`}
              chevron
              onClick={() => nav.push({ kind: 'settings', sectionId: section.id })}
            />
          )
        })}
        <Row leading={<IconTile className="bg-emerald-600"><Server /></IconTile>} title="System" trailing={`${TASKS.filter((t) => t.status === 'running').length} running`} chevron onClick={openSystem} />
      </Group>
      <Group>
        <Row leading={<LogOut className="size-5 text-amber-400" />} title="Log out" tone="warning" onClick={openSystem} />
        <Row leading={<RotateCcw className="size-5 text-destructive" />} title="Restart SlipStream" tone="destructive" onClick={openSystem} />
      </Group>
      <p className="px-screen text-center text-m-caption text-muted-foreground">SlipStream 0.9.4 · connected to nas.local:8080</p>
    </NativeScreen>
  )
}
