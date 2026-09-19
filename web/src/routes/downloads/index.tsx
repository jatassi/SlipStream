import { AlertTriangle } from 'lucide-react'

import { ErrorState } from '@/components/data/error-state'
import { Group, IconTile, Row, RowSkeleton } from '@/components/grouped-list'
import { Screen } from '@/components/screen/screen'
import { Segmented, type SegmentedOption } from '@/components/ui/segmented'
import { Skeleton } from '@/components/ui/skeleton'
import type { ClientError, QueueItem } from '@/types'

import { QueueRow } from './queue-row'
import type { MediaFilter } from './use-activity-page'
import { useActivityPage } from './use-activity-page'

const FILTER_OPTIONS: SegmentedOption<MediaFilter>[] = [
  { value: 'all', label: 'All' },
  { value: 'movies', label: 'Movies' },
  { value: 'series', label: 'Series' },
]

const SKELETONS = ['queue-a', 'queue-b', 'queue-c'] as const

function SummaryLine({ isLoading, summary }: { isLoading: boolean; summary: string }) {
  if (isLoading) {
    return <Skeleton className="mx-screen -mt-1 mb-4 h-[18px] w-52" />
  }
  return (
    <p className="nums text-footnote text-muted-foreground px-screen -mt-1 pb-4">{summary}</p>
  )
}

function UnreachableGroup({ errors }: { errors: ClientError[] }) {
  if (errors.length === 0) {
    return null
  }
  return (
    <Group>
      {errors.map((error) => (
        <Row
          key={error.clientId}
          tone="warning"
          leading={
            <IconTile className="bg-amber-500">
              <AlertTriangle />
            </IconTile>
          }
          title={`Unable to reach ${error.clientName}`}
          subtitle="Showing last known data"
        />
      ))}
    </Group>
  )
}

function QueueGroup({ isLoading, items }: { isLoading: boolean; items: QueueItem[] }) {
  if (isLoading) {
    return (
      <Group>
        {SKELETONS.map((id) => (
          <RowSkeleton key={id} leading="poster" progress trailing />
        ))}
      </Group>
    )
  }
  if (items.length === 0) {
    return (
      <Group>
        <Row title="Queue is empty" subtitle="Downloads appear here when they start" />
      </Group>
    )
  }
  return (
    <Group>
      {items.map((item) => (
        <QueueRow key={`${item.clientId}-${item.id}`} item={item} />
      ))}
    </Group>
  )
}

export function ActivityPage() {
  const state = useActivityPage()

  if (state.isError) {
    return (
      <Screen title="Downloads">
        <ErrorState onRetry={state.refetch} />
      </Screen>
    )
  }

  return (
    <Screen title="Downloads">
      <SummaryLine isLoading={state.isLoading} summary={state.summary} />
      <div className="px-screen pb-4">
        <Segmented
          label="Filter downloads"
          value={state.filter}
          onChange={state.setFilter}
          options={FILTER_OPTIONS}
        />
      </div>
      <UnreachableGroup errors={state.clientErrors} />
      <QueueGroup isLoading={state.isLoading} items={state.filteredItems} />
    </Screen>
  )
}
