import { useEffect, useState } from 'react'

import type { PortalNotification } from '@/api/portal'
import { useInbox, useMarkAllRead, useMarkRead, useUnreadCount } from '@/hooks/portal'

export function useNotificationBell() {
  const [open, setOpen] = useState(false)

  const { data: unreadData } = useUnreadCount()
  const { data: inboxData, isLoading } = useInbox(50, 0)
  const markAllRead = useMarkAllRead()
  const markRead = useMarkRead()
  const markAllReadMutate = markAllRead.mutate

  const hasUnread = (unreadData?.count ?? 0) > 0

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

  return {
    open,
    setOpen,
    hasUnread,
    inboxData,
    isLoading,
    handleNotificationClick,
  }
}
