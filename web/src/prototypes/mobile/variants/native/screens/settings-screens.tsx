import { AlertTriangle, CheckCircle2, Loader2 } from 'lucide-react'

import { HEALTH, SETTINGS, TASKS } from '../../../shared/mock-data'
import { Group, IconTile, Row } from '../grouped-list'
import { NativeScreen } from '../native-screen'

export function SettingsSectionScreen({ sectionId, onBack }: { sectionId: string; onBack: () => void }) {
  const section = SETTINGS.find((s) => s.id === sectionId) ?? SETTINGS[0]
  return (
    <NativeScreen title={section.title} back={{ label: 'More', onClick: onBack }} bottomInset="calc(var(--safe-bottom) + 24px)">
      <Group>
        {section.items.map((item) => (
          <Row key={item.title} title={item.title} trailing={item.detail} chevron onClick={onBack} />
        ))}
      </Group>
    </NativeScreen>
  )
}

export function SystemScreen({ onBack, backLabel }: { onBack: () => void; backLabel: string }) {
  return (
    <NativeScreen title="System" back={{ label: backLabel, onClick: onBack }} bottomInset="calc(var(--safe-bottom) + 24px)">
      <Group header="Health">
        {HEALTH.map((issue) => (
          <Row key={issue.id} leading={<IconTile className="bg-amber-500"><AlertTriangle /></IconTile>} title={issue.source} subtitle={issue.message} />
        ))}
        <Row leading={<IconTile className="bg-emerald-600"><CheckCircle2 /></IconTile>} title="Download clients" subtitle="qBittorrent · SABnzbd reachable" />
      </Group>
      <Group header="Scheduled tasks">
        {TASKS.map((task) => (
          <Row
            key={task.id}
            title={task.name}
            subtitle={`Last ${task.last} · Next ${task.next}`}
            trailing={task.status === 'running' ? <Loader2 className="size-4 animate-spin text-tv-400" /> : undefined}
            onClick={onBack}
          />
        ))}
      </Group>
    </NativeScreen>
  )
}
