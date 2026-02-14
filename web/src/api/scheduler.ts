import type { ScheduledTask } from '@/types/scheduler'

import { apiFetch } from './client'

export const schedulerApi = {
  listTasks: () => apiFetch<ScheduledTask[]>('/scheduler/tasks'),

  getTask: (id: string) => apiFetch<ScheduledTask>(`/scheduler/tasks/${id}`),

  runTask: (id: string) =>
    apiFetch<{ message: string; taskId: string }>(`/scheduler/tasks/${id}/run`, {
      method: 'POST',
    }),
}
