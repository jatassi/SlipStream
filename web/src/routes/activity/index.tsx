import { useState, useRef, useEffect, useCallback, createContext, useContext } from 'react'
import { Link } from '@tanstack/react-router'
import { Pause, Play, Trash2, Film, Tv, Download, FastForward, AlertTriangle, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ProgressBar } from '@/components/media/ProgressBar'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { PosterImage } from '@/components/media/PosterImage'
import {
  useQueue,
  useRemoveFromQueue,
  usePauseQueueItem,
  useResumeQueueItem,
  useFastForwardQueueItem,
  useMovie,
  useSeriesDetail,
} from '@/hooks'
import { formatBytes, formatSpeed, formatEta } from '@/lib/formatters'
import { toast } from 'sonner'
import type { QueueItem, ClientError } from '@/types'

type MediaFilter = 'all' | 'movies' | 'series'

// Context to share title column width across all rows
interface TitleWidthContextType {
  registerWidth: (id: string, width: number) => void
  unregisterWidth: (id: string) => void
  maxWidth: number
}

const TitleWidthContext = createContext<TitleWidthContextType>({
  registerWidth: () => {},
  unregisterWidth: () => {},
  maxWidth: 0,
})

function DownloadRow({ item }: { item: QueueItem }) {
  const [showReleaseName, setShowReleaseName] = useState(false)
  const titleRef = useRef<HTMLDivElement>(null)
  const { registerWidth, unregisterWidth, maxWidth } = useContext(TitleWidthContext)
  const rowId = `${item.clientId}-${item.id}`

  // Measure and report title width
  useEffect(() => {
    const measure = () => {
      if (titleRef.current) {
        registerWidth(rowId, titleRef.current.scrollWidth)
      }
    }
    measure()
    // Re-measure when release name visibility changes
    const timer = setTimeout(measure, 0)
    return () => {
      clearTimeout(timer)
      unregisterWidth(rowId)
    }
  }, [rowId, showReleaseName, registerWidth, unregisterWidth])

  const removeMutation = useRemoveFromQueue()
  const pauseMutation = usePauseQueueItem()
  const resumeMutation = useResumeQueueItem()
  const fastForwardMutation = useFastForwardQueueItem()

  // Fetch media data for poster and year
  const { data: movie } = useMovie(item.mediaType === 'movie' && item.movieId ? item.movieId : 0)
  const { data: series } = useSeriesDetail(item.mediaType === 'series' && item.seriesId ? item.seriesId : 0)

  // Get tmdbId/tvdbId for poster lookup
  const tmdbId = item.mediaType === 'movie' ? movie?.tmdbId : series?.tmdbId
  const tvdbId = item.mediaType === 'series' ? series?.tvdbId : undefined

  const isMovie = item.mediaType === 'movie'
  const isSeries = item.mediaType === 'series'

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

  // Format title suffix (year for movies, episode/season identifier for series)
  const getTitleSuffix = () => {
    if (isMovie) {
      return movie?.year ? `(${movie.year})` : ''
    }
    // Series: show episode or season identifier
    if (item.episode && item.season) {
      return `S${String(item.season).padStart(2, '0')}E${String(item.episode).padStart(2, '0')}`
    }
    if (item.isSeasonPack && item.season) {
      return `S${String(item.season).padStart(2, '0')}`
    }
    if (item.isCompleteSeries) {
      return 'Complete Series'
    }
    return ''
  }

  // Format progress text (condensed)
  const downloadedFormatted = formatBytes(item.downloadedSize)
  const totalFormatted = formatBytes(item.size)
  // Extract just the number from total and keep units from total
  const totalParts = totalFormatted.match(/^([\d.]+)\s*(.+)$/)
  const downloadedParts = downloadedFormatted.match(/^([\d.]+)\s*(.+)$/)

  let progressText: string
  if (totalParts && downloadedParts && totalParts[2] === downloadedParts[2]) {
    // Same units, show condensed format
    progressText = `${downloadedParts[1]}/${totalParts[1]} ${totalParts[2]}`
  } else {
    progressText = `${downloadedFormatted}/${totalFormatted}`
  }

  // Icon class with themed glow on hover
  const actionIconClass = cn(
    'size-4 transition-all',
    isMovie && 'group-hover/btn:icon-glow-movie',
    isSeries && 'group-hover/btn:icon-glow-tv'
  )

  const titleSuffix = getTitleSuffix()

  return (
    <div
      className={cn(
        'flex items-center gap-4 px-4 py-3 transition-colors',
        isMovie && 'hover:bg-movie-500/5',
        isSeries && 'hover:bg-tv-500/5',
        !isMovie && !isSeries && 'hover:bg-accent/50'
      )}
    >
      {/* Poster */}
      <div className="shrink-0 self-center">
        {tmdbId || tvdbId ? (
          <div className="size-10 rounded overflow-hidden">
            <PosterImage
              tmdbId={tmdbId}
              tvdbId={tvdbId}
              alt={item.title}
              type={isMovie ? 'movie' : 'series'}
              className="size-full object-cover"
            />
          </div>
        ) : (
          <div className={cn(
            'flex size-10 items-center justify-center rounded',
            isMovie && 'bg-movie-500/20 text-movie-500',
            isSeries && 'bg-tv-500/20 text-tv-500',
            !isMovie && !isSeries && 'bg-muted text-muted-foreground'
          )}>
            {isMovie ? (
              <Film className="size-5" />
            ) : isSeries ? (
              <Tv className="size-5" />
            ) : (
              <Download className="size-5" />
            )}
          </div>
        )}
      </div>

      {/* Title + Progress - wraps when space is limited */}
      <div className="flex-1 min-w-0 flex flex-wrap items-center gap-x-4 gap-y-0.5">
        {/* Title - width synced across all rows via context */}
        <div
          className="shrink-0 self-center transition-[width] duration-150 ease-out overflow-hidden"
          style={{ width: maxWidth > 0 ? maxWidth : 'auto' }}
        >
          <div ref={titleRef} className="inline-block">
            <div
              className={cn(
                'font-medium cursor-pointer transition-colors whitespace-nowrap',
                isMovie && 'hover:text-movie-500',
                isSeries && 'hover:text-tv-500'
              )}
              title={item.releaseName}
              onClick={() => setShowReleaseName(!showReleaseName)}
            >
              {item.title}
              {titleSuffix && (
                <span className="text-muted-foreground ml-1.5">{titleSuffix}</span>
              )}
            </div>
            {showReleaseName && (
              <div className="text-xs text-muted-foreground mt-0.5 whitespace-nowrap animate-[slide-down-fade_150ms_ease-out]">
                {item.releaseName}
              </div>
            )}
          </div>
        </div>

        {/* Progress - fills remaining space, wraps under title when needed */}
        <div className="flex-1 basis-56 min-w-[200px] self-center">
          <div className="relative py-2">
            <ProgressBar
              value={item.progress}
              size="sm"
              variant={isMovie ? 'movie' : isSeries ? 'tv' : undefined}
            />
            <div className="absolute left-0 right-0 mt-1 flex items-center text-xs text-muted-foreground">
              <span>{progressText}</span>
              <span className="mx-auto">
                {item.status === 'downloading' ? formatSpeed(item.downloadSpeed) : ''}
              </span>
              <span>
                {item.status === 'downloading' ? formatEta(item.eta) : ''}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="flex gap-1 shrink-0 self-center">
        {item.status === 'downloading' && (
          <Button
            variant="ghost"
            size="icon"
            onClick={handlePause}
            disabled={pauseMutation.isPending}
            title="Pause"
            className="group/btn"
          >
            <Pause className={actionIconClass} />
          </Button>
        )}
        {item.status === 'paused' && (
          <Button
            variant="ghost"
            size="icon"
            onClick={handleResume}
            disabled={resumeMutation.isPending}
            title="Resume"
            className="group/btn"
          >
            <Play className={actionIconClass} />
          </Button>
        )}
        {item.clientType === 'mock' && item.status !== 'completed' && (
          <Button
            variant="ghost"
            size="icon"
            onClick={handleFastForward}
            disabled={fastForwardMutation.isPending}
            title="Fast Forward"
            className="group/btn"
          >
            <FastForward className={actionIconClass} />
          </Button>
        )}
        <ConfirmDialog
          trigger={
            <Button variant="ghost" size="icon" title="Remove" className="group/btn">
              <Trash2 className={actionIconClass} />
            </Button>
          }
          title="Remove download"
          description={`Are you sure you want to remove "${item.title}" from the queue?`}
          confirmLabel="Remove"
          variant="destructive"
          onConfirm={() => handleRemove(false)}
        />
      </div>
    </div>
  )
}

function DownloadsTable({ items }: { items: QueueItem[] }) {
  const [widths, setWidths] = useState<Map<string, number>>(new Map())

  const registerWidth = useCallback((id: string, width: number) => {
    setWidths((prev) => {
      const next = new Map(prev)
      next.set(id, width)
      return next
    })
  }, [])

  const unregisterWidth = useCallback((id: string) => {
    setWidths((prev) => {
      const next = new Map(prev)
      next.delete(id)
      return next
    })
  }, [])

  const maxWidth = Math.max(0, ...widths.values())

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
    <TitleWidthContext.Provider value={{ registerWidth, unregisterWidth, maxWidth }}>
      <div className="divide-y divide-border">
        {items.map((item) => (
          <DownloadRow key={`${item.clientId}-${item.id}`} item={item} />
        ))}
      </div>
    </TitleWidthContext.Provider>
  )
}

function QueueErrorBanner({ errors, isFetching }: { errors: ClientError[]; isFetching: boolean }) {
  if (errors.length === 0) return null

  const clientNames = errors.map((e) => e.clientName).join(', ')

  return (
    <div className="flex items-center gap-3 rounded-md border border-yellow-500/30 bg-yellow-500/10 px-4 py-3 text-sm text-yellow-200">
      <AlertTriangle className="size-4 shrink-0 text-yellow-500" />
      <span className="flex-1">
        Unable to reach <span className="font-medium text-yellow-100">{clientNames}</span> — showing last known data
      </span>
      {isFetching && (
        <span className="flex items-center gap-1.5 text-xs text-yellow-400">
          <Loader2 className="size-3 animate-spin" />
          Retrying…
        </span>
      )}
    </div>
  )
}

export function ActivityPage() {
  const [filter, setFilter] = useState<MediaFilter>('all')
  const { data: queueResponse, isLoading, isError, isFetching, refetch } = useQueue()

  const items = queueResponse?.items ?? []
  const clientErrors = queueResponse?.errors ?? []

  // Filter by media type and sort by title (completed downloads are already filtered by backend)
  const filteredItems = items.filter((item) => {
    if (filter === 'all') return true
    if (filter === 'movies') return item.mediaType === 'movie'
    if (filter === 'series') return item.mediaType === 'series'
    return true
  }).sort((a, b) => a.title.localeCompare(b.title))

  // Count items by media type
  const movieCount = items.filter((q) => q.mediaType === 'movie').length
  const seriesCount = items.filter((q) => q.mediaType === 'series').length
  const totalCount = items.length

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
          <TabsTrigger
            value="all"
            className="px-4 data-active:bg-white data-active:text-black data-active:glow-media-sm"
          >
            All
            {totalCount > 0 && (
              <span className="ml-2 text-xs data-active:text-black/60">({totalCount})</span>
            )}
          </TabsTrigger>
          <TabsTrigger
            value="movies"
            className="data-active:bg-white data-active:text-black data-active:glow-movie"
          >
            <Film className="size-4 mr-1.5" />
            Movies
            {movieCount > 0 && (
              <span className="ml-2 text-xs text-muted-foreground">({movieCount})</span>
            )}
          </TabsTrigger>
          <TabsTrigger
            value="series"
            className="data-active:bg-white data-active:text-black data-active:glow-tv"
          >
            <Tv className="size-4 mr-1.5" />
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
              {clientErrors.length > 0 && (
                <div className="px-4 pt-4">
                  <QueueErrorBanner errors={clientErrors} isFetching={isFetching} />
                </div>
              )}
              <DownloadsTable items={filteredItems} />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
