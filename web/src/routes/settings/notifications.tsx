import { Bell, Edit, TestTube, Trash2 } from 'lucide-react'

import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { ConfirmDialog } from '@/components/forms/confirm-dialog'
import { PageHeader } from '@/components/layout/page-header'
import { NotificationDialog } from '@/components/notifications/notification-dialog'
import { AddPlaceholderCard } from '@/components/settings'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import type { Notification } from '@/types'

import { useNotificationsPage } from './use-notifications-page'

const EVENT_FLAGS = [
  ['onGrab', 'Grab'],
  ['onImport', 'Import'],
  ['onUpgrade', 'Upgrade'],
  ['onMovieAdded', 'Movie Added'],
  ['onMovieDeleted', 'Movie Deleted'],
  ['onSeriesAdded', 'Series Added'],
  ['onSeriesDeleted', 'Series Deleted'],
  ['onHealthIssue', 'Health'],
  ['onAppUpdate', 'App Update'],
] as const

function getActiveEventsText(notification: Notification): string {
  const labels = EVENT_FLAGS.filter(([key]) => notification[key]).map(([, label]) => label)
  return labels.length > 0 ? labels.join(', ') : 'No events configured'
}

type NotificationCardProps = {
  notification: Notification
  typeName: string
  testPending: boolean
  onToggle: (id: number, enabled: boolean) => void
  onTest: (id: number) => void
  onEdit: (notification: Notification) => void
  onDelete: (id: number) => void
}

function CardActions({ notification, testPending, onToggle, onTest, onEdit, onDelete }: Omit<NotificationCardProps, 'typeName'>) {
  return (
    <div className="flex items-center gap-4">
      <Switch
        checked={notification.enabled}
        onCheckedChange={(checked) => onToggle(notification.id, checked)}
      />
      <Button variant="outline" size="sm" onClick={() => onTest(notification.id)} disabled={testPending}>
        <TestTube className="mr-1 size-4" />
        Test
      </Button>
      <Button variant="ghost" size="icon" onClick={() => onEdit(notification)}>
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
        onConfirm={() => onDelete(notification.id)}
      />
    </div>
  )
}

function NotificationCard(props: NotificationCardProps) {
  const { notification, typeName } = props
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between py-4">
        <div className="flex items-center gap-4">
          <div className="bg-muted flex size-10 items-center justify-center rounded-lg">
            <Bell className="size-5" />
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <CardTitle className="text-base">{notification.name}</CardTitle>
              <Badge variant="outline">{typeName}</Badge>
            </div>
            <CardDescription className="text-xs">
              {getActiveEventsText(notification)}
            </CardDescription>
          </div>
        </div>
        <CardActions {...props} />
      </CardHeader>
    </Card>
  )
}

export function NotificationsPage() {
  const state = useNotificationsPage()

  if (state.isLoading) {
    return (
      <div>
        <PageHeader title="Notifications" />
        <LoadingState variant="list" count={3} />
      </div>
    )
  }

  if (state.isError) {
    return (
      <div>
        <PageHeader title="Notifications" />
        <ErrorState onRetry={state.refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="Notifications"
        description="Configure notification channels for events"
        breadcrumbs={[{ label: 'Settings', href: '/settings' }, { label: 'Notifications' }]}
      />

      <div className="space-y-4">
        {state.notifications?.map((notification) => (
          <NotificationCard
            key={notification.id}
            notification={notification}
            typeName={state.getTypeName(notification.type)}
            testPending={state.testPending}
            onToggle={state.handleToggleEnabled}
            onTest={state.handleTest}
            onEdit={state.handleOpenEdit}
            onDelete={state.handleDelete}
          />
        ))}
        <AddPlaceholderCard label="Add Notification Channel" onClick={state.handleOpenAdd} />
      </div>

      <NotificationDialog
        open={state.showDialog}
        onOpenChange={state.setShowDialog}
        notification={state.editingNotification}
      />
    </div>
  )
}
