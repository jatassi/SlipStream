import { X, Loader2, CheckCircle2, XCircle, AlertCircle, FolderSearch, Download, FileInput, RefreshCw, FileIcon } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Progress, ProgressTrack, ProgressIndicator } from '@/components/ui/progress'
import { Button } from '@/components/ui/button'
import type { Activity, ActivityType, ActivityStatus } from '@/types/progress'

interface ProgressItemProps {
  activity: Activity
  onDismiss?: () => void
}

const activityIcons: Record<ActivityType, React.ElementType> = {
  scan: FolderSearch,
  download: Download,
  import: FileInput,
  'metadata-refresh': RefreshCw,
  'file-operation': FileIcon,
}

const statusIcons: Record<ActivityStatus, React.ElementType | null> = {
  pending: Loader2,
  in_progress: Loader2,
  completed: CheckCircle2,
  failed: XCircle,
  cancelled: AlertCircle,
}

const statusColors: Record<ActivityStatus, string> = {
  pending: 'text-muted-foreground',
  in_progress: 'text-primary',
  completed: 'text-green-500',
  failed: 'text-destructive',
  cancelled: 'text-yellow-500',
}

export function ProgressItem({ activity, onDismiss }: ProgressItemProps) {
  const ActivityIcon = activityIcons[activity.type] || FileIcon
  const StatusIcon = statusIcons[activity.status]
  const isActive = activity.status === 'in_progress' || activity.status === 'pending'
  const isIndeterminate = activity.progress === -1

  return (
    <div
      className={cn(
        'group relative rounded-md border bg-card p-3 transition-all',
        isActive && 'border-primary/30',
        activity.status === 'completed' && 'border-green-500/30 opacity-80',
        activity.status === 'failed' && 'border-destructive/30',
        activity.status === 'cancelled' && 'border-yellow-500/30 opacity-80'
      )}
    >
      {/* Dismiss button */}
      {onDismiss && (
        <Button
          variant="ghost"
          size="icon-xs"
          className="absolute right-1 top-1 opacity-0 transition-opacity group-hover:opacity-100"
          onClick={onDismiss}
        >
          <X className="size-3" />
          <span className="sr-only">Dismiss</span>
        </Button>
      )}

      {/* Header with icon and title */}
      <div className="flex items-start gap-2">
        <div className={cn('mt-0.5 shrink-0', statusColors[activity.status])}>
          {StatusIcon && (
            <StatusIcon
              className={cn(
                'size-4',
                isActive && 'animate-spin'
              )}
            />
          )}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <ActivityIcon className="size-3.5 shrink-0 text-muted-foreground" />
            <span className="truncate text-sm font-medium">{activity.title}</span>
          </div>
          <p className="mt-0.5 truncate text-xs text-muted-foreground">
            {activity.subtitle}
          </p>
        </div>
      </div>

      {/* Progress bar */}
      {isActive && (
        <div className="mt-2">
          {isIndeterminate ? (
            <div className="h-1 w-full overflow-hidden rounded-full bg-muted">
              <div className="h-full w-1/3 animate-pulse rounded-full bg-primary" />
            </div>
          ) : (
            <Progress value={activity.progress}>
              <ProgressTrack className="h-1">
                <ProgressIndicator />
              </ProgressTrack>
            </Progress>
          )}
          {!isIndeterminate && activity.progress >= 0 && (
            <div className="mt-1 text-right text-xs text-muted-foreground">
              {activity.progress}%
            </div>
          )}
        </div>
      )}
    </div>
  )
}
