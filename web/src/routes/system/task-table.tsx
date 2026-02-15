import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import type { ScheduledTask } from '@/types'

import { TaskRow } from './task-row'

type TaskTableProps = {
  tasks: ScheduledTask[] | undefined
  isRunPending: boolean
  onRun: (taskId: string, taskName: string) => void
}

export function TaskTable({ tasks, isRunPending, onRun }: TaskTableProps) {
  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Task</TableHead>
            <TableHead>Frequency</TableHead>
            <TableHead>Last Run</TableHead>
            <TableHead>Next Run</TableHead>
            <TableHead>Status</TableHead>
            <TableHead className="w-[100px]">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {tasks && tasks.length > 0 ? (
            tasks.map((task: ScheduledTask) => (
              <TaskRow
                key={task.id}
                task={task}
                isRunPending={isRunPending}
                onRun={onRun}
              />
            ))
          ) : (
            <TableRow>
              <TableCell colSpan={6} className="text-muted-foreground py-8 text-center">
                No scheduled tasks found
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </div>
  )
}
