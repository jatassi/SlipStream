import { ArrowUpCircle, Clock, Loader2, ScrollText } from 'lucide-react'

import { ErrorState } from '@/components/data/error-state'
import { Group, IconTile, Row, RowSkeleton } from '@/components/grouped-list'
import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'
import { useScheduledTasks } from '@/hooks'

import { HealthGroup } from './health-group'
import { useHealthPage } from './use-health-page'

function runningTrailing(running: number) {
  if (running === 0) {
    return undefined
  }
  return (
    <span className="text-footnote flex items-center gap-1.5 text-tv-400">
      <Loader2 className="size-4 animate-spin" />
      {running} running
    </span>
  )
}

function SystemGroup() {
  const { data: tasks } = useScheduledTasks()
  const running = tasks?.filter((task) => task.running).length ?? 0

  return (
    <Group>
      <Row
        leading={
          <IconTile className="bg-tv-600">
            <Clock />
          </IconTile>
        }
        title="Scheduled Tasks"
        href="/system/tasks"
        trailing={runningTrailing(running)}
        chevron
      />
      <Row
        leading={
          <IconTile className="bg-zinc-600">
            <ScrollText />
          </IconTile>
        }
        title="Logs"
        href="/system/logs"
        chevron
      />
      <Row
        leading={
          <IconTile className="bg-emerald-600">
            <ArrowUpCircle />
          </IconTile>
        }
        title="Update"
        href="/system/update"
        chevron
      />
    </Group>
  )
}

export function SystemHealthPage() {
  const back = usePushBack()
  const {
    isLoading,
    error,
    isProwlarrMode,
    downloadClients,
    prowlarrItem,
    indexerItems,
    regularCategories,
  } = useHealthPage()

  if (isLoading) {
    return (
      <Screen title="System" back={back}>
        <SystemGroup />
        <Group header="Health">
          {[0, 1, 2, 3].map((index) => (
            <RowSkeleton key={index} />
          ))}
        </Group>
      </Screen>
    )
  }

  if (error) {
    return (
      <Screen title="System" back={back}>
        <SystemGroup />
        <ErrorState title="Failed to load health status" />
      </Screen>
    )
  }

  return (
    <Screen title="System" back={back}>
      <SystemGroup />
      <HealthGroup category="downloadClients" items={downloadClients} />
      <HealthGroup
        category="indexers"
        items={indexerItems}
        leadingItem={isProwlarrMode ? prowlarrItem : undefined}
      />
      {regularCategories.map(({ category, items }) => (
        <HealthGroup key={category} category={category} items={items} />
      ))}
    </Screen>
  )
}
