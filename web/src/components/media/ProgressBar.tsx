import { cn } from '@/lib/utils'
import { Progress as ProgressPrimitive } from '@base-ui/react/progress'

interface ProgressBarProps {
  value: number
  max?: number
  showLabel?: boolean
  size?: 'sm' | 'md' | 'lg'
  variant?: 'default' | 'movie' | 'tv'
  className?: string
}

export function ProgressBar({
  value,
  max = 100,
  showLabel = false,
  size = 'md',
  variant = 'default',
  className,
}: ProgressBarProps) {
  const percentage = Math.min((value / max) * 100, 100)

  const sizeClasses = {
    sm: 'h-1',
    md: 'h-2',
    lg: 'h-3',
  }

  const indicatorClasses = {
    default: 'bg-primary',
    movie: 'bg-movie-500',
    tv: 'bg-tv-500',
  }

  return (
    <div className={cn('flex items-center gap-2', className)}>
      <ProgressPrimitive.Root
        value={percentage}
        className="flex flex-wrap gap-3 flex-1"
      >
        <ProgressPrimitive.Track
          className={cn(
            'bg-muted rounded-full relative flex w-full items-center overflow-x-hidden',
            sizeClasses[size]
          )}
        >
          <ProgressPrimitive.Indicator
            className={cn('h-full transition-all rounded-full', indicatorClasses[variant])}
          />
        </ProgressPrimitive.Track>
      </ProgressPrimitive.Root>
      {showLabel && (
        <span className="text-xs text-muted-foreground tabular-nums">
          {percentage.toFixed(1)}%
        </span>
      )}
    </div>
  )
}
