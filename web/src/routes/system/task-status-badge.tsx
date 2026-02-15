import { CheckCircle, Loader2, XCircle } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import type { ScheduledTask } from '@/types'

export function TaskStatusBadge({ task }: { task: ScheduledTask }) {
  if (task.running) {
    return (
      <Badge className="bg-blue-600 hover:bg-blue-600">
        <Loader2 className="mr-1 size-3 animate-spin" />
        Running
      </Badge>
    )
  }

  if (task.lastError) {
    return (
      <Badge variant="destructive">
        <XCircle className="mr-1 size-3" />
        Failed
      </Badge>
    )
  }

  if (task.lastRun) {
    return (
      <Badge className="bg-green-600 hover:bg-green-600">
        <CheckCircle className="mr-1 size-3" />
        Success
      </Badge>
    )
  }

  return <Badge variant="secondary">Pending</Badge>
}
