import {
  AlertCircle,
  CheckCircle2,
  Download,
  FileIcon,
  FileInput,
  FolderSearch,
  Loader2,
  RefreshCw,
  X,
  XCircle,
} from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Progress, ProgressIndicator, ProgressTrack } from '@/components/ui/progress'
import { cn } from '@/lib/utils'
import type { Activity, ActivityStatus, ActivityType } from '@/types/progress'

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

type ProgressItemProps = {
  activity: Activity
  onDismiss?: () => void
}

function ProgressBar({ isIndeterminate, progress }: { isIndeterminate: boolean; progress: number }) {
  if (isIndeterminate) {
    return (
      <div className="bg-muted h-1 w-full overflow-hidden rounded-full">
        <div className="bg-primary h-full w-1/3 animate-pulse rounded-full" />
      </div>
    )
  }

  return (
    <>
      <Progress value={progress}>
        <ProgressTrack className="h-1">
          <ProgressIndicator />
        </ProgressTrack>
      </Progress>
      {progress >= 0 && (
        <div className="text-muted-foreground mt-1 text-right text-xs">{progress}%</div>
      )}
    </>
  )
}

export function ProgressItem({ activity, onDismiss }: ProgressItemProps) {
  const ActivityIcon = activityIcons[activity.type]
  const StatusIcon = statusIcons[activity.status]
  const isActive = activity.status === 'in_progress' || activity.status === 'pending'
  const isIndeterminate = activity.progress === -1

  return (
    <div
      className={cn(
        'group bg-card relative rounded-md border p-3 transition-all',
        isActive && 'border-primary/30',
        activity.status === 'completed' && 'border-green-500/30 opacity-80',
        activity.status === 'failed' && 'border-destructive/30',
        activity.status === 'cancelled' && 'border-yellow-500/30 opacity-80',
      )}
    >
      {/* Dismiss button */}
      {onDismiss ? (
        <Button
          variant="ghost"
          size="icon-xs"
          className="absolute top-1 right-1 opacity-0 transition-opacity group-hover:opacity-100"
          onClick={onDismiss}
        >
          <X className="size-3" />
          <span className="sr-only">Dismiss</span>
        </Button>
      ) : null}

      {/* Header with icon and title */}
      <div className="flex items-start gap-2">
        <div className={cn('mt-0.5 shrink-0', statusColors[activity.status])}>
          {StatusIcon ? <StatusIcon className={cn('size-4', isActive && 'animate-spin')} /> : null}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <ActivityIcon className="text-muted-foreground size-3.5 shrink-0" />
            <span className="truncate text-sm font-medium">{activity.title}</span>
          </div>
          <p className="text-muted-foreground mt-0.5 truncate text-xs">{activity.subtitle}</p>
        </div>
      </div>

      {isActive ? <div className="mt-2">
          <ProgressBar isIndeterminate={isIndeterminate} progress={activity.progress} />
        </div> : null}
    </div>
  )
}
