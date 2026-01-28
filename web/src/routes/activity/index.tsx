import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Pause, Play, Trash2, Film, Tv, Download, FastForward } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ProgressBar } from '@/components/media/ProgressBar'
import { QualityBadge } from '@/components/media/QualityBadge'
import { FormatBadges } from '@/components/media/FormatBadges'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  useQueue,
  useRemoveFromQueue,
  usePauseQueueItem,
  useResumeQueueItem,
  useFastForwardQueueItem,
} from '@/hooks'
import { formatBytes, formatSpeed, formatEta, formatSeriesTitle } from '@/lib/formatters'
import { toast } from 'sonner'
import type { QueueItem } from '@/types'

type MediaFilter = 'all' | 'movies' | 'series'

function DownloadRow({ item }: { item: QueueItem }) {
  const removeMutation = useRemoveFromQueue()
  const pauseMutation = usePauseQueueItem()
  const resumeMutation = useResumeQueueItem()
  const fastForwardMutation = useFastForwardQueueItem()

  const handlePause = async () => {
    try {
      await pauseMutation.mutateAsync({ clientId: item.clientId, id: item.id })
      toast.success('Download paused')
    } catch {
      toast.error('Failed to pause download')
    }
  }

  const handleResume = async () => {
    try {
      await resumeMutation.mutateAsync({ clientId: item.clientId, id: item.id })
      toast.success('Download resumed')
    } catch {
      toast.error('Failed to resume download')
    }
  }

  const handleFastForward = async () => {
    try {
      await fastForwardMutation.mutateAsync({ clientId: item.clientId, id: item.id })
      toast.success('Download completed')
    } catch {
      toast.error('Failed to fast forward download')
    }
  }

  const handleRemove = async (deleteFiles: boolean) => {
    try {
      await removeMutation.mutateAsync({
        clientId: item.clientId,
        id: item.id,
        deleteFiles,
      })
      toast.success(deleteFiles ? 'Download removed with files' : 'Download removed')
    } catch {
      toast.error('Failed to remove download')
    }
  }

  // Format title for display
  const displayTitle =
    item.mediaType === 'series'
      ? formatSeriesTitle(item.title, item.season, item.episode)
      : item.title

  // Format progress text
  const progressText = `${formatBytes(item.downloadedSize)} / ${formatBytes(item.size)}`

  return (
    <TableRow>
      {/* Title */}
      <TableCell>
        <div className="flex items-center gap-3">
          <div className="flex size-8 items-center justify-center rounded bg-muted shrink-0">
            {item.mediaType === 'movie' ? (
              <Film className="size-4" />
            ) : (
              <Tv className="size-4" />
            )}
          </div>
          <span className="font-medium truncate max-w-[300px]" title={displayTitle}>
            {displayTitle}
          </span>
        </div>
      </TableCell>

      {/* Quality */}
      <TableCell>
        {item.quality && <QualityBadge quality={item.quality} />}
      </TableCell>

      {/* Attributes */}
      <TableCell>
        <FormatBadges
          source={item.source}
          codec={item.codec}
          attributes={item.attributes}
        />
      </TableCell>

      {/* Progress */}
      <TableCell className="min-w-[180px]">
        <div className="space-y-1">
          <ProgressBar value={item.progress} size="sm" />
          <div className="text-xs text-muted-foreground">{progressText}</div>
        </div>
      </TableCell>

      {/* Time Left */}
      <TableCell className="text-muted-foreground w-[100px]">
        {item.status === 'downloading' ? formatEta(item.eta) : '--'}
      </TableCell>

      {/* Speed */}
      <TableCell className="text-muted-foreground w-[100px]">
        {item.status === 'downloading' ? formatSpeed(item.downloadSpeed) : '--'}
      </TableCell>

      {/* Actions */}
      <TableCell>
        <div className="flex gap-1">
          {item.status === 'downloading' && (
            <Button
              variant="ghost"
              size="icon"
              onClick={handlePause}
              disabled={pauseMutation.isPending}
              title="Pause"
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
              title="Resume"
            >
              <Play className="size-4" />
            </Button>
          )}
          {item.clientType === 'mock' && item.status !== 'completed' && (
            <Button
              variant="ghost"
              size="icon"
              onClick={handleFastForward}
              disabled={fastForwardMutation.isPending}
              title="Fast Forward"
            >
              <FastForward className="size-4" />
            </Button>
          )}
          <ConfirmDialog
            trigger={
              <Button variant="ghost" size="icon" title="Remove">
                <Trash2 className="size-4" />
              </Button>
            }
            title="Remove download"
            description={`Are you sure you want to remove "${displayTitle}" from the queue?`}
            confirmLabel="Remove"
            variant="destructive"
            onConfirm={() => handleRemove(false)}
          />
        </div>
      </TableCell>
    </TableRow>
  )
}

function DownloadsTable({ items }: { items: QueueItem[] }) {
  if (items.length === 0) {
    return (
      <EmptyState
        icon={<Download className="size-8" />}
        title="No downloads"
        description="Downloads will appear here when they start"
        className="py-8"
      />
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Title</TableHead>
          <TableHead>Quality</TableHead>
          <TableHead>Attributes</TableHead>
          <TableHead>Progress</TableHead>
          <TableHead className="w-[100px]">Time Left</TableHead>
          <TableHead className="w-[100px]">Speed</TableHead>
          <TableHead className="w-[100px]">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => (
          <DownloadRow key={`${item.clientId}-${item.id}`} item={item} />
        ))}
      </TableBody>
    </Table>
  )
}

export function ActivityPage() {
  const [filter, setFilter] = useState<MediaFilter>('all')
  const { data: queue, isLoading, isError, refetch } = useQueue()

  // Filter by media type and sort by title (completed downloads are already filtered by backend)
  const filteredItems = (queue?.filter((item) => {
    if (filter === 'all') return true
    if (filter === 'movies') return item.mediaType === 'movie'
    if (filter === 'series') return item.mediaType === 'series'
    return true
  }) || []).sort((a, b) => a.title.localeCompare(b.title))

  // Count items by media type
  const movieCount = queue?.filter((q) => q.mediaType === 'movie').length ?? 0
  const seriesCount = queue?.filter((q) => q.mediaType === 'series').length ?? 0
  const totalCount = queue?.length ?? 0

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Downloads" />
        <LoadingState variant="list" />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Downloads" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="Downloads"
        description="Monitor active downloads"
        actions={
          <Link to="/activity/history">
            <Button variant="outline">View History</Button>
          </Link>
        }
      />

      <Tabs value={filter} onValueChange={(v) => setFilter(v as MediaFilter)} className="space-y-4">
        <TabsList>
          <TabsTrigger value="all">
            All
            {totalCount > 0 && (
              <span className="ml-2 text-xs text-muted-foreground">({totalCount})</span>
            )}
          </TabsTrigger>
          <TabsTrigger value="movies">
            Movies
            {movieCount > 0 && (
              <span className="ml-2 text-xs text-muted-foreground">({movieCount})</span>
            )}
          </TabsTrigger>
          <TabsTrigger value="series">
            Series
            {seriesCount > 0 && (
              <span className="ml-2 text-xs text-muted-foreground">({seriesCount})</span>
            )}
          </TabsTrigger>
        </TabsList>

        <TabsContent value={filter}>
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Download className="size-5" />
                {filter === 'all' && 'All Downloads'}
                {filter === 'movies' && 'Movie Downloads'}
                {filter === 'series' && 'Series Downloads'}
              </CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <DownloadsTable items={filteredItems} />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
