import { Link } from '@tanstack/react-router'
import { Film, Tv } from 'lucide-react'

import { StorageCard } from '@/components/dashboard/storage-card'
import { HealthWidget } from '@/components/health'
import { PageHeader } from '@/components/layout/page-header'
import { ProgressBar } from '@/components/media/progress-bar'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useGlobalLoading, useHistory, useQueue } from '@/hooks'
import { useStorage } from '@/hooks/use-storage'
import { formatRelativeTime } from '@/lib/formatters'
import { eventTypeLabels } from '@/lib/history-utils'
import type { HistoryEntry } from '@/types/history'

function QueueLoadingSkeleton() {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Active Downloads</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {[1, 2, 3].map((i) => (
          <div key={i} className="space-y-2">
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-2 w-full" />
          </div>
        ))}
      </CardContent>
    </Card>
  )
}

function QueuePreview() {
  const globalLoading = useGlobalLoading()
  const { data: queue, isLoading: queryLoading } = useQueue()
  const isLoading = queryLoading || globalLoading

  if (isLoading) {return <QueueLoadingSkeleton />}

  const activeDownloads = queue?.items.filter((q) => q.status === 'downloading') ?? []

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-base">Active Downloads</CardTitle>
        <Link to="/activity">
          <Button variant="ghost" size="sm">
            View all
          </Button>
        </Link>
      </CardHeader>
      <CardContent>
        {activeDownloads.length === 0 ? (
          <p className="text-muted-foreground text-sm">No active downloads</p>
        ) : (
          <div className="space-y-4">
            {activeDownloads.slice(0, 5).map((item) => (
              <div key={item.id} className="space-y-1">
                <div className="flex items-center justify-between">
                  <span className="max-w-[200px] truncate text-sm font-medium">{item.title}</span>
                  <span className="text-muted-foreground text-xs">{item.progress.toFixed(1)}%</span>
                </div>
                <ProgressBar value={item.progress} size="sm" />
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function ActivityLoadingSkeleton() {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Recent Activity</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {[1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="flex items-center gap-3">
            <Skeleton className="size-8 rounded" />
            <div className="flex-1 space-y-1">
              <Skeleton className="h-3 w-40" />
              <Skeleton className="h-2 w-24" />
            </div>
          </div>
        ))}
      </CardContent>
    </Card>
  )
}

function ActivityItem({ item }: { item: HistoryEntry }) {
  return (
    <div className="flex items-center gap-3">
      <div className="bg-muted flex size-8 items-center justify-center rounded">
        {item.mediaType === 'movie' ? <Film className="size-4" /> : <Tv className="size-4" />}
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">
          {item.mediaTitle ?? `${item.mediaType} #${item.mediaId}`}
        </p>
        <p className="text-muted-foreground text-xs">
          {eventTypeLabels[item.eventType] || item.eventType} - {formatRelativeTime(item.createdAt)}
        </p>
      </div>
    </div>
  )
}

function RecentActivity() {
  const globalLoading = useGlobalLoading()
  const { data: history, isLoading: queryLoading } = useHistory({ pageSize: 10 })
  const isLoading = queryLoading || globalLoading

  if (isLoading) {return <ActivityLoadingSkeleton />}

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-base">Recent Activity</CardTitle>
        <Link to="/activity/history">
          <Button variant="ghost" size="sm">
            View all
          </Button>
        </Link>
      </CardHeader>
      <CardContent>
        {history?.items.length ? (
          <div className="space-y-3">
            {history.items.slice(0, 5).map((item) => (
              <ActivityItem key={item.id} item={item} />
            ))}
          </div>
        ) : (
          <p className="text-muted-foreground text-sm">No recent activity</p>
        )}
      </CardContent>
    </Card>
  )
}

export function DashboardPage() {
  const globalLoading = useGlobalLoading()
  const storage = useStorage()

  return (
    <div>
      <PageHeader title="Dashboard" description="Overview of your media library" />

      {/* Stats grid */}
      <div className="mb-6 grid gap-4 md:grid-cols-2">
        <StorageCard storage={storage.data} loading={storage.isLoading || globalLoading} />
        <HealthWidget />
      </div>

      {/* Activity section */}
      <div className="grid gap-4 md:grid-cols-2">
        <QueuePreview />
        <RecentActivity />
      </div>
    </div>
  )
}
