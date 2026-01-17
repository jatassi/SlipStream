import { Bell, Hammer, Loader2 } from 'lucide-react'
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
import { Toggle } from '@/components/ui/toggle'
import { useUIStore, useDevModeStore, useWebSocketStore } from '@/stores'
import { Badge } from '@/components/ui/badge'
import { SearchBar } from '@/components/search/SearchBar'
import { useScheduledTasks } from '@/hooks'
import { cn } from '@/lib/utils'

export function Header() {
  const { notifications, dismissNotification } = useUIStore()
  const { enabled: devModeEnabled, switching: devModeSwitching, setEnabled, setSwitching } = useDevModeStore()
  const { send } = useWebSocketStore()
  const { data: tasks } = useScheduledTasks()

  const runningTasks = tasks?.filter(t => t.running) || []
  const hasRunningTasks = runningTasks.length > 0

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

        {/* Developer Mode Toggle */}
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger
              render={
                <Toggle
                  pressed={devModeEnabled}
                  onPressedChange={handleDevModeToggle}
                  disabled={devModeSwitching}
                  size="sm"
                  className={cn(
                    devModeEnabled && 'bg-amber-600/20 text-amber-500 hover:bg-amber-600/30 hover:text-amber-400'
                  )}
                />
              }
            >
              {devModeSwitching ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <Hammer className="size-4" />
              )}
            </TooltipTrigger>
            <TooltipContent>
              {devModeEnabled ? 'Developer mode enabled' : 'Developer mode disabled'}
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>

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
