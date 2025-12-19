import { cn } from '@/lib/utils'
import { Progress } from '@/components/ui/progress'

interface ProgressBarProps {
  value: number
  max?: number
  showLabel?: boolean
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

export function ProgressBar({
  value,
  max = 100,
  showLabel = false,
  size = 'md',
  className,
}: ProgressBarProps) {
  const percentage = Math.min((value / max) * 100, 100)

  const sizeClasses = {
    sm: 'h-1',
    md: 'h-2',
    lg: 'h-3',
  }

  return (
    <div className={cn('flex items-center gap-2', className)}>
      <Progress
        value={percentage}
        className={cn(sizeClasses[size], 'flex-1')}
      />
      {showLabel && (
        <span className="text-xs text-muted-foreground tabular-nums">
          {percentage.toFixed(1)}%
        </span>
      )}
    </div>
  )
}
