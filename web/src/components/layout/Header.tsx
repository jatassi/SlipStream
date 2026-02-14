import { Bell, Hammer, Loader2, LayoutTemplate } from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from '@/components/ui/hover-card'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Switch } from '@/components/ui/switch'
import { useUIStore, useDevModeStore, useWebSocketStore, useProgressStore } from '@/stores'
import { Badge } from '@/components/ui/badge'
import { SearchBar } from '@/components/search/SearchBar'
import { useScheduledTasks } from '@/hooks'
import { cn } from '@/lib/utils'
import { ProgressItem } from '@/components/progress/ProgressItem'
import { ScrollArea } from '@/components/ui/scroll-area'

export function Header() {
  const { notifications, dismissNotification, globalLoading, setGlobalLoading } = useUIStore()
  const { enabled: devModeEnabled, switching: devModeSwitching, setEnabled, setSwitching } = useDevModeStore()
  const { send } = useWebSocketStore()
  const { data: tasks } = useScheduledTasks()
  const activities = useProgressStore((state) => state.visibleActivities)
  const activeCount = useProgressStore((state) => state.activeCount)
  const dismissActivity = useProgressStore((state) => state.dismissActivity)

  const runningTasks = tasks?.filter(t => t.running) || []
  const hasRunningTasks = runningTasks.length > 0
  const hasActiveActivities = activeCount > 0

  const handleDevModeToggle = (pressed: boolean) => {
    setSwitching(true)
    setEnabled(pressed)
    send({
      type: 'devmode:set',
      payload: { enabled: pressed }
    })
  }

  return (
    <header className="flex h-14 items-center gap-4 border-b border-border bg-card px-6">
      {/* Search */}
      <div className="flex-1 flex justify-center">
        <div className="flex-1 max-w-2xl">
          <SearchBar />
        </div>
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2">
        {/* Running Tasks Indicator */}
        {hasRunningTasks && (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger>
                <div className="flex items-center gap-1.5 px-2 py-1 rounded-md bg-blue-600/10 text-blue-600">
                  <Loader2 className="size-4 animate-spin" />
                  <span className="text-sm font-medium">
                    {runningTasks.length === 1 ? runningTasks[0].name : `${runningTasks.length} tasks`}
                  </span>
                </div>
              </TooltipTrigger>
              <TooltipContent>
                <div className="space-y-1">
                  {runningTasks.map(task => (
                    <p key={task.id}>{task.name}</p>
                  ))}
                </div>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )}

        {/* Activity Indicator */}
        {activities.length > 0 && (
          <HoverCard>
            <HoverCardTrigger>
              <div className={cn(
                "flex items-center gap-1.5 px-2 py-1 rounded-md cursor-default",
                hasActiveActivities ? "bg-blue-600/10 text-blue-600" : "bg-muted text-muted-foreground"
              )}>
                <Loader2 className={cn("size-4", hasActiveActivities && "animate-spin")} />
                <span className="text-sm font-medium">
                  {activeCount > 0
                    ? activeCount === 1
                      ? activities.find(a => a.status === 'in_progress' || a.status === 'pending')?.title || '1 activity'
                      : `${activeCount} activities`
                    : `${activities.length} recent`}
                </span>
              </div>
            </HoverCardTrigger>
            <HoverCardContent align="end" className="w-80 p-0">
              <div className="px-3 py-2 border-b border-border">
                <span className="text-xs font-medium text-muted-foreground">
                  Activity
                  {activeCount > 0 && (
                    <span className="ml-1.5 inline-flex size-4 items-center justify-center rounded-full bg-primary text-[10px] text-primary-foreground">
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
                  'inline-flex items-center justify-center rounded-md h-8 w-8 transition-colors',
                  'text-amber-500 hover:bg-amber-600/20'
                )}
              >
                {devModeSwitching ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <Hammer className="size-4" />
                )}
              </PopoverTrigger>
              <PopoverContent align="end" className="w-56 p-0 gap-0">
                <div className="px-3 py-2 border-b border-border">
                  <span className="text-xs font-medium text-muted-foreground">Developer Tools</span>
                </div>
                <div className="p-2 space-y-1">
                  <label className="flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm hover:bg-accent transition-colors cursor-pointer">
                    <LayoutTemplate className="size-4 shrink-0 text-muted-foreground" />
                    <span className="flex-1">Force Loading</span>
                    <Switch
                      checked={globalLoading}
                      onCheckedChange={setGlobalLoading}
                      size="sm"
                    />
                  </label>
                </div>
              </PopoverContent>
            </Popover>
          ) : (
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger className="inline-flex items-center justify-center rounded-md h-8 w-8 text-muted-foreground">
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
          <DropdownMenuTrigger className="relative inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring hover:bg-accent hover:text-accent-foreground h-9 w-9">
            <Bell className="size-5" />
            {notifications.length > 0 && (
              <Badge
                variant="destructive"
                className="absolute -right-1 -top-1 size-5 p-0 text-xs flex items-center justify-center"
              >
                {notifications.length}
              </Badge>
            )}
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-80">
            {notifications.length === 0 ? (
              <div className="p-4 text-center text-sm text-muted-foreground">
                No notifications
              </div>
            ) : (
              notifications.slice(0, 5).map((notification) => (
                <DropdownMenuItem
                  key={notification.id}
                  onClick={() => dismissNotification(notification.id)}
                  className="flex flex-col items-start gap-1 p-3"
                >
                  <span className="font-medium">{notification.title}</span>
                  {notification.message && (
                    <span className="text-sm text-muted-foreground">
                      {notification.message}
                    </span>
                  )}
                </DropdownMenuItem>
              ))
            )}
          </DropdownMenuContent>
        </DropdownMenu>

      </div>
    </header>
  )
}
