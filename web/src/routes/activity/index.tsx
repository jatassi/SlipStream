import { Link } from '@tanstack/react-router'
import { AlertTriangle, Download, Film, Loader2, Tv } from 'lucide-react'

import { ErrorState } from '@/components/data/error-state'
import { PageHeader } from '@/components/layout/page-header'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'
import type { ClientError, QueueItem } from '@/types'

import { DownloadsTable } from './downloads-table'
import type { MediaFilter } from './use-activity-page'
import { useActivityPage } from './use-activity-page'

function CountBadge({ count, className }: { count: number; className?: string }) {
  if (count <= 0) {
    return null
  }
  return <span className={cn('ml-2 text-xs', className)}>({count})</span>
}

function QueueErrorBanner({ errors, isFetching }: { errors: ClientError[]; isFetching: boolean }) {
  if (errors.length === 0) {
    return null
  }

  const clientNames = errors.map((e) => e.clientName).join(', ')

  return (
    <div className="flex items-center gap-3 rounded-md border border-yellow-500/30 bg-yellow-500/10 px-4 py-3 text-sm text-yellow-200">
      <AlertTriangle className="size-4 shrink-0 text-yellow-500" />
      <span className="flex-1">
        Unable to reach <span className="font-medium text-yellow-100">{clientNames}</span> — showing
        last known data
      </span>
      {isFetching ? (
        <span className="flex items-center gap-1.5 text-xs text-yellow-400">
          <Loader2 className="size-3 animate-spin" />
          Retrying…
        </span>
      ) : null}
    </div>
  )
}

function DownloadsSkeleton() {
  return (
    <div className="divide-border divide-y">
      {Array.from({ length: 6 }, (_, i) => (
        <div key={i} className="flex items-center gap-4 px-4 py-3">
          <Skeleton className="size-10 shrink-0 rounded" />
          <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-4 gap-y-0.5">
            <div className="shrink-0 space-y-1">
              <Skeleton className="h-4 w-36" />
              <Skeleton className="h-3 w-16" />
            </div>
            <div className="min-w-[200px] flex-1 basis-56 space-y-1.5">
              <Skeleton className="h-2 w-full rounded-full" />
              <div className="flex items-center text-xs">
                <Skeleton className="h-3 w-24" />
                <Skeleton className="mx-auto h-3 w-16" />
                <Skeleton className="h-3 w-12" />
              </div>
            </div>
          </div>
          <div className="flex shrink-0 gap-1">
            <Skeleton className="size-8 rounded-md" />
            <Skeleton className="size-8 rounded-md" />
          </div>
        </div>
      ))}
    </div>
  )
}

type FilterTabsProps = {
  isLoading: boolean
  totalCount: number
  movieCount: number
  seriesCount: number
}

function FilterTabs({ isLoading, totalCount, movieCount, seriesCount }: FilterTabsProps) {
  return (
    <TabsList>
      <TabsTrigger
        value="all"
        className="data-active:glow-media-sm px-4 data-active:bg-white data-active:text-black"
      >
        All
        {!isLoading && <CountBadge count={totalCount} className="data-active:text-black/60" />}
      </TabsTrigger>
      <TabsTrigger
        value="movies"
        className="data-active:glow-movie data-active:bg-white data-active:text-black"
      >
        <Film className="mr-1.5 size-4" />
        Movies
        {!isLoading && <CountBadge count={movieCount} className="text-muted-foreground" />}
      </TabsTrigger>
      <TabsTrigger
        value="series"
        className="data-active:glow-tv data-active:bg-white data-active:text-black"
      >
        <Tv className="mr-1.5 size-4" />
        Series
        {!isLoading && <CountBadge count={seriesCount} className="text-muted-foreground" />}
      </TabsTrigger>
    </TabsList>
  )
}

const FILTER_TITLES: Record<MediaFilter, string> = {
  all: 'All Downloads',
  movies: 'Movie Downloads',
  series: 'Series Downloads',
}

type DownloadsCardBodyProps = {
  isLoading: boolean
  clientErrors: ClientError[]
  isFetching: boolean
  filteredItems: QueueItem[]
}

function DownloadsCardBody({
  isLoading,
  clientErrors,
  isFetching,
  filteredItems,
}: DownloadsCardBodyProps) {
  if (isLoading) {
    return <DownloadsSkeleton />
  }

  return (
    <>
      {clientErrors.length > 0 && (
        <div className="px-4 pt-4">
          <QueueErrorBanner errors={clientErrors} isFetching={isFetching} />
        </div>
      )}
      <DownloadsTable items={filteredItems} />
    </>
  )
}

type DownloadsCardProps = {
  filter: MediaFilter
  setFilter: (v: MediaFilter) => void
  isLoading: boolean
  isFetching: boolean
  filteredItems: QueueItem[]
  clientErrors: ClientError[]
  movieCount: number
  seriesCount: number
  totalCount: number
}

function DownloadsCard({
  filter,
  setFilter,
  isLoading,
  isFetching,
  filteredItems,
  clientErrors,
  movieCount,
  seriesCount,
  totalCount,
}: DownloadsCardProps) {
  return (
    <Tabs value={filter} onValueChange={(v) => setFilter(v as MediaFilter)} className="space-y-4">
      <div className={cn(isLoading && 'pointer-events-none opacity-50')}>
        <FilterTabs
          isLoading={isLoading}
          totalCount={totalCount}
          movieCount={movieCount}
          seriesCount={seriesCount}
        />
      </div>

      <TabsContent value={filter}>
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Download className="size-5" />
              {FILTER_TITLES[filter]}
            </CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <DownloadsCardBody
              isLoading={isLoading}
              clientErrors={clientErrors}
              isFetching={isFetching}
              filteredItems={filteredItems}
            />
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  )
}

export function ActivityPage() {
  const state = useActivityPage()

  if (state.isError) {
    return (
      <div>
        <PageHeader title="Downloads" />
        <ErrorState onRetry={state.refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="Downloads"
        description={
          state.isLoading ? <Skeleton className="h-4 w-44" /> : 'Monitor active downloads'
        }
        actions={
          <Link to="/activity/history">
            <Button variant="outline">View History</Button>
          </Link>
        }
      />
      <DownloadsCard
        filter={state.filter}
        setFilter={state.setFilter}
        isLoading={state.isLoading}
        isFetching={state.isFetching}
        filteredItems={state.filteredItems}
        clientErrors={state.clientErrors}
        movieCount={state.movieCount}
        seriesCount={state.seriesCount}
        totalCount={state.totalCount}
      />
    </div>
  )
}
