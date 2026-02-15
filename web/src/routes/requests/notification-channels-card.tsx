import { Bell, Loader2, Plus } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import type { UserNotification } from '@/types'

import { NotificationChannelRow } from './notification-channel-row'

type NotificationChannelsCardProps = {
  notifications: UserNotification[]
  isLoading: boolean
  isTestPending: boolean
  getTypeName: (type: string) => string
  onCreate: () => void
  onEdit: (notification: UserNotification) => void
  onDelete: (id: number) => void
  onTest: (id: number) => void
  onToggleEnabled: (notification: UserNotification, enabled: boolean) => void
}

export function NotificationChannelsCard({
  notifications,
  isLoading,
  isTestPending,
  getTypeName,
  onCreate,
  onEdit,
  onDelete,
  onTest,
  onToggleEnabled,
}: NotificationChannelsCardProps) {
  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>Notification Channels</CardTitle>
            <CardDescription>Get notified when your requests become available</CardDescription>
          </div>
          <Button onClick={onCreate} className="text-xs md:text-sm">
            <Plus className="mr-1 size-3 md:mr-2 md:size-4" />
            Add Channel
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <NotificationChannelsList
          notifications={notifications}
          isLoading={isLoading}
          isTestPending={isTestPending}
          getTypeName={getTypeName}
          onCreate={onCreate}
          onEdit={onEdit}
          onDelete={onDelete}
          onTest={onTest}
          onToggleEnabled={onToggleEnabled}
        />
      </CardContent>
    </Card>
  )
}

type NotificationChannelsListProps = Omit<NotificationChannelsCardProps, never>

function NotificationChannelsList({
  notifications,
  isLoading,
  isTestPending,
  getTypeName,
  onCreate,
  onEdit,
  onDelete,
  onTest,
  onToggleEnabled,
}: NotificationChannelsListProps) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="text-muted-foreground size-6 animate-spin" />
      </div>
    )
  }

  if (notifications.length === 0) {
    return (
      <EmptyState
        icon={<Bell className="size-8" />}
        title="No notification channels"
        description="Add a notification channel to get notified when your requests become available"
        action={{ label: 'Add Channel', onClick: onCreate }}
      />
    )
  }

  return (
    <div className="space-y-2">
      {notifications.map((notification) => (
        <NotificationChannelRow
          key={notification.id}
          notification={notification}
          typeName={getTypeName(notification.type)}
          isTestPending={isTestPending}
          onToggleEnabled={onToggleEnabled}
          onTest={onTest}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      ))}
    </div>
  )
}
