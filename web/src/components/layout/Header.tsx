import { Bell } from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useUIStore } from '@/stores'
import { Badge } from '@/components/ui/badge'
import { SearchBar } from '@/components/search/SearchBar'

export function Header() {
  const { notifications, dismissNotification } = useUIStore()

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
