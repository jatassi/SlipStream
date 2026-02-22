import { formatEta } from '@/lib/formatters'
import type { PortalDownload, Request } from '@/types'

type DownloadProgress = {
  progress: number
  statusLabel: string
}

export function findMatchingDownloads(
  downloads: PortalDownload[],
  request: Request,
): PortalDownload[] {
  return downloads.filter((d) => {
    if (request.mediaType === 'movie') {
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
      return d.movieId !== null && request.mediaId !== null && d.movieId === request.mediaId
    }
    if (request.mediaType === 'series') {
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
      return d.seriesId !== null && request.mediaId !== null && d.seriesId === request.mediaId
    }
    return false
  })
}

export function computeDownloadProgress(
  matchingDownloads: PortalDownload[],
): DownloadProgress | null {
  if (matchingDownloads.length === 0) {
    return null
  }

  const totalSize = matchingDownloads.reduce((sum, d) => sum + (d.size || 0), 0)
  const totalDownloaded = matchingDownloads.reduce((sum, d) => sum + (d.downloadedSize || 0), 0)
  const totalSpeed = matchingDownloads.reduce((sum, d) => sum + (d.downloadSpeed || 0), 0)
  const progress = totalSize > 0 ? (totalDownloaded / totalSize) * 100 : 0
  const remainingBytes = totalSize - totalDownloaded
  const eta = totalSpeed > 0 ? Math.ceil(remainingBytes / totalSpeed) : 0
  const isActive = matchingDownloads.some((d) => d.status === 'downloading')
  const isPaused = matchingDownloads.every((d) => d.status === 'paused')
  const isComplete = Math.round(progress) >= 100

  const statusLabel = resolveDownloadStatus({ isComplete, isPaused, isActive, eta })

  return { progress, statusLabel }
}

type DownloadStatusParams = {
  isComplete: boolean
  isPaused: boolean
  isActive: boolean
  eta: number
}

function resolveDownloadStatus({
  isComplete,
  isPaused,
  isActive,
  eta,
}: DownloadStatusParams): string {
  if (isComplete) {
    return 'Importing'
  }
  if (isPaused) {
    return 'Paused'
  }
  if (isActive) {
    return formatEta(eta)
  }
  return 'Queued'
}
