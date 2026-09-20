import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { schedulerApi } from '@/api'

export const schedulerKeys = {
  all: ['scheduler'] as const,
  tasks: () => [...schedulerKeys.all, 'tasks'] as const,
}

export function useScheduledTasks() {
  return useQuery({
    queryKey: schedulerKeys.tasks(),
    queryFn: () => schedulerApi.listTasks(),
    refetchInterval: (query) =>
      query.state.data?.some((task) => task.running) === true ? 2000 : 60_000,
  })
}

export function useRunTask() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => schedulerApi.runTask(id),
    onSuccess: () => {
      // Invalidate tasks to refresh the list
      void queryClient.invalidateQueries({ queryKey: schedulerKeys.tasks() })
    },
  })
}
