import { ErrorState } from '@/components/data/error-state'
import { Group, Row, RowSkeleton } from '@/components/grouped-list'
import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'

import { TaskRow } from './task-row'
import { useTasksPage } from './use-tasks-page'

const TITLE = 'Scheduled Tasks'

export function TasksPage() {
  const back = usePushBack()
  const { tasks, isLoading, isError, refetch, isRunPending, handleRunTask } = useTasksPage()

  if (isLoading) {
    return (
      <Screen title={TITLE} back={back}>
        <Group>
          {[0, 1, 2, 3, 4].map((index) => (
            <RowSkeleton key={index} />
          ))}
        </Group>
      </Screen>
    )
  }

  if (isError) {
    return (
      <Screen title={TITLE} back={back}>
        <ErrorState onRetry={refetch} />
      </Screen>
    )
  }

  return (
    <Screen title={TITLE} back={back}>
      <Group>
        {tasks === undefined || tasks.length === 0 ? (
          <Row title="No scheduled tasks" />
        ) : (
          tasks.map((task) => (
            <TaskRow key={task.id} task={task} isRunPending={isRunPending} onRun={handleRunTask} />
          ))
        )}
      </Group>
    </Screen>
  )
}
