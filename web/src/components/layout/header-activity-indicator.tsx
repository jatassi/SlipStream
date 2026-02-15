import { Loader2 } from 'lucide-react'

import { ProgressItem } from '@/components/progress/progress-item'
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'
import type { Activity } from '@/types/progress'

type HeaderActivityIndicatorProps = {
  activities: Activity[]
  activeCount: number
  hasActiveActivities: boolean
  onDismiss: (id: string) => void
}

function getActivityLabel(activeCount: number, activities: Activity[]): string {
  if (activeCount <= 0) {
    return `${activities.length} recent`
  }
  if (activeCount === 1) {
    return (
      activities.find((a) => a.status === 'in_progress' || a.status === 'pending')?.title ??
      '1 activity'
    )
  }
  return `${activeCount} activities`
}

export function HeaderActivityIndicator({
  activities,
  activeCount,
  hasActiveActivities,
  onDismiss,
}: HeaderActivityIndicatorProps) {
  return (
    <HoverCard>
      <HoverCardTrigger>
        <div
          className={cn(
            'flex cursor-default items-center gap-1.5 rounded-md px-2 py-1',
            hasActiveActivities
              ? 'bg-blue-600/10 text-blue-600'
              : 'bg-muted text-muted-foreground',
          )}
        >
          <Loader2 className={cn('size-4', hasActiveActivities && 'animate-spin')} />
          <span className="text-sm font-medium">{getActivityLabel(activeCount, activities)}</span>
        </div>
      </HoverCardTrigger>
      <HoverCardContent align="end" className="w-80 p-0">
        <div className="border-border border-b px-3 py-2">
          <span className="text-muted-foreground text-xs font-medium">
            Activity
            {activeCount > 0 && (
              <span className="bg-primary text-primary-foreground ml-1.5 inline-flex size-4 items-center justify-center rounded-full text-[10px]">
                {activeCount}
              </span>
            )}
          </span>
        </div>
        <ScrollArea className="max-h-64">
          <div className="space-y-2 p-3">
            {activities.map((activity) => (
              <ProgressItem
                key={activity.id}
                activity={activity}
                onDismiss={() => onDismiss(activity.id)}
              />
            ))}
          </div>
        </ScrollArea>
      </HoverCardContent>
    </HoverCard>
  )
}
