import { useState } from 'react'
import { Plus, Edit, Trash2, Bell, TestTube } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { NotificationDialog } from '@/components/notifications/NotificationDialog'
import {
  useNotifications,
  useDeleteNotification,
  useTestNotification,
  useUpdateNotification,
  useNotificationSchemas,
} from '@/hooks'
import { toast } from 'sonner'
import type { Notification } from '@/types'

export function NotificationsPage() {
  const [showDialog, setShowDialog] = useState(false)
  const [editingNotification, setEditingNotification] = useState<Notification | null>(null)

  const { data: notifications, isLoading, isError, refetch } = useNotifications()
  const { data: schemas } = useNotificationSchemas()
  const deleteMutation = useDeleteNotification()
  const testMutation = useTestNotification()
  const updateMutation = useUpdateNotification()

  const getTypeName = (type: string) => {
    return schemas?.find((s) => s.type === type)?.name || type
  }

  const getActiveEvents = (notification: Notification) => {
    const events = []
    if (notification.onGrab) events.push('Grab')
    if (notification.onDownload) events.push('Download')
    if (notification.onUpgrade) events.push('Upgrade')
    if (notification.onMovieAdded) events.push('Movie Added')
    if (notification.onMovieDeleted) events.push('Movie Deleted')
    if (notification.onSeriesAdded) events.push('Series Added')
    if (notification.onSeriesDeleted) events.push('Series Deleted')
    if (notification.onHealthIssue) events.push('Health')
    if (notification.onAppUpdate) events.push('App Update')
    return events
  }

  const handleOpenAdd = () => {
    setEditingNotification(null)
    setShowDialog(true)
  }

  const handleOpenEdit = (notification: Notification) => {
    setEditingNotification(notification)
    setShowDialog(true)
  }

  const handleToggleEnabled = async (id: number, enabled: boolean) => {
    try {
      await updateMutation.mutateAsync({ id, data: { enabled } })
      toast.success(enabled ? 'Notification enabled' : 'Notification disabled')
    } catch {
      toast.error('Failed to update notification')
    }
  }

  const handleTest = async (id: number) => {
    try {
      const result = await testMutation.mutateAsync(id)
      if (result.success) {
        toast.success(result.message || 'Test notification sent')
      } else {
        toast.error(result.message || 'Test notification failed')
      }
    } catch {
      toast.error('Failed to test notification')
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteMutation.mutateAsync(id)
      toast.success('Notification deleted')
    } catch {
      toast.error('Failed to delete notification')
    }
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Notifications" />
        <LoadingState variant="list" count={3} />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Notifications" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="Notifications"
        description="Configure notification channels for events"
        breadcrumbs={[
          { label: 'Settings', href: '/settings' },
          { label: 'Notifications' },
        ]}
        actions={
          <Button onClick={handleOpenAdd}>
            <Plus className="size-4 mr-2" />
            Add Notification
          </Button>
        }
      />

      {!notifications?.length ? (
        <EmptyState
          icon={<Bell className="size-8" />}
          title="No notifications configured"
          description="Add a notification channel to receive alerts"
          action={{ label: 'Add Notification', onClick: handleOpenAdd }}
        />
      ) : (
        <div className="space-y-4">
          {notifications.map((notification) => (
            <Card key={notification.id}>
              <CardHeader className="flex flex-row items-center justify-between py-4">
                <div className="flex items-center gap-4">
                  <div className="flex size-10 items-center justify-center rounded-lg bg-muted">
                    <Bell className="size-5" />
                  </div>
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <CardTitle className="text-base">{notification.name}</CardTitle>
                      <Badge variant="outline">{getTypeName(notification.type)}</Badge>
                    </div>
                    <CardDescription className="text-xs">
                      {getActiveEvents(notification).length > 0
                        ? getActiveEvents(notification).join(', ')
                        : 'No events configured'}
                    </CardDescription>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <Switch
                    checked={notification.enabled}
                    onCheckedChange={(checked) => handleToggleEnabled(notification.id, checked)}
                  />
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleTest(notification.id)}
                    disabled={testMutation.isPending}
                  >
                    <TestTube className="size-4 mr-1" />
                    Test
                  </Button>
                  <Button variant="ghost" size="icon" onClick={() => handleOpenEdit(notification)}>
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
                    onConfirm={() => handleDelete(notification.id)}
                  />
                </div>
              </CardHeader>
            </Card>
          ))}
        </div>
      )}

      <NotificationDialog
        open={showDialog}
        onOpenChange={setShowDialog}
        notification={editingNotification}
      />
    </div>
  )
}
