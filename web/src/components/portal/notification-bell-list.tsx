import { formatDistanceToNow } from 'date-fns'
import { Bell, Check, CheckCircle } from 'lucide-react'

import type { PortalNotification } from '@/api/portal'
import { cn } from '@/lib/utils'

function getNotificationIcon(type: PortalNotification['type']) {
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

type NotificationBellListProps = {
  notifications: PortalNotification[]
  onNotificationClick: (notification: PortalNotification) => void
}

export function NotificationBellList({
  notifications,
  onNotificationClick,
}: NotificationBellListProps) {
  return (
    <div className="divide-border divide-y">
      {notifications.map((notification) => (
        <button
          key={notification.id}
          onClick={() => onNotificationClick(notification)}
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
  )
}
