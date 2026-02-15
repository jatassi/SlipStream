import { Bell, Edit, TestTube, Trash2 } from 'lucide-react'

import { ConfirmDialog } from '@/components/forms/confirm-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import type { UserNotification } from '@/types'

type NotificationChannelRowProps = {
  notification: UserNotification
  typeName: string
  isTestPending: boolean
  onToggleEnabled: (notification: UserNotification, enabled: boolean) => void
  onTest: (id: number) => void
  onEdit: (notification: UserNotification) => void
  onDelete: (id: number) => void
}

export function NotificationChannelRow({
  notification,
  typeName,
  isTestPending,
  onToggleEnabled,
  onTest,
  onEdit,
  onDelete,
}: NotificationChannelRowProps) {
  return (
    <div className="border-border flex items-center justify-between rounded-lg border p-4">
      <div className="flex items-center gap-3 md:gap-4">
        <div className="bg-muted rounded-lg p-1.5 md:p-2">
          <Bell className="size-4 md:size-5" />
        </div>
        <div className="flex items-center gap-2">
          <p className="font-medium">{notification.name}</p>
          <Badge variant="outline" className="text-xs">
            {typeName}
          </Badge>
        </div>
      </div>
      <ChannelActions
        notification={notification}
        isTestPending={isTestPending}
        onToggleEnabled={onToggleEnabled}
        onTest={onTest}
        onEdit={onEdit}
        onDelete={onDelete}
      />
    </div>
  )
}

type ChannelActionsProps = Omit<NotificationChannelRowProps, 'typeName'>

function ChannelActions({
  notification,
  isTestPending,
  onToggleEnabled,
  onTest,
  onEdit,
  onDelete,
}: ChannelActionsProps) {
  return (
    <div className="flex items-center gap-1 md:gap-2">
      <Switch
        checked={notification.enabled}
        onCheckedChange={(enabled) => onToggleEnabled(notification, enabled)}
      />
      <Button
        variant="ghost"
        size="icon"
        onClick={() => onTest(notification.id)}
        disabled={isTestPending}
        className="size-8 md:size-9"
      >
        <TestTube className="size-3 md:size-4" />
      </Button>
      <Button
        variant="ghost"
        size="icon"
        onClick={() => onEdit(notification)}
        className="size-8 md:size-9"
      >
        <Edit className="size-3 md:size-4" />
      </Button>
      <ConfirmDialog
        trigger={
          <Button variant="ghost" size="icon" className="size-8 md:size-9">
            <Trash2 className="size-3 md:size-4" />
          </Button>
        }
        title="Delete notification"
        description={`Are you sure you want to delete "${notification.name}"?`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={() => onDelete(notification.id)}
      />
    </div>
  )
}
