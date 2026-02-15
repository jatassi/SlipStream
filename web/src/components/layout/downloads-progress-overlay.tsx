import { cn } from '@/lib/utils'

import { getEdgeGlowClassName, getProgressBarGradient, getShimmerClassName } from './downloads-nav-classes'
import type { DownloadTheme } from './downloads-nav-types'

type DownloadsProgressOverlayProps = {
  theme: DownloadTheme
  progress: number
  allPaused: boolean
}

export function DownloadsProgressOverlay({ theme, progress, allPaused }: DownloadsProgressOverlayProps) {
  const clampedProgress = Math.max(progress, 2)

  return (
    <div
      className={cn(
        'absolute inset-0 overflow-hidden rounded-md',
        allPaused && 'animation-paused',
      )}
    >
      <div className="bg-muted/30 absolute inset-0" />

      <div
        className={cn(
          'absolute inset-y-0 left-0 transition-all duration-500 ease-out',
          getProgressBarGradient(theme),
        )}
        style={{ width: `${clampedProgress}%` }}
      >
        <div className="absolute inset-0 overflow-hidden">
          <div className={getShimmerClassName(theme)} />
        </div>
      </div>

      <div
        className={getEdgeGlowClassName(theme)}
        style={{ left: `calc(${clampedProgress}% - 2px)` }}
      />
    </div>
  )
}
