import { useEffect, useState } from 'react'
import { Bell, Check, CheckCircle } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Button } from '@/components/ui/button'
import { useInbox, useMarkAllRead, useMarkRead, useUnreadCount } from '@/hooks/portal'
import { cn } from '@/lib/utils'
import type { PortalNotification } from '@/api/portal'

export function NotificationBell() {
  const [open, setOpen] = useState(false)

  const { data: unreadData } = useUnreadCount()
  const { data: inboxData, isLoading } = useInbox(50, 0)
  const markAllRead = useMarkAllRead()
  const markRead = useMarkRead()

  const hasUnread = (unreadData?.count ?? 0) > 0

  // When popover opens, mark all as read
  useEffect(() => {
    if (open && hasUnread) {
      markAllRead.mutate()
    }
  }, [open, hasUnread])

  const handleNotificationClick = (notification: PortalNotification) => {
    if (!notification.read) {
      markRead.mutate(notification.id)
    }
  }

  const getNotificationIcon = (type: PortalNotification['type']) => {
    switch (type) {
      case 'approved':
        return <CheckCircle className="size-3 md:size-4 text-blue-500" />
      case 'denied':
        return <span className="size-3 md:size-4 text-red-500 font-bold text-center">✕</span>
      case 'available':
        return <Check className="size-3 md:size-4 text-green-500" />
      default:
        return <Bell className="size-3 md:size-4" />
    }
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button variant="ghost" size="icon" className="relative size-8 md:size-9">
            <Bell className="size-4 md:size-5" />
            {hasUnread && (
              <span className="absolute top-1 right-1 size-2 rounded-full bg-red-500" />
            )}
          </Button>
        }
      />
      <PopoverContent align="end" className="w-80 p-0">
        <div className="px-4 py-3 border-b border-border">
          <h3 className="font-semibold text-sm">Notifications</h3>
        </div>

        <div className="max-h-96 overflow-y-auto">
          {isLoading ? (
            <div className="p-4 text-center text-muted-foreground text-sm">
              Loading...
            </div>
          ) : !inboxData || inboxData.notifications.length === 0 ? (
            <div className="p-8 text-center text-muted-foreground text-sm">
              No notifications yet
            </div>
          ) : (
            <div className="divide-y divide-border">
              {inboxData.notifications.map((notification) => (
                <button
                  key={notification.id}
                  onClick={() => handleNotificationClick(notification)}
                  className={cn(
                    'w-full text-left px-3 md:px-4 py-2 md:py-3 hover:bg-muted/50 transition-colors',
                    !notification.read && 'bg-muted/30'
                  )}
                >
                  <div className="flex gap-2 md:gap-3">
                    <div className="shrink-0 mt-0.5">
                      {getNotificationIcon(notification.type)}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className={cn(
                        'text-xs md:text-sm',
                        !notification.read && 'font-medium'
                      )}>
                        {notification.title}
                      </p>
                      <p className="text-xs md:text-sm text-muted-foreground mt-0.5 line-clamp-2">
                        {notification.message}
                      </p>
                      <p className="text-[10px] md:text-xs text-muted-foreground mt-1">
                        {formatDistanceToNow(new Date(notification.createdAt), { addSuffix: true })}
                      </p>
                    </div>
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
      </PopoverContent>
    </Popover>
  )
}
