import { Link } from '@tanstack/react-router'
import { Pause, Play, X, Film, Tv, Activity } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ProgressBar } from '@/components/media/ProgressBar'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import {
  useQueue,
  useRemoveFromQueue,
  usePauseQueueItem,
  useResumeQueueItem,
} from '@/hooks'
import { formatBytes, formatSpeed, formatEta } from '@/lib/formatters'
import { toast } from 'sonner'
import type { QueueItem } from '@/types'

const statusColors: Record<string, string> = {
  queued: 'bg-gray-500',
  downloading: 'bg-blue-500',
  paused: 'bg-yellow-500',
  completed: 'bg-green-500',
  failed: 'bg-red-500',
}

function QueueItemRow({ item }: { item: QueueItem }) {
  const removeMutation = useRemoveFromQueue()
  const pauseMutation = usePauseQueueItem()
  const resumeMutation = useResumeQueueItem()

  const handlePause = async () => {
    try {
      await pauseMutation.mutateAsync(item.id)
      toast.success('Download paused')
    } catch {
      toast.error('Failed to pause download')
    }
  }

  const handleResume = async () => {
    try {
      await resumeMutation.mutateAsync(item.id)
      toast.success('Download resumed')
    } catch {
      toast.error('Failed to resume download')
    }
  }

  const handleRemove = async () => {
    try {
      await removeMutation.mutateAsync(item.id)
      toast.success('Removed from queue')
    } catch {
      toast.error('Failed to remove from queue')
    }
  }

  return (
    <div className="flex items-center gap-4 p-4 border-b last:border-0">
      <div className="flex size-10 items-center justify-center rounded bg-muted">
        {item.mediaType === 'movie' ? (
          <Film className="size-5" />
        ) : (
          <Tv className="size-5" />
        )}
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <h4 className="font-medium truncate">{item.title}</h4>
          <Badge
            variant="secondary"
            className={`${statusColors[item.status]} text-white text-xs`}
          >
            {item.status}
          </Badge>
        </div>
        <div className="flex items-center gap-4 text-sm text-muted-foreground mt-1">
          <span>{formatBytes(item.size)}</span>
          {item.speed && <span>{formatSpeed(item.speed)}</span>}
          {item.eta && <span>ETA: {formatEta(parseInt(item.eta))}</span>}
        </div>
        {item.status === 'downloading' && (
          <div className="mt-2">
            <ProgressBar value={item.progress} size="sm" showLabel />
          </div>
        )}
      </div>

      <div className="flex gap-1">
        {item.status === 'downloading' && (
          <Button
            variant="ghost"
            size="icon"
            onClick={handlePause}
            disabled={pauseMutation.isPending}
          >
            <Pause className="size-4" />
          </Button>
        )}
        {item.status === 'paused' && (
          <Button
            variant="ghost"
            size="icon"
            onClick={handleResume}
            disabled={resumeMutation.isPending}
          >
            <Play className="size-4" />
          </Button>
        )}
        <ConfirmDialog
          trigger={
            <Button variant="ghost" size="icon">
              <X className="size-4" />
            </Button>
          }
          title="Remove from queue"
          description={`Are you sure you want to remove "${item.title}" from the queue?`}
          confirmLabel="Remove"
          variant="destructive"
          onConfirm={handleRemove}
        />
      </div>
    </div>
  )
}

export function ActivityPage() {
  const { data: queue, isLoading, isError, refetch } = useQueue()

  const activeCount = queue?.filter((q) => q.status === 'downloading').length || 0
  const queuedCount = queue?.filter((q) => q.status === 'queued').length || 0

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Activity" />
        <LoadingState variant="list" />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Activity" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="Activity"
        description="Monitor downloads and activity"
        actions={
          <Link to="/activity/history">
            <Button variant="outline">View History</Button>
          </Link>
        }
      />

      <Tabs defaultValue="queue" className="space-y-4">
        <TabsList>
          <TabsTrigger value="queue">
            Queue
            {activeCount > 0 && (
              <Badge variant="secondary" className="ml-2">
                {activeCount}
              </Badge>
            )}
          </TabsTrigger>
          <TabsTrigger value="queued">
            Queued
            {queuedCount > 0 && (
              <Badge variant="secondary" className="ml-2">
                {queuedCount}
              </Badge>
            )}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="queue">
          <Card>
            <CardHeader>
              <CardTitle>Active Downloads</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              {!queue?.filter((q) => q.status === 'downloading').length ? (
                <EmptyState
                  icon={<Activity className="size-8" />}
                  title="No active downloads"
                  description="Downloads will appear here when they start"
                  className="py-8"
                />
              ) : (
                queue
                  .filter((q) => q.status === 'downloading')
                  .map((item) => <QueueItemRow key={item.id} item={item} />)
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="queued">
          <Card>
            <CardHeader>
              <CardTitle>Pending Downloads</CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              {!queue?.filter((q) => q.status === 'queued').length ? (
                <EmptyState
                  icon={<Activity className="size-8" />}
                  title="No pending downloads"
                  description="Queued downloads will appear here"
                  className="py-8"
                />
              ) : (
                queue
                  .filter((q) => q.status === 'queued')
                  .map((item) => <QueueItemRow key={item.id} item={item} />)
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
