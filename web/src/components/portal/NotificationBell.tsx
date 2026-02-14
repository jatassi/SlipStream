import { useEffect, useState } from 'react'

import { formatDistanceToNow } from 'date-fns'
import { Bell, Check, CheckCircle } from 'lucide-react'

import type { PortalNotification } from '@/api/portal'
import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { useInbox, useMarkAllRead, useMarkRead, useUnreadCount } from '@/hooks/portal'
import { cn } from '@/lib/utils'

export function NotificationBell() {
  const [open, setOpen] = useState(false)

  const { data: unreadData } = useUnreadCount()
  const { data: inboxData, isLoading } = useInbox(50, 0)
  const markAllRead = useMarkAllRead()
  const markRead = useMarkRead()
  const markAllReadMutate = markAllRead.mutate

  const hasUnread = (unreadData?.count ?? 0) > 0

  // When popover opens, mark all as read
  useEffect(() => {
    if (open && hasUnread) {
      markAllReadMutate()
    }
  }, [open, hasUnread, markAllReadMutate])

  const handleNotificationClick = (notification: PortalNotification) => {
    if (!notification.read) {
      markRead.mutate(notification.id)
    }
  }

  const getNotificationIcon = (type: PortalNotification['type']) => {
    switch (type) {
      case 'approved': {
        return <CheckCircle className="size-3 text-blue-500 md:size-4" />
      }
      case 'denied': {
        return <span className="size-3 text-center font-bold text-red-500 md:size-4">✕</span>
      }
      case 'available': {
        return <Check className="size-3 text-green-500 md:size-4" />
      }
      default: {
        return <Bell className="size-3 md:size-4" />
      }
    }
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button variant="ghost" size="icon" className="relative size-8 md:size-9">
            <Bell className="size-4 md:size-5" />
            {hasUnread ? (
              <span className="absolute top-1 right-1 size-2 rounded-full bg-red-500" />
            ) : null}
          </Button>
        }
      />
      <PopoverContent align="end" className="w-80 p-0">
        <div className="border-border border-b px-4 py-3">
          <h3 className="text-sm font-semibold">Notifications</h3>
        </div>

        <div className="max-h-96 overflow-y-auto">
          {isLoading ? (
            <div className="text-muted-foreground p-4 text-center text-sm">Loading...</div>
          ) : !inboxData || inboxData.notifications.length === 0 ? (
            <div className="text-muted-foreground p-8 text-center text-sm">
              No notifications yet
            </div>
          ) : (
            <div className="divide-border divide-y">
              {inboxData.notifications.map((notification) => (
                <button
                  key={notification.id}
                  onClick={() => handleNotificationClick(notification)}
                  className={cn(
                    'hover:bg-muted/50 w-full px-3 py-2 text-left transition-colors md:px-4 md:py-3',
                    !notification.read && 'bg-muted/30',
                  )}
                >
                  <div className="flex gap-2 md:gap-3">
                    <div className="mt-0.5 shrink-0">{getNotificationIcon(notification.type)}</div>
                    <div className="min-w-0 flex-1">
                      <p className={cn('text-xs md:text-sm', !notification.read && 'font-medium')}>
                        {notification.title}
                      </p>
                      <p className="text-muted-foreground mt-0.5 line-clamp-2 text-xs md:text-sm">
                        {notification.message}
                      </p>
                      <p className="text-muted-foreground mt-1 text-[10px] md:text-xs">
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
