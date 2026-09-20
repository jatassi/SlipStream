import { Loader2, Play } from 'lucide-react'

import { Row } from '@/components/grouped-list'
import type { ScheduledTask } from '@/types'

import { formatRelativeTime } from './task-utils'

type TaskRowProps = {
  task: ScheduledTask
  isRunPending: boolean
  onRun: (taskId: string, taskName: string) => void
}

function TaskTrailing({ task, isRunPending, onRun }: TaskRowProps) {
  if (task.running) {
    return (
      <span className="text-footnote flex items-center gap-1.5 text-tv-400" role="status" aria-label="Running">
        <Loader2 className="size-4 animate-spin" />
        Running
      </span>
    )
  }
  return (
    <div className="flex items-center gap-2">
      {task.lastError === undefined || task.lastError === '' ? null : (
        <span className="text-footnote text-destructive">Failed</span>
      )}
      <button
        type="button"
        aria-label={`Run ${task.name}`}
        disabled={isRunPending}
        onClick={() => onRun(task.id, task.name)}
        className="press flex size-tap items-center justify-center rounded-md text-tv-400 focus-visible:ring-ring outline-none focus-visible:ring-[3px] disabled:opacity-50"
      >
        <Play className="size-4" />
      </button>
    </div>
  )
}

export function TaskRow({ task, isRunPending, onRun }: TaskRowProps) {
  return (
    <Row
      title={task.name}
      subtitle={
        <span className="nums">
          Last {formatRelativeTime(task.lastRun)} · Next {formatRelativeTime(task.nextRun)}
        </span>
      }
      trailing={<TaskTrailing task={task} isRunPending={isRunPending} onRun={onRun} />}
    />
  )
}
