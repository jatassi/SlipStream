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

// Parse Go duration string (e.g., "6h0m0s", "15m0s", "1h30m0s")
function parseGoDuration(duration: string): { hours: number; minutes: number; seconds: number } | null {
  const match = duration.match(/^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$/)
  if (!match) return null
  return {
    hours: parseInt(match[1] || '0'),
    minutes: parseInt(match[2] || '0'),
    seconds: parseInt(match[3] || '0'),
  }
}

// Format time with AM/PM
function formatTime(hour: number, minute: number): string {
  if (hour === 0 && minute === 0) return 'midnight'
  if (hour === 12 && minute === 0) return 'noon'
  const period = hour >= 12 ? 'PM' : 'AM'
  const displayHour = hour === 0 ? 12 : hour > 12 ? hour - 12 : hour
  const displayMinute = minute.toString().padStart(2, '0')
  return `${displayHour}:${displayMinute} ${period}`
}

// Convert cron expression to human-readable text
function cronToPlainEnglish(cron: string): string {
  // Handle @every directive (Go duration format)
  if (cron.startsWith('@every ')) {
    const duration = parseGoDuration(cron.slice(7))
    if (duration) {
      const { hours, minutes } = duration
      if (hours > 0 && minutes === 0) {
        return hours === 1 ? 'Every hour' : `Every ${hours} hours`
      }
      if (hours === 0 && minutes > 0) {
        return minutes === 1 ? 'Every minute' : `Every ${minutes} minutes`
      }
      if (hours > 0 && minutes > 0) {
        return `Every ${hours}h ${minutes}m`
      }
    }
    return cron
  }

  const parts = cron.split(' ')
  if (parts.length !== 5) return cron

  const [minute, hour, dayOfMonth, month, dayOfWeek] = parts
  const minuteNum = parseInt(minute)
  const hourNum = parseInt(hour)

  // Every X minutes patterns (*/X * * * *)
  if (minute.startsWith('*/') && hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    const interval = parseInt(minute.slice(2))
    if (!isNaN(interval)) {
      return interval === 1 ? 'Every minute' : `Every ${interval} minutes`
    }
  }

  // Every X hours patterns (0 */X * * *)
  if (minute === '0' && hour.startsWith('*/') && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    const interval = parseInt(hour.slice(2))
    if (!isNaN(interval)) {
      return interval === 1 ? 'Every hour' : `Every ${interval} hours`
    }
  }

  // Every hour (minute is fixed, hour is wildcard)
  if (!isNaN(minuteNum) && hour === '*' && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    return minuteNum === 0 ? 'Every hour' : `Every hour at :${minute.padStart(2, '0')}`
  }

  // Daily at specific time
  if (!isNaN(minuteNum) && !isNaN(hourNum) && dayOfMonth === '*' && month === '*' && dayOfWeek === '*') {
    return `Daily at ${formatTime(hourNum, minuteNum)}`
  }

  // Weekly patterns
  const dayNames = ['Sundays', 'Mondays', 'Tuesdays', 'Wednesdays', 'Thursdays', 'Fridays', 'Saturdays']
  if (dayOfMonth === '*' && month === '*' && dayOfWeek !== '*') {
    const dayNum = parseInt(dayOfWeek)
    if (!isNaN(dayNum) && dayNum >= 0 && dayNum <= 6 && !isNaN(hourNum) && !isNaN(minuteNum)) {
      return `Weekly on ${dayNames[dayNum]} at ${formatTime(hourNum, minuteNum)}`
    }
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
