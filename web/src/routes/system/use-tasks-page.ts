import { toast } from 'sonner'

import { useRunTask, useScheduledTasks } from '@/hooks'
import { useUIStore } from '@/stores'

export function useTasksPage() {
  const globalLoading = useUIStore((s) => s.globalLoading)
  const { data: tasks, isLoading: queryLoading, isError, refetch } = useScheduledTasks()
  const isLoading = queryLoading || globalLoading
  const runTaskMutation = useRunTask()

  const handleRunTask = async (taskId: string, taskName: string) => {
    try {
      await runTaskMutation.mutateAsync(taskId)
      toast.success(`Started: ${taskName}`)
    } catch {
      toast.error(`Failed to start: ${taskName}`)
    }
  }

  return {
    tasks,
    isLoading,
    isError,
    refetch,
    isRunPending: runTaskMutation.isPending,
    handleRunTask,
  }
}
