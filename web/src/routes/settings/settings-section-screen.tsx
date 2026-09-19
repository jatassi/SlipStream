import type { ReactNode } from 'react'

import type { LucideIcon } from 'lucide-react'

import { Group, IconTile, Row } from '@/components/grouped-list'
import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'

export type SectionEntry = {
  title: string
  href: string
  icon: LucideIcon
  tile: string
  count?: number
  warnings?: number
}

export function SettingsSectionScreen({
  title,
  entries,
}: {
  title: string
  entries: SectionEntry[]
}) {
  const back = usePushBack()

  return (
    <Screen title={title} back={back}>
      <Group>
        {entries.map((entry) => (
          <Row
            key={entry.href}
            leading={
              <IconTile className={entry.tile}>
                <entry.icon />
              </IconTile>
            }
            title={entry.title}
            href={entry.href}
            trailing={entryDetail(entry)}
            chevron
          />
        ))}
      </Group>
    </Screen>
  )
}

function entryDetail({ count, warnings }: SectionEntry): ReactNode | undefined {
  const hasWarnings = warnings !== undefined && warnings > 0
  if (count === undefined && !hasWarnings) {
    return undefined
  }
  return (
    <span className="nums text-footnote flex items-center gap-1.5">
      {count === undefined ? null : <span>{count}</span>}
      {hasWarnings ? <span className="text-muted-foreground/50">·</span> : null}
      {hasWarnings ? (
        <span className="text-amber-400">
          {warnings} {warnings === 1 ? 'warning' : 'warnings'}
        </span>
      ) : null}
    </span>
  )
}
