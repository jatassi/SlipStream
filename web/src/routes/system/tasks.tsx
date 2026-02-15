import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { PageHeader } from '@/components/layout/page-header'

import { TaskTable } from './task-table'
import { useTasksPage } from './use-tasks-page'

export function TasksPage() {
  const { tasks, isLoading, isError, refetch, isRunPending, handleRunTask } =
    useTasksPage()

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
      <TaskTable tasks={tasks} isRunPending={isRunPending} onRun={handleRunTask} />
    </div>
  )
}
