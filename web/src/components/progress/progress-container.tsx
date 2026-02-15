import { Activity } from 'lucide-react'

import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'
import { useProgressStore } from '@/stores/progress'

import { ProgressItem } from './progress-item'

type ProgressContainerProps = {
  collapsed?: boolean
  className?: string
}

function CollapsedIndicator({ activeCount, className }: { activeCount: number; className?: string }) {
  return (
    <div className={cn('flex justify-center py-3', className)}>
      <div className="relative">
        <Activity className={cn('size-5', activeCount > 0 && 'text-primary animate-pulse')} />
        {activeCount > 0 && (
          <span className="bg-primary text-primary-foreground absolute -top-1.5 -right-1.5 flex size-4 items-center justify-center rounded-full text-[10px] font-medium">
            {activeCount}
          </span>
        )}
      </div>
    </div>
  )
}

export function ProgressContainer({ collapsed = false, className }: ProgressContainerProps) {
  const activities = useProgressStore((state) => state.visibleActivities)
  const activeCount = useProgressStore((state) => state.activeCount)
  const dismissActivity = useProgressStore((state) => state.dismissActivity)

  if (activities.length === 0) {
    return null
  }

  if (collapsed) {
    return <CollapsedIndicator activeCount={activeCount} className={className} />
  }

  return (
    <div className={cn('flex flex-col', className)}>
      <div className="flex items-center justify-between px-3 py-2">
        <div className="flex items-center gap-2">
          <Activity className={cn('size-4', activeCount > 0 && 'text-primary')} />
          <span className="text-muted-foreground text-xs font-medium">
            Activity
            {activeCount > 0 && (
              <span className="bg-primary text-primary-foreground ml-1.5 inline-flex size-4 items-center justify-center rounded-full text-[10px]">
                {activeCount}
              </span>
            )}
          </span>
        </div>
      </div>

      <ScrollArea className="max-h-64 px-3">
        <div className="space-y-2 pb-2">
          {activities.map((activity) => (
            <ProgressItem
              key={activity.id}
              activity={activity}
              onDismiss={() => dismissActivity(activity.id)}
            />
          ))}
        </div>
      </ScrollArea>
    </div>
  )
}
