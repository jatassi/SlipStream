import { Clock, Play } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { TableCell, TableRow } from '@/components/ui/table'
import { formatDate } from '@/lib/formatters'
import type { ScheduledTask } from '@/types'

import { TaskStatusBadge } from './task-status-badge'
import { cronToPlainEnglish, formatRelativeTime } from './task-utils'

type TaskRowProps = {
  task: ScheduledTask
  isRunPending: boolean
  onRun: (taskId: string, taskName: string) => void
}

export function TaskRow({ task, isRunPending, onRun }: TaskRowProps) {
  return (
    <TableRow>
      <TableCell>
        <div>
          <p className="font-medium">{task.name}</p>
          <p className="text-muted-foreground text-sm">{task.description}</p>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-2">
          <Clock className="text-muted-foreground size-4" />
          <span>{cronToPlainEnglish(task.cron)}</span>
        </div>
      </TableCell>
      <TableCell>
        <div>
          <p>{formatRelativeTime(task.lastRun)}</p>
          {task.lastRun ? (
            <p className="text-muted-foreground text-xs">{formatDate(task.lastRun)}</p>
          ) : null}
        </div>
      </TableCell>
      <TableCell>
        <div>
          <p>{formatRelativeTime(task.nextRun)}</p>
          {task.nextRun ? (
            <p className="text-muted-foreground text-xs">{formatDate(task.nextRun)}</p>
          ) : null}
        </div>
      </TableCell>
      <TableCell>
        <TaskStatusBadge task={task} />
      </TableCell>
      <TableCell>
        <Button
          variant="outline"
          size="sm"
          onClick={() => onRun(task.id, task.name)}
          disabled={task.running || isRunPending}
        >
          <Play className="size-4" />
        </Button>
      </TableCell>
    </TableRow>
  )
}
