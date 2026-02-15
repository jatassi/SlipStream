import { useState } from 'react'

import { useNavigate, useParams } from '@tanstack/react-router'
import { toast } from 'sonner'

import {
  useCancelRequest,
  useGlobalLoading,
  usePortalDownloads,
  useRequest,
  useUnwatchRequest,
  useWatchRequest,
} from '@/hooks'
import { usePortalAuthStore } from '@/stores'
import type { PortalDownload, Request } from '@/types'

function getMatchingDownloads(
  downloads: PortalDownload[] | undefined,
  request: Request | undefined,
) {
  if (!downloads || request?.mediaId === undefined || request.mediaId === null) {
    return []
  }
  return downloads.filter((d) => {
    if (request.mediaType === 'movie') {
      return d.movieId !== undefined && d.movieId === request.mediaId
    }
    if (request.mediaType === 'series') {
      return d.seriesId !== undefined && d.seriesId === request.mediaId
    }
    return false
  })
}

function computeDownloadState(downloads: PortalDownload[]) {
  const totalSize = downloads.reduce((sum, d) => sum + (d.size || 0), 0)
  const totalDownloaded = downloads.reduce((sum, d) => sum + (d.downloadedSize || 0), 0)
  const speed = downloads.reduce((sum, d) => sum + (d.downloadSpeed || 0), 0)
  const progress = totalSize > 0 ? (totalDownloaded / totalSize) * 100 : 0
  const remainingBytes = totalSize - totalDownloaded

  return {
    hasActive: downloads.length > 0,
    progress,
    speed,
    isActive: downloads.some((d) => d.status === 'downloading'),
    isPaused: downloads.length > 0 && downloads.every((d) => d.status === 'paused'),
    isComplete: Math.round(progress) >= 100,
    eta: speed > 0 ? Math.ceil(remainingBytes / speed) : 0,
  }
}

export function useRequestDetail() {
  const params: { id?: string } = useParams({ strict: false })
  const navigate = useNavigate()
  const requestId = Number.parseInt(params.id ?? '0', 10)
  const { user } = usePortalAuthStore()
  const globalLoading = useGlobalLoading()
  const { data: request, isLoading: queryLoading, error } = useRequest(requestId)
  const { data: downloads } = usePortalDownloads()
  const cancelMutation = useCancelRequest()
  const watchMutation = useWatchRequest()
  const unwatchMutation = useUnwatchRequest()
  const [cancelDialogOpen, setCancelDialogOpen] = useState(false)

  const handleCancel = () => cancelMutation.mutate(requestId, {
    onSuccess: () => { toast.success('Request cancelled'); setCancelDialogOpen(false) },
    onError: (err) => { toast.error('Failed to cancel request', { description: err.message }) },
  })

  const handleWatch = () => {
    if (!request) { return }
    const [mutation, ok, fail] = request.isWatching
      ? [unwatchMutation, 'Stopped watching request', 'Failed to unwatch'] as const
      : [watchMutation, 'Now watching request', 'Failed to watch'] as const
    mutation.mutate(requestId, {
      onSuccess: () => toast.success(ok),
      onError: (err) => toast.error(fail, { description: err.message }),
    })
  }

  const isOwner = request ? request.userId === user?.id : false
  return {
    request, isLoading: queryLoading || globalLoading, error,
    isMovie: request?.mediaType === 'movie', isOwner,
    canCancel: isOwner && request?.status === 'pending',
    cancelDialogOpen, setCancelDialogOpen,
    cancelPending: cancelMutation.isPending,
    download: computeDownloadState(getMatchingDownloads(downloads, request)),
    goBack: () => { void navigate({ to: '/requests' }) },
    handleCancel, handleWatch,
  }
}
