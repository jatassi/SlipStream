import { History } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import { ErrorState } from '@/components/data/error-state'
import { PageHeader } from '@/components/layout/page-header'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import type { HistoryEntry } from '@/types'

import { HistoryTable, HistoryTableSkeleton } from './history-components'
import { ClearHistoryAction, HistoryFilters } from './history-filters'
import { HistoryPagination } from './history-pagination'
import { useHistoryPage } from './use-history-page'

export function HistoryPage() {
  const s = useHistoryPage()

  if (s.isError) {
    return (
      <div>
        <PageHeader title="History" />
        <ErrorState onRetry={s.refetch} />
      </div>
    )
  }

  const description = s.isLoading ? <Skeleton className="h-4 w-48" /> : 'View past activity and events'

  return (
    <div>
      <PageHeader
        title="History"
        description={description}
        actions={<ClearHistoryAction isLoading={s.isLoading} onConfirm={s.handleClearHistory} />}
      />
      <HistoryFilters
        mediaType={s.mediaType}
        datePreset={s.datePreset}
        eventTypes={s.eventTypes}
        isLoading={s.isLoading}
        onMediaTypeChange={s.handleMediaTypeChange}
        onDatePresetChange={s.handleDatePresetChange}
        onToggleEventType={s.handleToggleEventType}
        onResetEventTypes={s.handleResetEventTypes}
      />
      <Card>
        <CardContent className="p-0">
          <HistoryCardBody isLoading={s.isLoading} items={s.history?.items} expandedId={s.expandedId} onToggleExpanded={s.handleToggleExpanded} />
        </CardContent>
      </Card>
      {!s.isLoading && s.history ? (
        <HistoryPagination page={s.page} totalPages={s.history.totalPages} onPreviousPage={s.handlePreviousPage} onNextPage={s.handleNextPage} onPageSelect={s.handlePageSelect} />
      ) : null}
    </div>
  )
}

function HistoryCardBody({
  isLoading,
  items,
  expandedId,
  onToggleExpanded,
}: {
  isLoading: boolean
  items: HistoryEntry[] | undefined
  expandedId: number | null
  onToggleExpanded: (id: number, hasDetails: boolean) => void
}) {
  if (isLoading) {
    return <HistoryTableSkeleton />
  }
  if (!items?.length) {
    return (
      <EmptyState
        icon={<History className="size-8" />}
        title="No history"
        description="Activity history will appear here"
        className="py-8"
      />
    )
  }
  return <HistoryTable items={items} expandedId={expandedId} onToggleExpanded={onToggleExpanded} />
}
