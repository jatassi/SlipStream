import { Play, Clock, CheckCircle, XCircle, Loader2 } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { LoadingState } from '@/components/data/LoadingState'
import { ErrorState } from '@/components/data/ErrorState'
import { useScheduledTasks, useRunTask } from '@/hooks'
import { formatDate } from '@/lib/formatters'
import { toast } from 'sonner'
import type { ScheduledTask } from '@/types'

// Convert cron expression to human-readable text
function cronToPlainEnglish(cron: string): string {
  const parts = cron.split(' ')
  if (parts.length !== 5) return cron

  const [minute, hour, dayOfMonth, month, dayOfWeek] = parts

  // Common patterns
  if (minute === '0' && hour === '0' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    return 'Daily at midnight'
  }
  if (minute === '0' && hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    return 'Every hour'
  }
  if (minute === '*/5' && hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    return 'Every 5 minutes'
  }
  if (minute === '*/15' && hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    return 'Every 15 minutes'
  }
  if (minute === '*/30' && hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    return 'Every 30 minutes'
  }
  if (minute === '0' && hour !== '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    const hourNum = parseInt(hour)
    const period = hourNum >= 12 ? 'PM' : 'AM'
    const displayHour = hourNum === 0 ? 12 : hourNum > 12 ? hourNum - 12 : hourNum
    return `Daily at ${displayHour}:00 ${period}`
  }
  if (dayOfWeek === '0' && dayOfMonth === '*' && month === '*') {
    return `Weekly on Sundays at ${hour}:${minute.padStart(2, '0')}`
  }
  if (dayOfWeek === '1' && dayOfMonth === '*' && month === '*') {
    return `Weekly on Mondays at ${hour}:${minute.padStart(2, '0')}`
  }

  return cron
}

// Format relative time for last/next run
function formatRelativeTime(dateString?: string): string {
  if (!dateString) return 'Never'

  const date = new Date(dateString)
  const now = new Date()
  const diffMs = date.getTime() - now.getTime()
  const diffMins = Math.round(diffMs / 60000)
  const diffHours = Math.round(diffMs / 3600000)
  const diffDays = Math.round(diffMs / 86400000)

  if (Math.abs(diffMins) < 1) return 'Just now'
  if (diffMins > 0) {
    if (diffMins < 60) return `in ${diffMins} min`
    if (diffHours < 24) return `in ${diffHours} hours`
    return `in ${diffDays} days`
  } else {
    if (Math.abs(diffMins) < 60) return `${Math.abs(diffMins)} min ago`
    if (Math.abs(diffHours) < 24) return `${Math.abs(diffHours)} hours ago`
    return `${Math.abs(diffDays)} days ago`
  }
}

export function TasksPage() {
  const { data: tasks, isLoading, isError, refetch } = useScheduledTasks()
  const runTaskMutation = useRunTask()

  const handleRunTask = async (taskId: string, taskName: string) => {
    try {
      await runTaskMutation.mutateAsync(taskId)
      toast.success(`Started: ${taskName}`)
    } catch {
      toast.error(`Failed to start: ${taskName}`)
    }
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Scheduled Tasks" />
        <LoadingState variant="list" />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Scheduled Tasks" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="Scheduled Tasks"
        description="Automated background tasks that run on a schedule"
      />

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
                <TableRow key={task.id}>
                  <TableCell>
                    <div>
                      <p className="font-medium">{task.name}</p>
                      <p className="text-sm text-muted-foreground">{task.description}</p>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Clock className="size-4 text-muted-foreground" />
                      <span>{cronToPlainEnglish(task.cron)}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div>
                      <p>{formatRelativeTime(task.lastRun)}</p>
                      {task.lastRun && (
                        <p className="text-xs text-muted-foreground">
                          {formatDate(task.lastRun)}
                        </p>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div>
                      <p>{formatRelativeTime(task.nextRun)}</p>
                      {task.nextRun && (
                        <p className="text-xs text-muted-foreground">
                          {formatDate(task.nextRun)}
                        </p>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    {task.running ? (
                      <Badge className="bg-blue-600 hover:bg-blue-600">
                        <Loader2 className="size-3 mr-1 animate-spin" />
                        Running
                      </Badge>
                    ) : task.lastError ? (
                      <Badge variant="destructive">
                        <XCircle className="size-3 mr-1" />
                        Failed
                      </Badge>
                    ) : task.lastRun ? (
                      <Badge className="bg-green-600 hover:bg-green-600">
                        <CheckCircle className="size-3 mr-1" />
                        Success
                      </Badge>
                    ) : (
                      <Badge variant="secondary">Pending</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleRunTask(task.id, task.name)}
                      disabled={task.running || runTaskMutation.isPending}
                    >
                      <Play className="size-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                  No scheduled tasks found
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}
