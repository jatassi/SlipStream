import { Download } from 'lucide-react'

import { Progress } from '@/components/ui/progress'
import { formatEta } from '@/lib/formatters'

type RequestDownloadProgressProps = {
  isMovie: boolean
  progress: number
  downloadSpeed: number
  isActive: boolean
  isPaused: boolean
  isComplete: boolean
  eta: number
}

export function RequestDownloadProgress({
  isMovie,
  progress,
  downloadSpeed,
  isActive,
  isPaused,
  isComplete,
  eta,
}: RequestDownloadProgressProps) {
  const themeClass = isMovie
    ? 'bg-movie-500/10 border-movie-500/20 border'
    : 'bg-tv-500/10 border-tv-500/20 border'
  const textClass = isMovie ? 'text-movie-400' : 'text-tv-400'

  return (
    <div className={`space-y-2 rounded-lg px-2 py-3 ${themeClass}`}>
      <div className="flex items-center justify-between text-xs md:text-sm">
        <span className={`flex items-center gap-1 font-medium md:gap-2 ${textClass}`}>
          <Download className="size-3 md:size-4" />
          Download Progress
        </span>
        <span className="text-muted-foreground">
          <DownloadStatusLabel
            isComplete={isComplete}
            isPaused={isPaused}
            isActive={isActive}
            eta={eta}
          />
        </span>
      </div>
      <Progress value={progress} variant={isMovie ? 'movie' : 'tv'} className="h-2" />
      <div className="text-muted-foreground flex justify-between text-xs">
        <span>{Math.round(progress)}%</span>
        <SpeedLabel isComplete={isComplete} isActive={isActive} downloadSpeed={downloadSpeed} />
      </div>
    </div>
  )
}

function DownloadStatusLabel({
  isComplete,
  isPaused,
  isActive,
  eta,
}: {
  isComplete: boolean
  isPaused: boolean
  isActive: boolean
  eta: number
}) {
  if (isComplete) {
    return <>Importing</>
  }
  if (isPaused) {
    return <>Paused</>
  }
  if (isActive) {
    return <>{formatEta(eta)}</>
  }
  return <>Queued</>
}

function SpeedLabel({
  isComplete,
  isActive,
  downloadSpeed,
}: {
  isComplete: boolean
  isActive: boolean
  downloadSpeed: number
}) {
  if (isComplete) {
    return <span>--</span>
  }
  if (isActive && downloadSpeed > 0) {
    return <span>{(downloadSpeed / 1024 / 1024).toFixed(1)} MB/s</span>
  }
  return null
}
