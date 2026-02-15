import { Download } from 'lucide-react'

import { Progress } from '@/components/ui/progress'
import { formatEta } from '@/lib/formatters'
import type { PortalDownload } from '@/types'

export function DownloadStatusCard({ download }: { download: PortalDownload }) {
  const progress = download.progress
  const downloadSpeed = download.downloadSpeed
  const eta = download.eta
  const isDownloading = download.status === 'downloading'
  const isPaused = download.status === 'paused'

  return (
    <div className="space-y-2 rounded-lg border border-purple-500/20 bg-purple-500/10 p-3">
      <div className="flex items-center justify-between text-sm">
        <span className="flex items-center gap-2 font-medium text-purple-400">
          <Download className="size-4" />
          Downloading
        </span>
        <span className="text-muted-foreground text-xs">
          <DownloadEtaLabel isPaused={isPaused} isDownloading={isDownloading} eta={eta} />
        </span>
      </div>
      <Progress value={progress} className="h-2" />
      <div className="text-muted-foreground flex justify-between text-xs">
        <span>{Math.round(progress)}%</span>
        {isDownloading && downloadSpeed > 0 ? (
          <span>{(downloadSpeed / 1024 / 1024).toFixed(1)} MB/s</span>
        ) : null}
      </div>
    </div>
  )
}

function DownloadEtaLabel({ isPaused, isDownloading, eta }: { isPaused: boolean; isDownloading: boolean; eta: number }) {
  if (isPaused) {
    return <>Paused</>
  }
  if (isDownloading) {
    return <>{formatEta(eta)}</>
  }
  return <>Queued</>
}
