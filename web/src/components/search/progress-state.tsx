import { Download } from 'lucide-react'

import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { formatBytes, formatEta, formatSpeed } from '@/lib/formatters'
import { cn } from '@/lib/utils'

import type { ControlVariant, MediaTheme } from './media-search-monitor-types'

const SHAPE: Record<ControlVariant, string> = {
  pill: 'h-tap rounded-full',
  row: 'h-8 rounded-md',
}

type ProgressStateProps = {
  variant: ControlVariant
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
  variant,
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
        <ProgressBar variant={variant} theme={theme} progress={progress} isPaused={isPaused} eta={eta} />
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
  variant: ControlVariant
  theme: MediaTheme
  progress: number
  isPaused: boolean
  eta: number
}

function ProgressBar({ variant, theme, progress, isPaused, eta }: BarProps) {
  const clampedProgress = Math.max(progress, 2)

  return (
    <div
      className={cn(
        'relative w-full overflow-hidden',
        isPaused && 'animation-paused',
        SHAPE[variant],
      )}
    >
      <div className="bg-muted/30 absolute inset-0" />
      <ProgressFill theme={theme} clampedProgress={clampedProgress} showShimmer />
      <EdgeGlow theme={theme} clampedProgress={clampedProgress} />
      <InsetGlow theme={theme} shape={SHAPE[variant]} />
      <ProgressLabel variant={variant} eta={eta} />
    </div>
  )
}

function ProgressFill({ theme, clampedProgress, showShimmer }: { theme: MediaTheme; clampedProgress: number; showShimmer: boolean }) {
  return (
    <div
      className={cn(
        'absolute inset-y-0 left-0 transition-[width] duration-700 ease-linear',
        theme === 'movie'
          ? 'from-movie-600/40 via-movie-500/50 to-movie-500/60 bg-gradient-to-r'
          : 'from-tv-600/40 via-tv-500/50 to-tv-500/60 bg-gradient-to-r',
      )}
      style={{ width: `${clampedProgress}%` }}
    >
      {showShimmer ? <div className="absolute inset-0 overflow-hidden">
          <div
            className={cn(
              'absolute inset-y-0 w-12 animate-[shimmer_1.5s_linear_infinite]',
              theme === 'movie'
                ? 'via-movie-400/25 bg-gradient-to-r from-transparent to-transparent'
                : 'via-tv-400/25 bg-gradient-to-r from-transparent to-transparent',
            )}
          />
        </div> : null}
    </div>
  )
}

function EdgeGlow({ theme, clampedProgress }: { theme: MediaTheme; clampedProgress: number }) {
  return (
    <div
      className={cn(
        'absolute top-0 bottom-0 w-1 rounded-full blur-sm transition-[left] duration-700 ease-linear',
        theme === 'movie' ? 'bg-movie-400' : 'bg-tv-400',
      )}
      style={{ left: `calc(${clampedProgress}% - 2px)` }}
    />
  )
}

function InsetGlow({ theme, shape }: { theme: MediaTheme; shape: string }) {
  return (
    <div
      className={cn(
        'absolute inset-0 ring-1 ring-inset',
        shape,
        theme === 'movie'
          ? 'ring-movie-500/40 animate-[inset-glow-pulse-movie_2s_ease-in-out_infinite]'
          : 'ring-tv-500/40 animate-[inset-glow-pulse-tv_2s_ease-in-out_infinite]',
      )}
    />
  )
}

function ProgressLabel({ variant, eta }: { variant: ControlVariant; eta: number }) {
  return (
    <div className="text-muted-foreground text-footnote absolute inset-0 flex items-center justify-center gap-2">
      <Download className="size-4" />
      {variant === 'pill' && `Downloading${eta > 0 ? ` (${formatEta(eta)})` : ''}`}
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
