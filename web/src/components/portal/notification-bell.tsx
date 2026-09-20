import { Bell } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

import { NotificationBellList } from './notification-bell-list'
import { useNotificationBell } from './use-notification-bell'

export function NotificationBell() {
  const { open, setOpen, hasUnread, inboxData, isLoading, handleNotificationClick } =
    useNotificationBell()

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button variant="ghost" size="icon" aria-label="Notifications" className="relative size-8 md:size-9">
            <Bell className="size-4 md:size-5" />
            {/* eslint-disable-next-line react/jsx-no-leaked-render -- condition is already a boolean */}
            {hasUnread && <span className="absolute top-1 right-1 size-2 rounded-full bg-red-500" />}
          </Button>
        }
      />
      <PopoverContent align="end" className="w-80 p-0">
        <div className="border-border border-b px-4 py-3">
          <h3 className="text-sm font-semibold">Notifications</h3>
        </div>

        <div className="max-h-96 overflow-y-auto">
          {/* eslint-disable-next-line react/jsx-no-leaked-render -- condition is already a boolean */}
          {isLoading && <div className="text-muted-foreground p-4 text-center text-sm">Loading...</div>}
          {!isLoading && (!inboxData || inboxData.notifications.length === 0) && (
            <div className="text-muted-foreground p-8 text-center text-sm">
              No notifications yet
            </div>
          )}
          {!isLoading && inboxData !== undefined && inboxData.notifications.length > 0 && <NotificationBellList
              notifications={inboxData.notifications}
              onNotificationClick={handleNotificationClick}
            />}
        </div>
      </PopoverContent>
    </Popover>
  )
}
