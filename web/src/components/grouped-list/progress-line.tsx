import { cn } from '@/lib/utils'

export type ProgressKind = 'movie' | 'series'

type ProgressLineProps = {
  value: number
  kind: ProgressKind
  className?: string
  height?: string
  muted?: boolean
}

const KIND_BG: Record<ProgressKind, string> = {
  movie: 'bg-movie-500',
  series: 'bg-tv-500',
}

export function ProgressLine({
  value,
  kind,
  className,
  height = 'h-1',
  muted = false,
}: ProgressLineProps) {
  const clamped = Math.min(100, Math.max(0, value))
  return (
    <div
      className={cn('w-full overflow-hidden rounded-full bg-foreground/10', height, className)}
      role="progressbar"
      aria-valuenow={Math.round(clamped)}
      aria-valuemin={0}
      aria-valuemax={100}
    >
      <div
        className={cn(
          'h-full rounded-full transition-[width] duration-700 ease-linear',
          muted ? 'bg-muted-foreground' : KIND_BG[kind],
        )}
        style={{ width: `${clamped}%` }}
      />
    </div>
  )
}
