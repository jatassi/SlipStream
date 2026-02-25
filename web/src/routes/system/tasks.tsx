import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { PageHeader } from '@/components/layout/page-header'

import { SystemNav } from './system-nav'
import { TaskTable } from './task-table'
import { useTasksPage } from './use-tasks-page'

const PAGE_TITLE = 'System'
const PAGE_DESCRIPTION = 'Monitor system health, tasks, logs, and updates'

export function TasksPage() {
  const { tasks, isLoading, isError, refetch, isRunPending, handleRunTask } =
    useTasksPage()

  if (isLoading) {
    return (
      <div className="space-y-6">
        <PageHeader title={PAGE_TITLE} description={PAGE_DESCRIPTION} />
        <SystemNav />
        <LoadingState variant="list" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="space-y-6">
        <PageHeader title={PAGE_TITLE} description={PAGE_DESCRIPTION} />
        <SystemNav />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader title={PAGE_TITLE} description={PAGE_DESCRIPTION} />
      <SystemNav />
      <TaskTable tasks={tasks} isRunPending={isRunPending} onRun={handleRunTask} />
    </div>
  )
}
