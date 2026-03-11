import type { ReactNode } from 'react'

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
import { useNotificationEventCatalog } from '@/hooks'
import type { Notification, NotificationEventGroup } from '@/types'

import { GeneralNav } from './general-nav'
import { useNotificationsPage } from './use-notifications-page'

function getActiveEventsText(notification: Notification, catalog: NotificationEventGroup[]): string {
  const allEvents = catalog.flatMap((g) => g.events)
  const activeLabels = allEvents
    .filter((e) => notification.eventToggles[e.id])
    .map((e) => e.label)
  return activeLabels.length > 0 ? activeLabels.join(', ') : 'No events'
}

type NotificationCardProps = {
  notification: Notification
  typeName: string
  eventCatalog: NotificationEventGroup[]
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
      <Button variant="ghost" size="icon" aria-label="Edit" onClick={() => onEdit(notification)}>
        <Edit className="size-4" />
      </Button>
      <ConfirmDialog
        trigger={
          <Button variant="ghost" size="icon" aria-label="Delete">
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
  const { notification, typeName, eventCatalog } = props
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
              {getActiveEventsText(notification, eventCatalog)}
            </CardDescription>
          </div>
        </div>
        <CardActions {...props} />
      </CardHeader>
    </Card>
  )
}

function NotificationsLayout({ children }: { children: ReactNode }) {
  return (
    <div className="space-y-6">
      <PageHeader
        title="General"
        description="Server configuration, authentication, and notification settings"
        breadcrumbs={[{ label: 'Settings', href: '/settings/media' }, { label: 'General' }]}
      />
      <GeneralNav />
      {children}
    </div>
  )
}

export function NotificationsPage() {
  const state = useNotificationsPage()
  const { data: eventCatalog } = useNotificationEventCatalog()

  if (state.isLoading) {
    return (
      <NotificationsLayout>
        <LoadingState variant="list" count={3} />
      </NotificationsLayout>
    )
  }

  if (state.isError) {
    return (
      <NotificationsLayout>
        <ErrorState onRetry={state.refetch} />
      </NotificationsLayout>
    )
  }

  return (
    <NotificationsLayout>

      <div className="space-y-4">
        {state.notifications?.map((notification) => (
          <NotificationCard
            key={notification.id}
            notification={notification}
            typeName={state.getTypeName(notification.type)}
            eventCatalog={eventCatalog ?? []}
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
    </NotificationsLayout>
  )
}
