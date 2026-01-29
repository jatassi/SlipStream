import { useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import {
  ArrowLeft,
  Lock,
  Bell,
  Plus,
  Trash2,
  Loader2,
  TestTube,
  Edit,
  LogOut,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { EmptyState } from '@/components/data/EmptyState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { NotificationDialog } from '@/components/notifications/NotificationDialog'
import { PasskeyManager, ChangePinDialog } from '@/components/portal'
import {
  usePortalLogout,
  useUserNotifications,
  useUserNotificationSchema,
  useCreateUserNotification,
  useUpdateUserNotification,
  useDeleteUserNotification,
  useTestUserNotification,
} from '@/hooks'
import type { UserNotification, CreateNotificationInput, Notification, NotifierType } from '@/types'
import { toast } from 'sonner'

const portalEventTriggers = [
  { key: 'onApproved', label: 'Request Approved', description: 'When your request is approved by an admin' },
  { key: 'onDenied', label: 'Request Denied', description: 'When your request is denied by an admin' },
  { key: 'onAvailable', label: 'Request Available', description: 'When your requested content becomes available' },
]

export function PortalSettingsPage() {
  const navigate = useNavigate()
  const logoutMutation = usePortalLogout()
  const [pinDialogOpen, setPinDialogOpen] = useState(false)

  const goBack = () => {
    window.history.back()
  }

  const handleLogout = () => {
    logoutMutation.mutate(undefined, {
      onSuccess: () => {
        navigate({ to: '/requests/auth/login' })
      },
    })
  }

  return (
    <div className="max-w-4xl mx-auto pt-6 px-6 space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" onClick={goBack} className="text-xs md:text-sm">
          <ArrowLeft className="size-3 md:size-4 mr-0.5 md:mr-1" />
          Back
        </Button>
        <h1 className="text-xl md:text-2xl font-bold flex-1">Settings</h1>
        <Button variant="destructive" onClick={handleLogout} disabled={logoutMutation.isPending} className="text-xs md:text-sm">
          {logoutMutation.isPending ? (
            <Loader2 className="size-3 md:size-4 mr-1 md:mr-2 animate-spin" />
          ) : (
            <LogOut className="size-3 md:size-4 mr-1 md:mr-2" />
          )}
          Log Out
        </Button>
      </div>

      <Tabs defaultValue="security">
        <TabsList>
          <TabsTrigger value="security" className="text-xs md:text-sm">
            <Lock className="size-3 md:size-4 mr-1 md:mr-2" />
            Security
          </TabsTrigger>
          <TabsTrigger value="notifications" className="text-xs md:text-sm">
            <Bell className="size-3 md:size-4 mr-1 md:mr-2" />
            Notifications
          </TabsTrigger>
        </TabsList>

        <TabsContent value="security" className="mt-6 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>PIN</CardTitle>
              <CardDescription>Update your account PIN</CardDescription>
            </CardHeader>
            <CardContent>
              <Button onClick={() => setPinDialogOpen(true)} className="text-xs md:text-sm">
                <Lock className="size-3 md:size-4 mr-1 md:mr-2" />
                Change PIN...
              </Button>
            </CardContent>
          </Card>

          <PasskeyManager />
        </TabsContent>

        <TabsContent value="notifications" className="mt-6">
          <NotificationsSection />
        </TabsContent>
      </Tabs>

      <ChangePinDialog open={pinDialogOpen} onOpenChange={setPinDialogOpen} />
    </div>
  )
}

function NotificationsSection() {
  const { data: notifications = [], isLoading } = useUserNotifications()
  const { data: schemas = [] } = useUserNotificationSchema()
  const createMutation = useCreateUserNotification()
  const updateMutation = useUpdateUserNotification()
  const deleteMutation = useDeleteUserNotification()
  const testMutation = useTestUserNotification()

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingNotification, setEditingNotification] = useState<UserNotification | null>(null)

  const getTypeName = (type: string) => {
    return schemas.find((s) => s.type === type)?.name || type
  }

  const handleCreate = () => {
    setEditingNotification(null)
    setDialogOpen(true)
  }

  const handleEdit = (notification: UserNotification) => {
    setEditingNotification(notification)
    setDialogOpen(true)
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteMutation.mutateAsync(id)
      toast.success('Notification deleted')
    } catch {
      toast.error('Failed to delete notification')
    }
  }

  const handleTest = async (id: number) => {
    try {
      await testMutation.mutateAsync(id)
      toast.success('Test notification sent')
    } catch {
      toast.error('Failed to send test notification')
    }
  }

  const handleToggleEnabled = async (notification: UserNotification, enabled: boolean) => {
    try {
      await updateMutation.mutateAsync({
        id: notification.id,
        data: {
          type: notification.type,
          name: notification.name,
          settings: notification.settings,
          onAvailable: notification.onAvailable,
          onApproved: notification.onApproved,
          onDenied: notification.onDenied,
          enabled,
        },
      })
      toast.success(enabled ? 'Notification enabled' : 'Notification disabled')
    } catch {
      toast.error('Failed to update notification')
    }
  }

  const handleCreateNotification = async (data: CreateNotificationInput) => {
    const eventData = data as unknown as Record<string, unknown>
    await createMutation.mutateAsync({
      type: data.type,
      name: data.name,
      settings: data.settings,
      onAvailable: (eventData.onAvailable as boolean | undefined) ?? true,
      onApproved: (eventData.onApproved as boolean | undefined) ?? true,
      onDenied: (eventData.onDenied as boolean | undefined) ?? true,
      enabled: data.enabled ?? true,
    })
  }

  const handleUpdateNotification = async (id: number, data: CreateNotificationInput) => {
    const eventData = data as unknown as Record<string, unknown>
    await updateMutation.mutateAsync({
      id,
      data: {
        type: data.type,
        name: data.name,
        settings: data.settings,
        onAvailable: (eventData.onAvailable as boolean | undefined) ?? true,
        onApproved: (eventData.onApproved as boolean | undefined) ?? true,
        onDenied: (eventData.onDenied as boolean | undefined) ?? true,
        enabled: data.enabled ?? true,
      },
    })
  }

  // Convert UserNotification to Notification type for the dialog
  const notificationForDialog: Notification | null = editingNotification
    ? {
        id: editingNotification.id,
        name: editingNotification.name,
        type: editingNotification.type as NotifierType,
        enabled: editingNotification.enabled,
        settings: editingNotification.settings,
        onGrab: false,
        onImport: false,
        onUpgrade: false,
        onMovieAdded: false,
        onMovieDeleted: false,
        onSeriesAdded: false,
        onSeriesDeleted: false,
        onHealthIssue: false,
        onHealthRestored: false,
        onAppUpdate: false,
        includeHealthWarnings: false,
        onAvailable: editingNotification.onAvailable,
        onApproved: editingNotification.onApproved,
        onDenied: editingNotification.onDenied,
        tags: [],
      }
    : null

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Notification Channels</CardTitle>
              <CardDescription>
                Get notified when your requests become available
              </CardDescription>
            </div>
            <Button onClick={handleCreate} className="text-xs md:text-sm">
              <Plus className="size-3 md:size-4 mr-1 md:mr-2" />
              Add Channel
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="size-6 animate-spin text-muted-foreground" />
            </div>
          ) : notifications.length === 0 ? (
            <EmptyState
              icon={<Bell className="size-8" />}
              title="No notification channels"
              description="Add a notification channel to get notified when your requests become available"
              action={{ label: 'Add Channel', onClick: handleCreate }}
            />
          ) : (
            <div className="space-y-2">
              {notifications.map((notification) => (
                <div
                  key={notification.id}
                  className="flex items-center justify-between p-4 rounded-lg border border-border"
                >
                  <div className="flex items-center gap-3 md:gap-4">
                    <div className="p-1.5 md:p-2 rounded-lg bg-muted">
                      <Bell className="size-4 md:size-5" />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <p className="font-medium">{notification.name}</p>
                        <Badge variant="outline" className="text-xs">
                          {getTypeName(notification.type)}
                        </Badge>
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-1 md:gap-2">
                    <Switch
                      checked={notification.enabled}
                      onCheckedChange={(enabled) => handleToggleEnabled(notification, enabled)}
                    />
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleTest(notification.id)}
                      disabled={testMutation.isPending}
                      className="size-8 md:size-9"
                    >
                      <TestTube className="size-3 md:size-4" />
                    </Button>
                    <Button variant="ghost" size="icon" onClick={() => handleEdit(notification)} className="size-8 md:size-9">
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
                      onConfirm={() => handleDelete(notification.id)}
                    />
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <NotificationDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        notification={notificationForDialog}
        eventTriggers={portalEventTriggers}
        schemas={schemas}
        onCreate={handleCreateNotification}
        onUpdate={handleUpdateNotification}
      />
    </div>
  )
}
