import { ProgressBar } from '@/components/media/progress-bar'
import { formatEta, formatSpeed } from '@/lib/formatters'
import type { QueueItem } from '@/types'

const MEDIA_TYPE_VARIANT: Record<string, string> = {
  movie: 'movie',
  series: 'tv',
}

type DownloadRowProgressProps = {
  item: QueueItem
  mediaType: string
  progressText: string
}

export function DownloadRowProgress({
  item,
  mediaType,
  progressText,
}: DownloadRowProgressProps) {
  const isDownloading = item.status === 'downloading'

  return (
    <div className="min-w-[200px] flex-1 basis-56 self-center">
      <div className="relative py-2">
        <ProgressBar
          value={item.progress}
          size="sm"
          variant={MEDIA_TYPE_VARIANT[mediaType]}
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
