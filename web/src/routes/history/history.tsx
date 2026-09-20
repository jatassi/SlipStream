import { useRef, useState } from 'react'

import { EmptyState } from '@/components/data/empty-state'
import { ErrorState } from '@/components/data/error-state'
import { Group, RowSkeleton } from '@/components/grouped-list'
import { usePushBack } from '@/components/layout/use-push-back'
import { ActionPresenter } from '@/components/presenter'
import { Screen } from '@/components/screen/screen'

import { HistoryFilters } from './history-filters'
import { HistoryDayGroup } from './history-groups'
import { groupByDay } from './history-utils'
import { useHistoryPage } from './use-history-page'

const SKELETONS = ['history-a', 'history-b', 'history-c', 'history-d', 'history-e', 'history-f'] as const

type PageState = ReturnType<typeof useHistoryPage>

function ClearHistoryAction({ onConfirm }: { onConfirm: () => Promise<void> }) {
  const [open, setOpen] = useState(false)
  const anchor = useRef<HTMLButtonElement>(null)

  return (
    <>
      <button
        ref={anchor}
        type="button"
        onClick={() => {
          setOpen(true)
        }}
        className="press-dim min-h-tap text-title focus-visible:ring-ring px-2 outline-none focus-visible:ring-[3px]"
      >
        Clear
      </button>
      <ActionPresenter
        open={open}
        onOpenChange={setOpen}
        title="Clear history"
        description="Every history entry is removed. This cannot be undone."
        wide="dialog"
        anchor={anchor}
        actions={[
          {
            label: 'Clear History',
            destructive: true,
            onClick: () => {
              void onConfirm()
            },
          },
        ]}
      />
    </>
  )
}

function LoadMore({ state }: { state: PageState }) {
  if (!state.hasMore) {
    return null
  }
  return (
    <div className="px-screen pb-6">
      <button
        type="button"
        onClick={state.handleLoadMore}
        className="press-dim min-h-tap text-body bg-card focus-visible:ring-ring rounded-card text-primary w-full font-medium outline-none focus-visible:ring-[3px]"
      >
        Load more
      </button>
    </div>
  )
}

function HistoryBody({ state }: { state: PageState }) {
  if (state.isLoading) {
    return (
      <Group>
        {SKELETONS.map((id) => (
          <RowSkeleton key={id} trailing />
        ))}
      </Group>
    )
  }

  if (state.items.length === 0) {
    return <EmptyState title="No history" description="Activity history will appear here" />
  }

  return (
    <>
      {groupByDay(state.items).map((group) => (
        <HistoryDayGroup key={group.key} group={group} />
      ))}
      <LoadMore state={state} />
    </>
  )
}

export function HistoryPage() {
  const state = useHistoryPage()
  const back = usePushBack()

  if (state.isError) {
    return (
      <Screen title="History" back={back}>
        <ErrorState onRetry={state.refetch} />
      </Screen>
    )
  }

  return (
    <Screen
      title="History"
      back={back}
      trailing={<ClearHistoryAction onConfirm={state.handleClearHistory} />}
    >
      <HistoryFilters
        mediaType={state.mediaType}
        datePreset={state.datePreset}
        eventTypes={state.eventTypes}
        onMediaTypeChange={state.handleMediaTypeChange}
        onDatePresetChange={state.handleDatePresetChange}
        onToggleEventType={state.handleToggleEventType}
        onResetEventTypes={state.handleResetEventTypes}
      />
      <HistoryBody state={state} />
    </Screen>
  )
}
