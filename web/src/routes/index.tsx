import { Link } from '@tanstack/react-router'
import {
  Film,
  Tv,
  Download,
} from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useStatus, useQueue, useHistory } from '@/hooks'
import { useStorage } from '@/hooks/useStorage'
import { formatRelativeTime } from '@/lib/formatters'
import { ProgressBar } from '@/components/media/ProgressBar'
import { StorageCard } from '@/components/dashboard/StorageCard'

function StatCard({
  title,
  value,
  icon: Icon,
  description,
  loading,
}: {
  title: string
  value: string | number
  icon: React.ElementType
  description?: string
  loading?: boolean
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
        <Icon className="size-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        {loading ? (
          <Skeleton className="h-8 w-20" />
        ) : (
          <div className="text-2xl font-bold">{value}</div>
        )}
        {description && (
          <p className="text-xs text-muted-foreground">{description}</p>
        )}
      </CardContent>
    </Card>
  )
}

function QueuePreview() {
  const { data: queue, isLoading } = useQueue()

  if (isLoading) {
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

  const activeDownloads = queue?.filter((q) => q.status === 'downloading') || []

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
          <p className="text-sm text-muted-foreground">No active downloads</p>
        ) : (
          <div className="space-y-4">
            {activeDownloads.slice(0, 5).map((item) => (
              <div key={item.id} className="space-y-1">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium truncate max-w-[200px]">
                    {item.title}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    {item.progress.toFixed(1)}%
                  </span>
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

function RecentActivity() {
  const { data: history, isLoading } = useHistory({ pageSize: 10 })

  if (isLoading) {
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
        {!history?.items?.length ? (
          <p className="text-sm text-muted-foreground">No recent activity</p>
        ) : (
          <div className="space-y-3">
            {history.items.slice(0, 5).map((item) => (
              <div key={item.id} className="flex items-center gap-3">
                <div className="flex size-8 items-center justify-center rounded bg-muted">
                  {item.mediaType === 'movie' ? (
                    <Film className="size-4" />
                  ) : (
                    <Tv className="size-4" />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">
                    {item.mediaTitle || `${item.mediaType} #${item.mediaId}`}
                  </p>
                  <p className="text-xs text-muted-foreground">
                    {item.eventType} - {formatRelativeTime(item.createdAt)}
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function DashboardPage() {
  const { data: status, isLoading: statusLoading } = useStatus()
  const { data: queue } = useQueue()
  const storage = useStorage()

  const activeDownloads = queue?.filter((q) => q.status === 'downloading').length || 0

  return (
    <div>
      <PageHeader
        title="Dashboard"
        description="Overview of your media library"
      />

      {/* Stats grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-6">
        <StatCard
          title="Movies"
          value={status?.movieCount ?? 0}
          icon={Film}
          description="In library"
          loading={statusLoading}
        />
        <StatCard
          title="Series"
          value={status?.seriesCount ?? 0}
          icon={Tv}
          description={`${status?.episodeCount ?? 0} episodes`}
          loading={statusLoading}
        />
        <StatCard
          title="Downloads"
          value={activeDownloads}
          icon={Download}
          description="Active"
          loading={statusLoading}
        />
        <StorageCard
          storage={storage?.data}
          loading={storage?.isLoading}
        />
      </div>

      {/* Activity section */}
      <div className="grid gap-4 md:grid-cols-2">
        <QueuePreview />
        <RecentActivity />
      </div>
    </div>
  )
}
