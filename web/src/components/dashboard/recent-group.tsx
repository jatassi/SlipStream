import { ArrowDownToLine } from 'lucide-react'

import { Group, IconTile, Row, RowSkeleton } from '@/components/grouped-list'
import { useHistory } from '@/hooks'
import { formatRelativeTime } from '@/lib/formatters'
import { eventLook } from '@/lib/history-utils'
import { useUIStore } from '@/stores'
import type { HistoryEntry } from '@/types/history'

const SKELETONS = ['recent-a', 'recent-b', 'recent-c', 'recent-d', 'recent-e'] as const

function recentHref(entry: HistoryEntry): string | undefined {
  if (entry.mediaType === 'movie') {
    return `/movies/${entry.mediaId}`
  }
  if (entry.seriesId) {
    return `/series/${entry.seriesId}`
  }
  return undefined
}

function RecentRow({ entry }: { entry: HistoryEntry }) {
  const look = eventLook(entry)
  const Icon = look.icon
  const title = entry.mediaTitle ?? `${entry.mediaType} #${entry.mediaId}`
  return (
    <Row
      href={recentHref(entry)}
      leading={
        <IconTile className={look.className}>
          <Icon />
        </IconTile>
      }
      title={title}
      subtitle={`${look.label} · ${entry.quality ?? entry.source ?? look.label}`}
      trailing={<span className="nums text-footnote">{formatRelativeTime(entry.createdAt)}</span>}
    />
  )
}

function EmptyRecent() {
  return (
    <Row
      leading={
        <IconTile className="bg-zinc-600">
          <ArrowDownToLine />
        </IconTile>
      }
      title="No recent activity"
      subtitle="History is empty"
    />
  )
}

export function RecentGroup() {
  const globalLoading = useUIStore((s) => s.globalLoading)
  const { data, isLoading } = useHistory({ pageSize: 5 })

  if (isLoading || globalLoading) {
    return (
      <Group header="Recent" inset={false} className="mb-0">
        {SKELETONS.map((id) => (
          <RowSkeleton key={id} trailing />
        ))}
      </Group>
    )
  }

  const items = data?.items ?? []
  return (
    <Group header="Recent" inset={false} className="mb-0">
      {items.length === 0 ? <EmptyRecent /> : items.map((entry) => <RecentRow key={entry.id} entry={entry} />)}
    </Group>
  )
}
