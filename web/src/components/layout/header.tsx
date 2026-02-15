import { HeaderActivityIndicator } from '@/components/layout/header-activity-indicator'
import { HeaderDevMode } from '@/components/layout/header-dev-mode'
import { HeaderNotifications } from '@/components/layout/header-notifications'
import { HeaderRunningTasks } from '@/components/layout/header-running-tasks'
import { useHeader } from '@/components/layout/use-header'
import { SearchBar } from '@/components/search/search-bar'

export function Header() {
  const {
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
  } = useHeader()

  return (
    <header className="border-border bg-card flex h-14 items-center gap-4 border-b px-6">
      <div className="flex flex-1 justify-center">
        <div className="max-w-2xl flex-1">
          <SearchBar />
        </div>
      </div>

      <div className="flex items-center gap-2">
        {hasRunningTasks ? <HeaderRunningTasks tasks={runningTasks} /> : null}

        {activities.length > 0 ? (
          <HeaderActivityIndicator
            activities={activities}
            activeCount={activeCount}
            hasActiveActivities={hasActiveActivities}
            onDismiss={dismissActivity}
          />
        ) : null}

        <HeaderDevMode
          devModeEnabled={devModeEnabled}
          devModeSwitching={devModeSwitching}
          onToggle={handleDevModeToggle}
          globalLoading={globalLoading}
          onGlobalLoadingChange={setGlobalLoading}
        />

        <HeaderNotifications notifications={notifications} onDismiss={dismissNotification} />
      </div>
    </header>
  )
}
