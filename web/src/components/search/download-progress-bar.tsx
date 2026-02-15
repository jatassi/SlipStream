import { Progress } from '@/components/ui/progress'
import { usePortalDownloads } from '@/hooks'
import { formatEta } from '@/lib/formatters'

type DownloadProgressBarProps = {
  mediaId?: number
  mediaType: 'movie' | 'series'
}

export function DownloadProgressBar({ mediaId, mediaType }: DownloadProgressBarProps) {
  const { data: downloads } = usePortalDownloads()

  // Find ALL downloads matching the media (for series, there may be multiple season downloads)
  const matchingDownloads =
    downloads?.filter((d) => {
      if (mediaType === 'movie') {
        return d.movieId !== undefined && mediaId !== undefined && d.movieId === mediaId
      }
      return d.seriesId !== undefined && mediaId !== undefined && d.seriesId === mediaId
    }) ?? []

  if (matchingDownloads.length === 0) {
    return (
      <div className="w-full space-y-1">
        <Progress value={0} className="h-2" />
        <div className="text-muted-foreground flex justify-between text-xs">
          <span>0%</span>
          <span>Queued</span>
        </div>
      </div>
    )
  }

  // Aggregate stats across all downloads
  const totalSize = matchingDownloads.reduce((sum, d) => sum + (d.size || 0), 0)
  const totalDownloaded = matchingDownloads.reduce((sum, d) => sum + (d.downloadedSize || 0), 0)
  const totalSpeed = matchingDownloads.reduce((sum, d) => sum + (d.downloadSpeed || 0), 0)

  // Calculate combined progress as weighted average by size
  const progress = totalSize > 0 ? (totalDownloaded / totalSize) * 100 : 0

  // Calculate combined ETA based on remaining bytes and total speed
  const remainingBytes = totalSize - totalDownloaded
  const combinedEta = totalSpeed > 0 ? Math.ceil(remainingBytes / totalSpeed) : 0

  // Check status: if any is downloading, show as downloading; if all paused, show paused
  const hasActiveDownload = matchingDownloads.some((d) => d.status === 'downloading')
  const allPaused = matchingDownloads.every((d) => d.status === 'paused')

  const isComplete = Math.round(progress) >= 100

  let statusText = 'Queued'
  if (isComplete) {
    statusText = 'Importing'
  } else if (allPaused) {
    statusText = 'Paused'
  } else if (hasActiveDownload) {
    statusText = formatEta(combinedEta)
  }

  return (
    <div className="w-full space-y-1">
      <Progress value={progress} className="h-2" />
      <div className="text-muted-foreground flex justify-between text-xs">
        <span>{Math.round(progress)}%</span>
        <span>{statusText}</span>
      </div>
    </div>
  )
}
