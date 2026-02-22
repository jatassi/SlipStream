import { format } from 'date-fns'
import { Calendar, CheckCircle, Loader2, Trash2, User, XCircle } from 'lucide-react'

import { PosterImage } from '@/components/media/poster-image'
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
import type { Request } from '@/types'

import { RequestDownloadProgress } from './request-download-progress'
import type { StatusConfigEntry } from './request-status-config'

type DownloadState = {
  hasActive: boolean
  progress: number
  speed: number
  isActive: boolean
  isPaused: boolean
  isComplete: boolean
  eta: number
}

type RequestDetailCardProps = {
  request: Request
  isMovie: boolean
  statusConfig: StatusConfigEntry
  canCancel: boolean
  cancelDialogOpen: boolean
  setCancelDialogOpen: (open: boolean) => void
  cancelPending: boolean
  download: DownloadState
  onCancel: () => void
}

export function RequestDetailCard({
  request,
  isMovie,
  statusConfig,
  canCancel,
  cancelDialogOpen,
  setCancelDialogOpen,
  cancelPending,
  download,
  onCancel,
}: RequestDetailCardProps) {
  return (
    <>
      <Card className={isMovie ? 'border-movie-500/30' : 'border-tv-500/30'}>
        <CardHeader>
          <CardHeaderContent request={request} isMovie={isMovie} statusConfig={statusConfig} />
        </CardHeader>
        <CardBodyContent
          request={request}
          isMovie={isMovie}
          download={download}
          canCancel={canCancel}
          onOpenCancelDialog={() => setCancelDialogOpen(true)}
        />
      </Card>

      <CancelRequestDialog
        open={cancelDialogOpen}
        onOpenChange={setCancelDialogOpen}
        title={request.title}
        isPending={cancelPending}
        onConfirm={onCancel}
      />
    </>
  )
}

function CardHeaderContent({
  request,
  isMovie,
  statusConfig,
}: {
  request: Request
  isMovie: boolean
  statusConfig: StatusConfigEntry
}) {
  return (
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
        <MediaTitleBadges request={request} isMovie={isMovie} statusConfig={statusConfig} />
      </div>

      <Badge
        className={`${statusConfig.color} hidden shrink-0 px-2 py-0.5 text-sm text-white sm:flex md:px-3 md:py-1 md:text-base`}
      >
        {statusConfig.icon}
        <span className="ml-1 md:ml-2">{statusConfig.label}</span>
      </Badge>
    </div>
  )
}

function MediaTitleBadges({
  request,
  isMovie,
  statusConfig,
}: {
  request: Request
  isMovie: boolean
  statusConfig: StatusConfigEntry
}) {
  return (
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
  )
}

function CardBodyContent({
  request,
  isMovie,
  download,
  canCancel,
  onOpenCancelDialog,
}: {
  request: Request
  isMovie: boolean
  download: DownloadState
  canCancel: boolean
  onOpenCancelDialog: () => void
}) {
  return (
    <CardContent className="space-y-6">
      {download.hasActive ? (
        <RequestDownloadProgress
          isMovie={isMovie}
          progress={download.progress}
          downloadSpeed={download.speed}
          isActive={download.isActive}
          isPaused={download.isPaused}
          isComplete={download.isComplete}
          eta={download.eta}
        />
      ) : null}

      <RequestMetadataGrid request={request} />

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
            onClick={onOpenCancelDialog}
            className="text-xs md:text-sm"
          >
            <Trash2 className="mr-1 size-3 md:mr-2 md:size-4" />
            Cancel Request
          </Button>
        </div>
      ) : null}
    </CardContent>
  )
}

function RequestMetadataGrid({ request }: { request: Request }) {
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <div className="space-y-1">
        <p className="text-muted-foreground flex items-center gap-1 text-xs md:text-sm">
          <User className="size-3 md:size-4" /> Requested By
        </p>
        <p className="text-sm font-medium md:text-base">
          {request.user?.displayName ?? request.user?.username ?? 'Unknown'}
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
          <p className="text-sm font-medium text-red-500 md:text-base">{request.deniedReason}</p>
        </div>
      ) : null}
    </div>
  )
}

function CancelRequestDialog({
  open,
  onOpenChange,
  title,
  isPending,
  onConfirm,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  isPending: boolean
  onConfirm: () => void
}) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Cancel Request</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to cancel the request for &quot;{title}&quot;? This action cannot
            be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Keep Request</AlertDialogCancel>
          <AlertDialogAction
            onClick={onConfirm}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90 text-xs md:text-sm"
          >
            {isPending ? (
              <Loader2 className="mr-1 size-3 animate-spin md:mr-2 md:size-4" />
            ) : null}
            Cancel Request
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
