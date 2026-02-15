import { Download } from 'lucide-react'

import { cn } from '@/lib/utils'

import type { ControlSize, MediaTheme } from './controls-types'
import { progressDimensions } from './controls-utils'

type ProgressMockupProps = {
  theme: MediaTheme
  size: ControlSize
  progress: number
  paused: boolean
  fullWidth?: boolean
}

export function ProgressMockup({ theme, size, progress, paused, fullWidth }: ProgressMockupProps) {
  const isMovie = theme === 'movie'
  const clampedProgress = Math.max(progress, 2)
  const showDetails = size !== 'xs'

  return (
    <div
      className={cn(
        'relative overflow-hidden rounded-md',
        paused && 'animation-paused',
        progressDimensions(size, fullWidth),
      )}
    >
      <div className="bg-muted/30 absolute inset-0" />
      <ProgressFill theme={theme} progress={clampedProgress} showShimmer={showDetails} />
      {showDetails ? <EdgeGlow theme={theme} progress={clampedProgress} /> : null}
      {showDetails ? <InsetGlowRing isMovie={isMovie} /> : null}
      <ProgressLabel size={size} />
    </div>
  )
}

function ProgressFill({
  theme,
  progress,
  showShimmer,
}: {
  theme: MediaTheme
  progress: number
  showShimmer: boolean
}) {
  const isMovie = theme === 'movie'
  const gradientClass = isMovie
    ? 'from-movie-600/40 via-movie-500/50 to-movie-500/60 bg-gradient-to-r'
    : 'from-tv-600/40 via-tv-500/50 to-tv-500/60 bg-gradient-to-r'
  const shimmerVia = isMovie ? 'via-movie-400/25' : 'via-tv-400/25'

  return (
    <div
      className={cn('absolute inset-y-0 left-0 transition-all duration-500 ease-out', gradientClass)}
      style={{ width: `${progress}%` }}
    >
      {showShimmer ? <div className="absolute inset-0 overflow-hidden">
          <div
            className={cn(
              'absolute inset-y-0 w-12 animate-[shimmer_1.5s_linear_infinite]',
              shimmerVia,
              'bg-gradient-to-r from-transparent to-transparent',
            )}
          />
        </div> : null}
    </div>
  )
}

function EdgeGlow({ theme, progress }: { theme: MediaTheme; progress: number }) {
  return (
    <div
      className={cn(
        'absolute top-0 bottom-0 w-1 rounded-full blur-sm transition-all duration-500',
        theme === 'movie' ? 'bg-movie-400' : 'bg-tv-400',
      )}
      style={{ left: `calc(${progress}% - 2px)` }}
    />
  )
}

function InsetGlowRing({ isMovie }: { isMovie: boolean }) {
  return (
    <div
      className={cn(
        'absolute inset-0 rounded-md ring-1 ring-inset',
        isMovie
          ? 'ring-movie-500/40 animate-[inset-glow-pulse-movie_2s_ease-in-out_infinite]'
          : 'ring-tv-500/40 animate-[inset-glow-pulse-tv_2s_ease-in-out_infinite]',
      )}
    />
  )
}

function ProgressLabel({ size }: { size: ControlSize }) {
  return (
    <div className="text-muted-foreground absolute inset-0 flex items-center justify-center gap-2 text-sm">
      <Download className={size === 'xs' ? 'size-3.5' : 'size-4'} />
      {size === 'lg' && 'Downloading'}
    </div>
  )
}
