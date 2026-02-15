import { Bell } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

type Notification = {
  id: string
  title: string
  message?: string
}

type HeaderNotificationsProps = {
  notifications: Notification[]
  onDismiss: (id: string) => void
}

export function HeaderNotifications({ notifications, onDismiss }: HeaderNotificationsProps) {
  return (
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
              onClick={() => onDismiss(notification.id)}
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
  )
}
