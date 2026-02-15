import { Loader2 } from 'lucide-react'

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import type { ScheduledTask } from '@/types'

type HeaderRunningTasksProps = {
  tasks: ScheduledTask[]
}

export function HeaderRunningTasks({ tasks }: HeaderRunningTasksProps) {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>
          <div className="flex items-center gap-1.5 rounded-md bg-blue-600/10 px-2 py-1 text-blue-600">
            <Loader2 className="size-4 animate-spin" />
            <span className="text-sm font-medium">
              {tasks.length === 1 ? tasks[0].name : `${tasks.length} tasks`}
            </span>
          </div>
        </TooltipTrigger>
        <TooltipContent>
          <div className="space-y-1">
            {tasks.map((task) => (
              <p key={task.id}>{task.name}</p>
            ))}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
