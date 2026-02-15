import { ProgressBar } from '@/components/media/progress-bar'
import { formatEta, formatSpeed } from '@/lib/formatters'
import type { QueueItem } from '@/types'

type DownloadRowProgressProps = {
  item: QueueItem
  isMovie: boolean
  isSeries: boolean
  progressText: string
}

function getProgressVariant(isMovie: boolean, isSeries: boolean): 'movie' | 'tv' | undefined {
  if (isMovie) {
    return 'movie'
  }
  if (isSeries) {
    return 'tv'
  }
  return undefined
}

export function DownloadRowProgress({
  item,
  isMovie,
  isSeries,
  progressText,
}: DownloadRowProgressProps) {
  const isDownloading = item.status === 'downloading'

  return (
    <div className="min-w-[200px] flex-1 basis-56 self-center">
      <div className="relative py-2">
        <ProgressBar
          value={item.progress}
          size="sm"
          variant={getProgressVariant(isMovie, isSeries)}
        />
        <div className="text-muted-foreground absolute right-0 left-0 mt-1 flex items-center text-xs">
          <span>{progressText}</span>
          <span className="mx-auto">{isDownloading ? formatSpeed(item.downloadSpeed) : ''}</span>
          <span>{isDownloading ? formatEta(item.eta) : ''}</span>
        </div>
      </div>
    </div>
  )
}
