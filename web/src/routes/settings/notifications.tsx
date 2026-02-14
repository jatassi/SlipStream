import { useState } from 'react'

import { Bell, Edit, TestTube, Trash2 } from 'lucide-react'
import { toast } from 'sonner'

import { ErrorState } from '@/components/data/ErrorState'
import { LoadingState } from '@/components/data/LoadingState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { PageHeader } from '@/components/layout/PageHeader'
import { NotificationDialog } from '@/components/notifications/NotificationDialog'
import { AddPlaceholderCard } from '@/components/settings'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import {
  useDeleteNotification,
  useGlobalLoading,
  useNotifications,
  useNotificationSchemas,
  useTestNotification,
  useUpdateNotification,
} from '@/hooks'
import type { Notification } from '@/types'

export function NotificationsPage() {
  const [showDialog, setShowDialog] = useState(false)
  const [editingNotification, setEditingNotification] = useState<Notification | null>(null)

  const globalLoading = useGlobalLoading()
  const { data: notifications, isLoading: queryLoading, isError, refetch } = useNotifications()
  const isLoading = queryLoading || globalLoading
  const { data: schemas } = useNotificationSchemas()
  const deleteMutation = useDeleteNotification()
  const testMutation = useTestNotification()
  const updateMutation = useUpdateNotification()

  const getTypeName = (type: string) => {
    return schemas?.find((s) => s.type === type)?.name || type
  }

  const getActiveEvents = (notification: Notification) => {
    const events = []
    if (notification.onGrab) {
      events.push('Grab')
    }
    if (notification.onImport) {
      events.push('Import')
    }
    if (notification.onUpgrade) {
      events.push('Upgrade')
    }
    if (notification.onMovieAdded) {
      events.push('Movie Added')
    }
    if (notification.onMovieDeleted) {
      events.push('Movie Deleted')
    }
    if (notification.onSeriesAdded) {
      events.push('Series Added')
    }
    if (notification.onSeriesDeleted) {
      events.push('Series Deleted')
    }
    if (notification.onHealthIssue) {
      events.push('Health')
    }
    if (notification.onAppUpdate) {
      events.push('App Update')
    }
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
        breadcrumbs={[{ label: 'Settings', href: '/settings' }, { label: 'Notifications' }]}
      />

      <div className="space-y-4">
        {notifications?.map((notification) => (
          <Card key={notification.id}>
            <CardHeader className="flex flex-row items-center justify-between py-4">
              <div className="flex items-center gap-4">
                <div className="bg-muted flex size-10 items-center justify-center rounded-lg">
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
                  <TestTube className="mr-1 size-4" />
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
        <AddPlaceholderCard label="Add Notification Channel" onClick={handleOpenAdd} />
      </div>

      <NotificationDialog
        open={showDialog}
        onOpenChange={setShowDialog}
        notification={editingNotification}
      />
    </div>
  )
}
