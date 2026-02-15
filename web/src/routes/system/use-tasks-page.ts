import { toast } from 'sonner'

import { useGlobalLoading, useRunTask, useScheduledTasks } from '@/hooks'

export function useTasksPage() {
  const globalLoading = useGlobalLoading()
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
