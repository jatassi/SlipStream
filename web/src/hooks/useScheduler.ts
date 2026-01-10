import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { schedulerApi } from '@/api'

export const schedulerKeys = {
  all: ['scheduler'] as const,
  tasks: () => [...schedulerKeys.all, 'tasks'] as const,
  task: (id: string) => [...schedulerKeys.all, 'task', id] as const,
}

export function useScheduledTasks() {
  return useQuery({
    queryKey: schedulerKeys.tasks(),
    queryFn: () => schedulerApi.listTasks(),
    refetchInterval: 30000, // Refresh every 30 seconds to update running status
  })
}

export function useScheduledTask(id: string) {
  return useQuery({
    queryKey: schedulerKeys.task(id),
    queryFn: () => schedulerApi.getTask(id),
    enabled: !!id,
  })
}

export function useRunTask() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => schedulerApi.runTask(id),
    onSuccess: () => {
      // Invalidate tasks to refresh the list
      queryClient.invalidateQueries({ queryKey: schedulerKeys.tasks() })
    },
  })
}
