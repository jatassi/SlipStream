import { Download } from 'lucide-react'

import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { formatBytes, formatEta, formatSpeed } from '@/lib/formatters'
import { cn } from '@/lib/utils'

import type { MediaTheme, ResolvedSize } from './media-search-monitor-types'

const HEIGHT_BY_SIZE: Record<ResolvedSize, string> = {
  xs: 'h-6',
  sm: 'h-8',
  lg: 'h-9',
}

type ProgressStateProps = {
  size: ResolvedSize
  theme: MediaTheme
  progress: number
  isPaused: boolean
  releaseName: string
  speed: number
  eta: number
  downloadedSize: number
  totalSize: number
}

export function ProgressState({
  size,
  theme,
  progress,
  isPaused,
  releaseName,
  speed,
  eta,
  downloadedSize,
  totalSize,
}: ProgressStateProps) {
  return (
    <Tooltip>
      <TooltipTrigger render={<div className="w-full" />}>
        <ProgressBar size={size} theme={theme} progress={progress} isPaused={isPaused} eta={eta} />
      </TooltipTrigger>
      <TooltipContent>
        <ProgressTooltip
          releaseName={releaseName}
          progress={progress}
          downloadedSize={downloadedSize}
          totalSize={totalSize}
          isPaused={isPaused}
          speed={speed}
          eta={eta}
        />
      </TooltipContent>
    </Tooltip>
  )
}

type BarProps = {
  size: ResolvedSize
  theme: MediaTheme
  progress: number
  isPaused: boolean
  eta: number
}

function ProgressBar({ size, theme, progress, isPaused, eta }: BarProps) {
  const clampedProgress = Math.max(progress, 2)
  const showEffects = size !== 'xs'

  return (
    <div
      className={cn(
        'relative w-full overflow-hidden rounded-md',
        isPaused && 'animation-paused',
        HEIGHT_BY_SIZE[size],
      )}
    >
      <div className="bg-muted/30 absolute inset-0" />
      <ProgressFill theme={theme} clampedProgress={clampedProgress} showShimmer={showEffects} />
      {showEffects ? <EdgeGlow theme={theme} clampedProgress={clampedProgress} /> : null}
      {showEffects ? <InsetGlow theme={theme} /> : null}
      <ProgressLabel size={size} eta={eta} />
    </div>
  )
}

function ProgressFill({ theme, clampedProgress, showShimmer }: { theme: MediaTheme; clampedProgress: number; showShimmer: boolean }) {
  return (
    <div
      className={cn(
        'absolute inset-y-0 left-0 transition-all duration-500 ease-out',
        theme === 'movie'
          ? 'from-movie-600/40 via-movie-500/50 to-movie-500/60 bg-gradient-to-r'
          : 'from-tv-600/40 via-tv-500/50 to-tv-500/60 bg-gradient-to-r',
      )}
      style={{ width: `${clampedProgress}%` }}
    >
      {showShimmer ? (
        <div className="absolute inset-0 overflow-hidden">
          <div
            className={cn(
              'absolute inset-y-0 w-12 animate-[shimmer_1.5s_linear_infinite]',
              theme === 'movie'
                ? 'via-movie-400/25 bg-gradient-to-r from-transparent to-transparent'
                : 'via-tv-400/25 bg-gradient-to-r from-transparent to-transparent',
            )}
          />
        </div>
      ) : null}
    </div>
  )
}

function EdgeGlow({ theme, clampedProgress }: { theme: MediaTheme; clampedProgress: number }) {
  return (
    <div
      className={cn(
        'absolute top-0 bottom-0 w-1 rounded-full blur-sm transition-all duration-500',
        theme === 'movie' ? 'bg-movie-400' : 'bg-tv-400',
      )}
      style={{ left: `calc(${clampedProgress}% - 2px)` }}
    />
  )
}

function InsetGlow({ theme }: { theme: MediaTheme }) {
  return (
    <div
      className={cn(
        'absolute inset-0 rounded-md ring-1 ring-inset',
        theme === 'movie'
          ? 'ring-movie-500/40 animate-[inset-glow-pulse-movie_2s_ease-in-out_infinite]'
          : 'ring-tv-500/40 animate-[inset-glow-pulse-tv_2s_ease-in-out_infinite]',
      )}
    />
  )
}

function ProgressLabel({ size, eta }: { size: ResolvedSize; eta: number }) {
  const iconSize = size === 'xs' ? 'size-3.5' : 'size-4'
  return (
    <div className="text-muted-foreground absolute inset-0 flex items-center justify-center gap-2 text-sm">
      <Download className={iconSize} />
      {size === 'lg' && `Downloading${eta > 0 ? ` (${formatEta(eta)})` : ''}`}
    </div>
  )
}

function ProgressTooltip({
  releaseName,
  progress,
  downloadedSize,
  totalSize,
  isPaused,
  speed,
  eta,
}: {
  releaseName: string
  progress: number
  downloadedSize: number
  totalSize: number
  isPaused: boolean
  speed: number
  eta: number
}) {
  return (
    <div className="space-y-1 text-xs">
      {releaseName ? <p className="max-w-64 truncate font-medium">{releaseName}</p> : null}
      <p>
        {progress.toFixed(1)}% — {formatBytes(downloadedSize)} / {formatBytes(totalSize)}
      </p>
      {!isPaused && speed > 0 && (
        <p>
          {formatSpeed(speed)} — ETA: {formatEta(eta)}
        </p>
      )}
      {isPaused ? <p className="text-amber-400">Paused</p> : null}
    </div>
  )
}
