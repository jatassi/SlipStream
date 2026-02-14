import { useState } from 'react'

import { useNavigate, useParams } from '@tanstack/react-router'
import { format } from 'date-fns'
import {
  ArrowLeft,
  Calendar,
  CheckCircle,
  Clock,
  Download,
  Eye,
  EyeOff,
  Loader2,
  Trash2,
  User,
  XCircle,
} from 'lucide-react'
import { toast } from 'sonner'

import { PosterImage } from '@/components/media/PosterImage'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import {
  useCancelRequest,
  useGlobalLoading,
  usePortalDownloads,
  useRequest,
  useUnwatchRequest,
  useWatchRequest,
} from '@/hooks'
import { formatEta } from '@/lib/formatters'
import { usePortalAuthStore } from '@/stores'
import type { RequestStatus } from '@/types'

const STATUS_CONFIG: Record<
  RequestStatus,
  { label: string; icon: React.ReactNode; color: string }
> = {
  pending: {
    label: 'Pending Approval',
    icon: <Clock className="size-4 md:size-5" />,
    color: 'bg-yellow-500',
  },
  approved: {
    label: 'Approved',
    icon: <CheckCircle className="size-4 md:size-5" />,
    color: 'bg-blue-500',
  },
  denied: { label: 'Denied', icon: <XCircle className="size-4 md:size-5" />, color: 'bg-red-500' },
  downloading: {
    label: 'Downloading',
    icon: <Download className="size-4 md:size-5" />,
    color: 'bg-purple-500',
  },
  failed: { label: 'Failed', icon: <XCircle className="size-4 md:size-5" />, color: 'bg-red-700' },
  available: {
    label: 'Available',
    icon: <CheckCircle className="size-4 md:size-5" />,
    color: 'bg-green-500',
  },
  cancelled: {
    label: 'Cancelled',
    icon: <XCircle className="size-4 md:size-5" />,
    color: 'bg-gray-500',
  },
}

export function RequestDetailPage() {
  const params = useParams({ strict: false })
  const navigate = useNavigate()
  const requestId = Number.parseInt(params.id ?? '0', 10)

  const { user } = usePortalAuthStore()
  const globalLoading = useGlobalLoading()
  const { data: request, isLoading: queryLoading, error } = useRequest(requestId)
  const isLoading = queryLoading || globalLoading
  const { data: downloads } = usePortalDownloads()
  const cancelMutation = useCancelRequest()
  const watchMutation = useWatchRequest()
  const unwatchMutation = useUnwatchRequest()

  const [cancelDialogOpen, setCancelDialogOpen] = useState(false)

  // Find ALL active downloads for this request (there may be multiple season downloads)
  const matchingDownloads =
    downloads?.filter((d) => {
      if (request?.mediaType === 'movie') {
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        return d.movieId !== null && request.mediaId !== null && d.movieId === request.mediaId
      }
      if (request?.mediaType === 'series') {
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        return d.seriesId !== null && request.mediaId !== null && d.seriesId === request.mediaId
      }
      return false
    }) ?? []
  const hasActiveDownload = matchingDownloads.length > 0

  // Aggregate stats across all downloads
  const totalSize = matchingDownloads.reduce((sum, d) => sum + (d.size || 0), 0)
  const totalDownloaded = matchingDownloads.reduce((sum, d) => sum + (d.downloadedSize || 0), 0)
  const downloadSpeed = matchingDownloads.reduce((sum, d) => sum + (d.downloadSpeed || 0), 0)
  const progress = totalSize > 0 ? (totalDownloaded / totalSize) * 100 : 0
  const remainingBytes = totalSize - totalDownloaded
  const eta = downloadSpeed > 0 ? Math.ceil(remainingBytes / downloadSpeed) : 0
  const isActive = matchingDownloads.some((d) => d.status === 'downloading')
  const isPaused =
    matchingDownloads.length > 0 && matchingDownloads.every((d) => d.status === 'paused')
  const isComplete = Math.round(progress) >= 100

  const goBack = () => {
    navigate({ to: '/requests' })
  }

  const handleCancel = () => {
    cancelMutation.mutate(requestId, {
      onSuccess: () => {
        toast.success('Request cancelled')
        setCancelDialogOpen(false)
      },
      onError: (error) => {
        toast.error('Failed to cancel request', { description: error.message })
      },
    })
  }

  const handleWatch = () => {
    if (!request) {
      return
    }

    if (request.isWatching) {
      unwatchMutation.mutate(requestId, {
        onSuccess: () => toast.success('Stopped watching request'),
        onError: (error) => toast.error('Failed to unwatch', { description: error.message }),
      })
    } else {
      watchMutation.mutate(requestId, {
        onSuccess: () => toast.success('Now watching request'),
        onError: (error) => toast.error('Failed to watch', { description: error.message }),
      })
    }
  }

  if (isLoading) {
    return (
      <div className="mx-auto max-w-4xl space-y-6 px-6 pt-6">
        <Skeleton className="h-10 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (error || !request) {
    return (
      <div className="mx-auto max-w-4xl px-6 pt-6">
        <Card>
          <CardContent className="py-12 text-center">
            <p className="text-muted-foreground">Request not found</p>
            <Button onClick={goBack} className="mt-4">
              Back to Requests
            </Button>
          </CardContent>
        </Card>
      </div>
    )
  }

  const statusConfig = STATUS_CONFIG[request.status]
  const isOwner = request.userId === user?.id
  const canCancel = isOwner && request.status === 'pending'
  const isMovie = request.mediaType === 'movie'

  return (
    <div className="mx-auto max-w-4xl space-y-6 px-6 pt-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" onClick={goBack} className="text-xs md:text-sm">
          <ArrowLeft className="mr-0.5 size-3 md:mr-1 md:size-4" />
          Back
        </Button>
        <div className="flex-1" />
        {isOwner ? (
          <Button variant="outline" disabled className="text-xs md:text-sm">
            <Eye className="mr-1 size-3 md:mr-2 md:size-4" />
            Watching
          </Button>
        ) : (
          <Button variant="outline" onClick={handleWatch} className="text-xs md:text-sm">
            {request.isWatching ? (
              <>
                <EyeOff className="mr-1 size-3 md:mr-2 md:size-4" />
                Unwatch
              </>
            ) : (
              <>
                <Eye className="mr-1 size-3 md:mr-2 md:size-4" />
                Watch
              </>
            )}
          </Button>
        )}
      </div>

      <Card className={isMovie ? 'border-movie-500/30' : 'border-tv-500/30'}>
        <CardHeader>
          <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div className="flex items-center gap-4">
              <div
                className={`h-24 w-16 flex-shrink-0 overflow-hidden rounded-lg ${isMovie ? 'glow-movie-sm' : 'glow-tv-sm'}`}
              >
                <PosterImage
                  url={request.posterUrl}
                  alt={request.title}
                  type={request.mediaType === 'movie' ? 'movie' : 'series'}
                  className="h-full w-full"
                />
              </div>
              <div className="min-w-0">
                <CardTitle className="text-xl sm:text-2xl">
                  {request.title}
                  {request.year ? (
                    <span className="text-muted-foreground ml-2">({request.year})</span>
                  ) : null}
                </CardTitle>
                <div className="mt-1 flex flex-wrap items-center gap-2">
                  <Badge
                    variant="outline"
                    className={`capitalize ${isMovie ? 'border-movie-500/50 text-movie-400' : 'border-tv-500/50 text-tv-400'}`}
                  >
                    {request.mediaType}
                  </Badge>
                  {request.seasonNumber ? (
                    <Badge variant="outline">Season {request.seasonNumber}</Badge>
                  ) : null}
                  {request.episodeNumber ? (
                    <Badge variant="outline">Episode {request.episodeNumber}</Badge>
                  ) : null}
                  <Badge
                    className={`${statusConfig.color} px-1.5 py-0.5 text-[10px] text-white sm:hidden md:px-2 md:text-xs`}
                  >
                    {statusConfig.icon}
                    <span className="ml-0.5 md:ml-1">{statusConfig.label}</span>
                  </Badge>
                </div>
              </div>
            </div>

            <Badge
              className={`${statusConfig.color} hidden shrink-0 px-2 py-0.5 text-sm text-white sm:flex md:px-3 md:py-1 md:text-base`}
            >
              {statusConfig.icon}
              <span className="ml-1 md:ml-2">{statusConfig.label}</span>
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          {hasActiveDownload ? (
            <div
              className={`space-y-2 rounded-lg px-2 py-3 ${isMovie ? 'bg-movie-500/10 border-movie-500/20 border' : 'bg-tv-500/10 border-tv-500/20 border'}`}
            >
              <div className="flex items-center justify-between text-xs md:text-sm">
                <span
                  className={`flex items-center gap-1 font-medium md:gap-2 ${isMovie ? 'text-movie-400' : 'text-tv-400'}`}
                >
                  <Download className="size-3 md:size-4" />
                  Download Progress
                </span>
                <span className="text-muted-foreground">
                  {isComplete
                    ? 'Importing'
                    : isPaused
                      ? 'Paused'
                      : isActive
                        ? formatEta(eta)
                        : 'Queued'}
                </span>
              </div>
              <Progress value={progress} variant={isMovie ? 'movie' : 'tv'} className="h-2" />
              <div className="text-muted-foreground flex justify-between text-xs">
                <span>{Math.round(progress)}%</span>
                {isComplete ? (
                  <span>--</span>
                ) : isActive && downloadSpeed > 0 ? (
                  <span>{(downloadSpeed / 1024 / 1024).toFixed(1)} MB/s</span>
                ) : null}
              </div>
            </div>
          ) : null}

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1">
              <p className="text-muted-foreground flex items-center gap-1 text-xs md:text-sm">
                <User className="size-3 md:size-4" /> Requested By
              </p>
              <p className="text-sm font-medium md:text-base">
                {request.user?.displayName || request.user?.username || 'Unknown'}
              </p>
            </div>
            <div className="space-y-1">
              <p className="text-muted-foreground flex items-center gap-1 text-xs md:text-sm">
                <Calendar className="size-3 md:size-4" /> Requested On
              </p>
              <p className="text-sm font-medium md:text-base">
                {format(new Date(request.createdAt), 'PPP')}
              </p>
            </div>
            {request.approvedAt ? (
              <div className="space-y-1">
                <p className="text-muted-foreground flex items-center gap-1 text-xs md:text-sm">
                  <CheckCircle className="size-3 md:size-4" /> Approved On
                </p>
                <p className="text-sm font-medium md:text-base">
                  {format(new Date(request.approvedAt), 'PPP')}
                </p>
              </div>
            ) : null}
            {request.deniedReason ? (
              <div className="space-y-1 sm:col-span-2">
                <p className="text-muted-foreground flex items-center gap-1 text-xs md:text-sm">
                  <XCircle className="size-3 md:size-4" /> Denied Reason
                </p>
                <p className="text-sm font-medium text-red-500 md:text-base">
                  {request.deniedReason}
                </p>
              </div>
            ) : null}
          </div>

          {request.mediaType === 'series' && (
            <div className="bg-muted rounded-lg p-4">
              <p className="text-sm">
                <span className="font-medium">Monitor Future Episodes:</span>{' '}
                {request.monitorFuture ? 'Yes' : 'No'}
              </p>
            </div>
          )}

          {canCancel ? (
            <div className="flex items-center gap-2 border-t pt-4">
              <Button
                variant="destructive"
                onClick={() => setCancelDialogOpen(true)}
                className="text-xs md:text-sm"
              >
                <Trash2 className="mr-1 size-3 md:mr-2 md:size-4" />
                Cancel Request
              </Button>
            </div>
          ) : null}
        </CardContent>
      </Card>

      <AlertDialog open={cancelDialogOpen} onOpenChange={setCancelDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Cancel Request</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to cancel the request for &quot;{request.title}&quot;? This action cannot
              be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep Request</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleCancel}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90 text-xs md:text-sm"
            >
              {cancelMutation.isPending ? (
                <Loader2 className="mr-1 size-3 animate-spin md:mr-2 md:size-4" />
              ) : null}
              Cancel Request
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
