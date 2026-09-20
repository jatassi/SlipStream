import { Group, IconTile, Row } from '@/components/grouped-list'
import { formatRelativeTime } from '@/lib/formatters'
import { eventLook } from '@/lib/history-utils'
import type { HistoryEntry } from '@/types'

import type { DayGroup } from './history-utils'
import { getDetailsText } from './history-utils'

function entryHref(entry: HistoryEntry): string | undefined {
  if (entry.mediaType === 'movie') {
    return `/movies/${entry.mediaId}`
  }
  if (entry.seriesId) {
    return `/series/${entry.seriesId}`
  }
  return undefined
}

function entryQualifier(entry: HistoryEntry): string | undefined {
  if (entry.mediaType === 'movie') {
    return entry.year === undefined ? undefined : String(entry.year)
  }
  return entry.mediaQualifier
}

function HistoryRow({ entry }: { entry: HistoryEntry }) {
  const look = eventLook(entry)
  const Icon = look.icon
  const title = entry.mediaTitle ?? `${entry.mediaType} #${entry.mediaId}`
  const qualifier = entryQualifier(entry)

  return (
    <Row
      href={entryHref(entry)}
      leading={
        <IconTile className={look.className}>
          <Icon />
        </IconTile>
      }
      title={
        <>
          {title}
          {qualifier === undefined ? null : (
            <span className="text-muted-foreground ml-1.5">{qualifier}</span>
          )}
        </>
      }
      subtitle={`${look.label} · ${getDetailsText(entry)}`}
      trailing={<span className="nums text-footnote">{formatRelativeTime(entry.createdAt)}</span>}
    />
  )
}

export function HistoryDayGroup({ group }: { group: DayGroup }) {
  return (
    <Group header={group.label}>
      {group.entries.map((entry) => (
        <HistoryRow key={entry.id} entry={entry} />
      ))}
    </Group>
  )
}
