import { useParams, useNavigate } from '@tanstack/react-router'
import {
  ArrowLeft,
  Clock,
  CheckCircle,
  XCircle,
  Download,
  Eye,
  EyeOff,
  Trash2,
  Loader2,
  User,
  Calendar,
} from 'lucide-react'
import { PosterImage } from '@/components/media/PosterImage'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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
import { Skeleton } from '@/components/ui/skeleton'
import { useRequest, useCancelRequest, useWatchRequest, useUnwatchRequest, usePortalDownloads } from '@/hooks'
import { usePortalAuthStore } from '@/stores'
import { Progress } from '@/components/ui/progress'
import { formatEta } from '@/lib/formatters'
import type { RequestStatus } from '@/types'
import { toast } from 'sonner'
import { format } from 'date-fns'
import { useState } from 'react'

const STATUS_CONFIG: Record<RequestStatus, { label: string; icon: React.ReactNode; color: string }> = {
  pending: { label: 'Pending Approval', icon: <Clock className="size-4 md:size-5" />, color: 'bg-yellow-500' },
  approved: { label: 'Approved', icon: <CheckCircle className="size-4 md:size-5" />, color: 'bg-blue-500' },
  denied: { label: 'Denied', icon: <XCircle className="size-4 md:size-5" />, color: 'bg-red-500' },
  downloading: { label: 'Downloading', icon: <Download className="size-4 md:size-5" />, color: 'bg-purple-500' },
  available: { label: 'Available', icon: <CheckCircle className="size-4 md:size-5" />, color: 'bg-green-500' },
  cancelled: { label: 'Cancelled', icon: <XCircle className="size-4 md:size-5" />, color: 'bg-gray-500' },
}

export function RequestDetailPage() {
  const params = useParams({ strict: false })
  const navigate = useNavigate()
  const requestId = parseInt(params.id || '0', 10)

  const { user } = usePortalAuthStore()
  const { data: request, isLoading, error } = useRequest(requestId)
  const { data: downloads } = usePortalDownloads()
  const cancelMutation = useCancelRequest()
  const watchMutation = useWatchRequest()
  const unwatchMutation = useUnwatchRequest()

  const [cancelDialogOpen, setCancelDialogOpen] = useState(false)

  // Find ALL active downloads for this request (there may be multiple season downloads)
  const matchingDownloads = downloads?.filter(d => {
    if (request?.mediaType === 'movie') return d.movieId != null && request.mediaId != null && d.movieId === request.mediaId
    if (request?.mediaType === 'series') return d.seriesId != null && request.mediaId != null && d.seriesId === request.mediaId
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
  const isActive = matchingDownloads.some(d => d.status === 'downloading')
  const isPaused = matchingDownloads.length > 0 && matchingDownloads.every(d => d.status === 'paused')
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
    if (!request) return

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
      <div className="max-w-4xl mx-auto pt-6 px-6 space-y-6">
        <Skeleton className="h-10 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (error || !request) {
    return (
      <div className="max-w-4xl mx-auto pt-6 px-6">
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
    <div className="max-w-4xl mx-auto pt-6 px-6 space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" onClick={goBack} className="text-xs md:text-sm">
          <ArrowLeft className="size-3 md:size-4 mr-0.5 md:mr-1" />
          Back
        </Button>
        <div className="flex-1" />
        {isOwner ? (
          <Button variant="outline" disabled className="text-xs md:text-sm">
            <Eye className="size-3 md:size-4 mr-1 md:mr-2" />
            Watching
          </Button>
        ) : (
          <Button variant="outline" onClick={handleWatch} className="text-xs md:text-sm">
            {request.isWatching ? (
              <>
                <EyeOff className="size-3 md:size-4 mr-1 md:mr-2" />
                Unwatch
              </>
            ) : (
              <>
                <Eye className="size-3 md:size-4 mr-1 md:mr-2" />
                Watch
              </>
            )}
          </Button>
        )}
      </div>

      <Card className={isMovie ? 'border-movie-500/30' : 'border-tv-500/30'}>
        <CardHeader>
          <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className={`w-16 h-24 rounded-lg overflow-hidden flex-shrink-0 ${isMovie ? 'glow-movie-sm' : 'glow-tv-sm'}`}>
                <PosterImage
                  url={request.posterUrl}
                  alt={request.title}
                  type={request.mediaType === 'movie' ? 'movie' : 'series'}
                  className="w-full h-full"
                />
              </div>
              <div className="min-w-0">
                <CardTitle className="text-xl sm:text-2xl">
                  {request.title}
                  {request.year && <span className="text-muted-foreground ml-2">({request.year})</span>}
                </CardTitle>
                <div className="flex items-center gap-2 mt-1 flex-wrap">
                  <Badge variant="outline" className={`capitalize ${isMovie ? 'border-movie-500/50 text-movie-400' : 'border-tv-500/50 text-tv-400'}`}>{request.mediaType}</Badge>
                  {request.seasonNumber && (
                    <Badge variant="outline">Season {request.seasonNumber}</Badge>
                  )}
                  {request.episodeNumber && (
                    <Badge variant="outline">Episode {request.episodeNumber}</Badge>
                  )}
                  <Badge className={`${statusConfig.color} text-white sm:hidden text-[10px] md:text-xs px-1.5 md:px-2 py-0.5`}>
                    {statusConfig.icon}
                    <span className="ml-0.5 md:ml-1">{statusConfig.label}</span>
                  </Badge>
                </div>
              </div>
            </div>

            <Badge className={`${statusConfig.color} text-white text-sm md:text-base px-2 md:px-3 py-0.5 md:py-1 hidden sm:flex shrink-0`}>
              {statusConfig.icon}
              <span className="ml-1 md:ml-2">{statusConfig.label}</span>
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          {hasActiveDownload && (
            <div className={`px-2 py-3 rounded-lg space-y-2 ${isMovie ? 'bg-movie-500/10 border border-movie-500/20' : 'bg-tv-500/10 border border-tv-500/20'}`}>
              <div className="flex items-center justify-between text-xs md:text-sm">
                <span className={`font-medium flex items-center gap-1 md:gap-2 ${isMovie ? 'text-movie-400' : 'text-tv-400'}`}>
                  <Download className="size-3 md:size-4" />
                  Download Progress
                </span>
                <span className="text-muted-foreground">
                  {isComplete ? 'Importing' : isPaused ? 'Paused' : isActive ? formatEta(eta) : 'Queued'}
                </span>
              </div>
              <Progress value={progress} variant={isMovie ? 'movie' : 'tv'} className="h-2" />
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>{Math.round(progress)}%</span>
                {isComplete ? (
                  <span>--</span>
                ) : isActive && downloadSpeed > 0 ? (
                  <span>{(downloadSpeed / 1024 / 1024).toFixed(1)} MB/s</span>
                ) : null}
              </div>
            </div>
          )}

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1">
              <p className="text-xs md:text-sm text-muted-foreground flex items-center gap-1">
                <User className="size-3 md:size-4" /> Requested By
              </p>
              <p className="font-medium text-sm md:text-base">{request.user?.displayName || request.user?.username || 'Unknown'}</p>
            </div>
            <div className="space-y-1">
              <p className="text-xs md:text-sm text-muted-foreground flex items-center gap-1">
                <Calendar className="size-3 md:size-4" /> Requested On
              </p>
              <p className="font-medium text-sm md:text-base">{format(new Date(request.createdAt), 'PPP')}</p>
            </div>
            {request.approvedAt && (
              <div className="space-y-1">
                <p className="text-xs md:text-sm text-muted-foreground flex items-center gap-1">
                  <CheckCircle className="size-3 md:size-4" /> Approved On
                </p>
                <p className="font-medium text-sm md:text-base">{format(new Date(request.approvedAt), 'PPP')}</p>
              </div>
            )}
            {request.deniedReason && (
              <div className="space-y-1 sm:col-span-2">
                <p className="text-xs md:text-sm text-muted-foreground flex items-center gap-1">
                  <XCircle className="size-3 md:size-4" /> Denied Reason
                </p>
                <p className="font-medium text-red-500 text-sm md:text-base">{request.deniedReason}</p>
              </div>
            )}
          </div>

          {request.mediaType === 'series' && (
            <div className="p-4 rounded-lg bg-muted">
              <p className="text-sm">
                <span className="font-medium">Monitor Future Episodes:</span>{' '}
                {request.monitorFuture ? 'Yes' : 'No'}
              </p>
            </div>
          )}

          {canCancel && (
            <div className="flex items-center gap-2 pt-4 border-t">
              <Button variant="destructive" onClick={() => setCancelDialogOpen(true)} className="text-xs md:text-sm">
                <Trash2 className="size-3 md:size-4 mr-1 md:mr-2" />
                Cancel Request
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      <AlertDialog open={cancelDialogOpen} onOpenChange={setCancelDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Cancel Request</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to cancel the request for "{request.title}"? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep Request</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleCancel}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90 text-xs md:text-sm"
            >
              {cancelMutation.isPending && <Loader2 className="size-3 md:size-4 mr-1 md:mr-2 animate-spin" />}
              Cancel Request
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
