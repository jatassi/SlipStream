import { Bell, Hammer, LayoutTemplate, Loader2 } from 'lucide-react'

import { ProgressItem } from '@/components/progress/ProgressItem'
import { SearchBar } from '@/components/search/SearchBar'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Switch } from '@/components/ui/switch'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { useScheduledTasks } from '@/hooks'
import { cn } from '@/lib/utils'
import { useDevModeStore, useProgressStore, useUIStore, useWebSocketStore } from '@/stores'

export function Header() {
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

  const runningTasks = tasks?.filter((t) => t.running) || []
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

  return (
    <header className="border-border bg-card flex h-14 items-center gap-4 border-b px-6">
      {/* Search */}
      <div className="flex flex-1 justify-center">
        <div className="max-w-2xl flex-1">
          <SearchBar />
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2">
        {/* Running Tasks Indicator */}
        {hasRunningTasks ? (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger>
                <div className="flex items-center gap-1.5 rounded-md bg-blue-600/10 px-2 py-1 text-blue-600">
                  <Loader2 className="size-4 animate-spin" />
                  <span className="text-sm font-medium">
                    {runningTasks.length === 1
                      ? runningTasks[0].name
                      : `${runningTasks.length} tasks`}
                  </span>
                </div>
              </TooltipTrigger>
              <TooltipContent>
                <div className="space-y-1">
                  {runningTasks.map((task) => (
                    <p key={task.id}>{task.name}</p>
                  ))}
                </div>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        ) : null}

        {/* Activity Indicator */}
        {activities.length > 0 && (
          <HoverCard>
            <HoverCardTrigger>
              <div
                className={cn(
                  'flex cursor-default items-center gap-1.5 rounded-md px-2 py-1',
                  hasActiveActivities
                    ? 'bg-blue-600/10 text-blue-600'
                    : 'bg-muted text-muted-foreground',
                )}
              >
                <Loader2 className={cn('size-4', hasActiveActivities && 'animate-spin')} />
                <span className="text-sm font-medium">
                  {activeCount > 0
                    ? activeCount === 1
                      ? activities.find((a) => a.status === 'in_progress' || a.status === 'pending')
                          ?.title || '1 activity'
                      : `${activeCount} activities`
                    : `${activities.length} recent`}
                </span>
              </div>
            </HoverCardTrigger>
            <HoverCardContent align="end" className="w-80 p-0">
              <div className="border-border border-b px-3 py-2">
                <span className="text-muted-foreground text-xs font-medium">
                  Activity
                  {activeCount > 0 && (
                    <span className="bg-primary text-primary-foreground ml-1.5 inline-flex size-4 items-center justify-center rounded-full text-[10px]">
                      {activeCount}
                    </span>
                  )}
                </span>
              </div>
              <ScrollArea className="max-h-64">
                <div className="space-y-2 p-3">
                  {activities.map((activity) => (
                    <ProgressItem
                      key={activity.id}
                      activity={activity}
                      onDismiss={() => dismissActivity(activity.id)}
                    />
                  ))}
                </div>
              </ScrollArea>
            </HoverCardContent>
          </HoverCard>
        )}

        {/* Developer Mode */}
        <div className="flex items-center gap-1.5">
          {devModeEnabled ? (
            <Popover>
              <PopoverTrigger
                className={cn(
                  'inline-flex h-8 w-8 items-center justify-center rounded-md transition-colors',
                  'text-amber-500 hover:bg-amber-600/20',
                )}
              >
                {devModeSwitching ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <Hammer className="size-4" />
                )}
              </PopoverTrigger>
              <PopoverContent align="end" className="w-56 gap-0 p-0">
                <div className="border-border border-b px-3 py-2">
                  <span className="text-muted-foreground text-xs font-medium">Developer Tools</span>
                </div>
                <div className="space-y-1 p-2">
                  <label className="hover:bg-accent flex cursor-pointer items-center gap-2.5 rounded-md px-2 py-1.5 text-sm transition-colors">
                    <LayoutTemplate className="text-muted-foreground size-4 shrink-0" />
                    <span className="flex-1">Force Loading</span>
                    <Switch checked={globalLoading} onCheckedChange={setGlobalLoading} size="sm" />
                  </label>
                </div>
              </PopoverContent>
            </Popover>
          ) : (
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger className="text-muted-foreground inline-flex h-8 w-8 items-center justify-center rounded-md">
                  <Hammer className="size-4" />
                </TooltipTrigger>
                <TooltipContent>Enable developer mode</TooltipContent>
              </Tooltip>
            </TooltipProvider>
          )}
          <Switch
            checked={devModeEnabled}
            onCheckedChange={handleDevModeToggle}
            disabled={devModeSwitching}
            size="sm"
            className={cn(devModeEnabled && 'data-checked:bg-amber-500')}
          />
        </div>

        {/* Notifications */}
        <DropdownMenu>
          <DropdownMenuTrigger className="focus-visible:ring-ring hover:bg-accent hover:text-accent-foreground relative inline-flex h-9 w-9 items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:ring-1 focus-visible:outline-none">
            <Bell className="size-5" />
            {notifications.length > 0 && (
              <Badge
                variant="destructive"
                className="absolute -top-1 -right-1 flex size-5 items-center justify-center p-0 text-xs"
              >
                {notifications.length}
              </Badge>
            )}
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-80">
            {notifications.length === 0 ? (
              <div className="text-muted-foreground p-4 text-center text-sm">No notifications</div>
            ) : (
              notifications.slice(0, 5).map((notification) => (
                <DropdownMenuItem
                  key={notification.id}
                  onClick={() => dismissNotification(notification.id)}
                  className="flex flex-col items-start gap-1 p-3"
                >
                  <span className="font-medium">{notification.title}</span>
                  {notification.message ? (
                    <span className="text-muted-foreground text-sm">{notification.message}</span>
                  ) : null}
                </DropdownMenuItem>
              ))
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  )
}
