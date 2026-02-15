import { useScheduledTasks } from '@/hooks'
import { useDevModeStore, useProgressStore, useUIStore, useWebSocketStore } from '@/stores'

export function useHeader() {
  const { notifications, dismissNotification, globalLoading, setGlobalLoading } = useUIStore()
  const {
    enabled: devModeEnabled,
    switching: devModeSwitching,
    setEnabled,
    setSwitching,
  } = useDevModeStore()
  const { send } = useWebSocketStore()
  const { data: tasks } = useScheduledTasks()
  const activities = useProgressStore((state) => state.visibleActivities)
  const activeCount = useProgressStore((state) => state.activeCount)
  const dismissActivity = useProgressStore((state) => state.dismissActivity)

  const runningTasks = tasks?.filter((t) => t.running) ?? []
  const hasRunningTasks = runningTasks.length > 0
  const hasActiveActivities = activeCount > 0

  const handleDevModeToggle = (pressed: boolean) => {
    setSwitching(true)
    setEnabled(pressed)
    send({
      type: 'devmode:set',
      payload: { enabled: pressed },
    })
  }

  return {
    notifications,
    dismissNotification,
    globalLoading,
    setGlobalLoading,
    devModeEnabled,
    devModeSwitching,
    handleDevModeToggle,
    runningTasks,
    hasRunningTasks,
    activities,
    activeCount,
    hasActiveActivities,
    dismissActivity,
  }
}
