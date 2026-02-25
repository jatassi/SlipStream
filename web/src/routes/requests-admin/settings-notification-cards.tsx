import { Bell, Edit, Plus, TestTube, Trash2 } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import { ConfirmDialog } from '@/components/forms/confirm-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import type { Notification, RequestSettings } from '@/types'

type FormChangeHandler = <K extends keyof RequestSettings>(
  key: K,
  value: RequestSettings[K],
) => void

type NotificationsCardProps = {
  notifications: Notification[] | undefined
  formData: Partial<RequestSettings>
  onChange: FormChangeHandler
  getTypeName: (type: string) => string
  onAdd: () => void
  onEdit: (notification: Notification) => void
  onTest: (id: number) => Promise<void>
  onDelete: (id: number) => Promise<void>
  onToggleEnabled: (id: number, enabled: boolean) => Promise<void>
  isTestPending: boolean
}

export function NotificationsCard(props: NotificationsCardProps) {
  const { formData, onChange, onAdd } = props

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <CardTitle>Notifications</CardTitle>
          <CardDescription>Configure admin notifications for the request portal.</CardDescription>
        </div>
        <Button onClick={onAdd}>
          <Plus className="mr-2 size-4" />
          Add Channel
        </Button>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="space-y-0.5">
            <Label>Notify on New Requests</Label>
            <p className="text-muted-foreground text-sm">
              Send a notification to admins when a new request is submitted.
            </p>
          </div>
          <Switch
            checked={formData.adminNotifyNew ?? false}
            onCheckedChange={(checked) => onChange('adminNotifyNew', checked)}
          />
        </div>
        <div className="mt-4 border-t pt-4">
          <Label className="text-sm font-medium">Notification Channels</Label>
          <p className="text-muted-foreground mb-3 text-sm">
            Channels configured here will receive request notifications.
          </p>
          <NotificationChannelsList
            notifications={props.notifications}
            getTypeName={props.getTypeName}
            onAdd={onAdd}
            onEdit={props.onEdit}
            onTest={props.onTest}
            onDelete={props.onDelete}
            onToggleEnabled={props.onToggleEnabled}
            isTestPending={props.isTestPending}
          />
        </div>
      </CardContent>
    </Card>
  )
}

function NotificationChannelsList(props: {
  notifications: Notification[] | undefined
  getTypeName: (type: string) => string
  onAdd: () => void
  onEdit: (notification: Notification) => void
  onTest: (id: number) => Promise<void>
  onDelete: (id: number) => Promise<void>
  onToggleEnabled: (id: number, enabled: boolean) => Promise<void>
  isTestPending: boolean
}) {
  if (!props.notifications?.length) {
    return (
      <EmptyState
        icon={<Bell className="size-6" />}
        title="No channels configured"
        description="Add a notification channel to receive request alerts"
        action={{ label: 'Add Channel', onClick: props.onAdd }}
        className="py-6"
      />
    )
  }

  return (
    <div className="space-y-2">
      {props.notifications.map((notification) => (
        <NotificationChannelRow
          key={notification.id}
          notification={notification}
          typeName={props.getTypeName(notification.type)}
          onEdit={props.onEdit}
          onTest={props.onTest}
          onDelete={props.onDelete}
          onToggleEnabled={props.onToggleEnabled}
          isTestPending={props.isTestPending}
        />
      ))}
    </div>
  )
}

function NotificationChannelRow(props: {
  notification: Notification
  typeName: string
  onEdit: (notification: Notification) => void
  onTest: (id: number) => Promise<void>
  onDelete: (id: number) => Promise<void>
  onToggleEnabled: (id: number, enabled: boolean) => Promise<void>
  isTestPending: boolean
}) {
  const { notification } = props

  return (
    <div className="bg-muted/40 flex items-center justify-between rounded-lg border p-3">
      <div className="flex items-center gap-3">
        <div className="bg-background flex size-8 items-center justify-center rounded">
          <Bell className="size-4" />
        </div>
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">{notification.name}</span>
          <Badge variant="outline" className="text-xs">
            {props.typeName}
          </Badge>
        </div>
      </div>
      <NotificationChannelActions
        notification={notification}
        onEdit={props.onEdit}
        onTest={props.onTest}
        onDelete={props.onDelete}
        onToggleEnabled={props.onToggleEnabled}
        isTestPending={props.isTestPending}
      />
    </div>
  )
}

function NotificationChannelActions(props: {
  notification: Notification
  onEdit: (notification: Notification) => void
  onTest: (id: number) => Promise<void>
  onDelete: (id: number) => Promise<void>
  onToggleEnabled: (id: number, enabled: boolean) => Promise<void>
  isTestPending: boolean
}) {
  const { notification } = props

  return (
    <div className="flex items-center gap-2">
      <Switch
        checked={notification.enabled}
        onCheckedChange={(checked) => props.onToggleEnabled(notification.id, checked)}
      />
      <Button
        variant="ghost"
        size="icon"
        onClick={() => props.onTest(notification.id)}
        disabled={props.isTestPending}
      >
        <TestTube className="size-4" />
      </Button>
      <Button variant="ghost" size="icon" onClick={() => props.onEdit(notification)}>
        <Edit className="size-4" />
      </Button>
      <ConfirmDialog
        trigger={
          <Button variant="ghost" size="icon">
            <Trash2 className="size-4" />
          </Button>
        }
        title="Delete notification"
        description={`Are you sure you want to delete "${notification.name}"?`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={() => props.onDelete(notification.id)}
      />
    </div>
  )
}
