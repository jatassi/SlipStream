import { useState } from 'react'
import { Save, Loader2, Plus, Edit, Trash2, Bell, TestTube } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { RequestsNav } from './RequestsNav'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select'
import { LoadingState } from '@/components/data/LoadingState'
import { ErrorState } from '@/components/data/ErrorState'
import { EmptyState } from '@/components/data/EmptyState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { NotificationDialog } from '@/components/notifications/NotificationDialog'
import {
  useRequestSettings,
  useUpdateRequestSettings,
  useRootFolders,
  useNotifications,
  useDeleteNotification,
  useTestNotification,
  useUpdateNotification,
  useNotificationSchemas,
  systemKeys,
  useGlobalLoading,
} from '@/hooks'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import type { RequestSettings, Notification } from '@/types'

export function RequestSettingsPage() {
  const queryClient = useQueryClient()
  const globalLoading = useGlobalLoading()
  const { data: settings, isLoading: queryLoading, isError, refetch } = useRequestSettings()
  const isLoading = queryLoading || globalLoading
  const { data: rootFolders } = useRootFolders()
  const updateMutation = useUpdateRequestSettings()

  const [formData, setFormData] = useState<Partial<RequestSettings>>({})
  const [hasChanges, setHasChanges] = useState(false)
  const portalEnabled = formData.enabled ?? settings?.enabled ?? true

  // Notification state
  const [showNotificationDialog, setShowNotificationDialog] = useState(false)
  const [editingNotification, setEditingNotification] = useState<Notification | null>(null)

  const { data: notifications } = useNotifications()
  const { data: schemas } = useNotificationSchemas()
  const deleteNotificationMutation = useDeleteNotification()
  const testNotificationMutation = useTestNotification()
  const updateNotificationMutation = useUpdateNotification()

  const getTypeName = (type: string) => {
    return schemas?.find((s) => s.type === type)?.name || type
  }

  const handleOpenAddNotification = () => {
    setEditingNotification(null)
    setShowNotificationDialog(true)
  }

  const handleOpenEditNotification = (notification: Notification) => {
    setEditingNotification(notification)
    setShowNotificationDialog(true)
  }

  const handleToggleNotificationEnabled = async (id: number, enabled: boolean) => {
    try {
      await updateNotificationMutation.mutateAsync({ id, data: { enabled } })
      toast.success(enabled ? 'Notification enabled' : 'Notification disabled')
    } catch {
      toast.error('Failed to update notification')
    }
  }

  const handleTestNotification = async (id: number) => {
    try {
      const result = await testNotificationMutation.mutateAsync(id)
      if (result.success) {
        toast.success(result.message || 'Test notification sent')
      } else {
        toast.error(result.message || 'Test notification failed')
      }
    } catch {
      toast.error('Failed to test notification')
    }
  }

  const handleDeleteNotification = async (id: number) => {
    try {
      await deleteNotificationMutation.mutateAsync(id)
      toast.success('Notification deleted')
    } catch {
      toast.error('Failed to delete notification')
    }
  }

  // Track previous settings for render-time state sync
  const [prevSettings, setPrevSettings] = useState(settings)

  // Sync form state when settings change (React-recommended pattern)
  if (settings !== prevSettings) {
    setPrevSettings(settings)
    if (settings) {
      setFormData({
        enabled: settings.enabled,
        defaultMovieQuota: settings.defaultMovieQuota,
        defaultSeasonQuota: settings.defaultSeasonQuota,
        defaultEpisodeQuota: settings.defaultEpisodeQuota,
        defaultRootFolderId: settings.defaultRootFolderId,
        adminNotifyNew: settings.adminNotifyNew,
        searchRateLimit: settings.searchRateLimit,
      })
      setHasChanges(false)
    }
  }

  const handleChange = <K extends keyof RequestSettings>(key: K, value: RequestSettings[K]) => {
    setFormData((prev) => ({ ...prev, [key]: value }))
    setHasChanges(true)
  }

  const handleSave = async () => {
    try {
      await updateMutation.mutateAsync(formData)
      // Invalidate status query to update portalEnabled globally
      queryClient.invalidateQueries({ queryKey: systemKeys.status() })
      toast.success('Settings saved')
      setHasChanges(false)
    } catch {
      toast.error('Failed to save settings')
    }
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Request Settings" />
        <LoadingState variant="card" />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Request Settings" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="External Requests"
        description="Manage portal users and content requests"
        breadcrumbs={[
          { label: 'Settings', href: '/settings/media' },
          { label: 'External Requests' },
        ]}
        actions={
          <Button onClick={handleSave} disabled={!hasChanges || updateMutation.isPending}>
            {updateMutation.isPending ? (
              <Loader2 className="size-4 mr-2 animate-spin" />
            ) : (
              <Save className="size-4 mr-2" />
            )}
            Save Changes
          </Button>
        }
      />

      <RequestsNav />

      <div className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Portal Access</CardTitle>
            <CardDescription>
              Enable or disable the external requests portal for all users.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>Enable External Requests Portal</Label>
                <p className="text-sm text-muted-foreground">
                  When disabled, portal users cannot access the request system. Existing users and data are preserved.
                </p>
              </div>
              <Switch
                checked={formData.enabled ?? true}
                onCheckedChange={(checked) => handleChange('enabled', checked)}
              />
            </div>
          </CardContent>
        </Card>

        {portalEnabled && (
          <>
            <Card>
              <CardHeader>
                <CardTitle>Default Quotas</CardTitle>
            <CardDescription>
              Set the default weekly quota limits for new users. Users can have individual overrides.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-3">
              <div className="space-y-2">
                <Label htmlFor="movieQuota">Movies per Week</Label>
                <Input
                  id="movieQuota"
                  type="number"
                  min="0"
                  value={formData.defaultMovieQuota ?? ''}
                  onChange={(e) => handleChange('defaultMovieQuota', parseInt(e.target.value, 10) || 0)}
                  placeholder="e.g., 5"
                />
                <p className="text-xs text-muted-foreground">Set to 0 for unlimited</p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="seasonQuota">Seasons per Week</Label>
                <Input
                  id="seasonQuota"
                  type="number"
                  min="0"
                  value={formData.defaultSeasonQuota ?? ''}
                  onChange={(e) => handleChange('defaultSeasonQuota', parseInt(e.target.value, 10) || 0)}
                  placeholder="e.g., 3"
                />
                <p className="text-xs text-muted-foreground">Set to 0 for unlimited</p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="episodeQuota">Episodes per Week</Label>
                <Input
                  id="episodeQuota"
                  type="number"
                  min="0"
                  value={formData.defaultEpisodeQuota ?? ''}
                  onChange={(e) => handleChange('defaultEpisodeQuota', parseInt(e.target.value, 10) || 0)}
                  placeholder="e.g., 10"
                />
                <p className="text-xs text-muted-foreground">Set to 0 for unlimited</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Content Settings</CardTitle>
            <CardDescription>
              Configure default settings for requested content.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Default Root Folder</Label>
              <Select
                value={formData.defaultRootFolderId?.toString() || ''}
                onValueChange={(value) => handleChange('defaultRootFolderId', value ? parseInt(value, 10) : null)}
              >
                <SelectTrigger className="w-full max-w-md">
                  {formData.defaultRootFolderId
                    ? rootFolders?.find((f) => f.id === formData.defaultRootFolderId)?.path || 'Selected folder'
                    : 'Select default root folder'}
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">No default (use first available)</SelectItem>
                  {rootFolders?.map((folder) => (
                    <SelectItem key={folder.id} value={folder.id.toString()}>
                      {folder.path}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                The root folder where requested content will be downloaded by default.
              </p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <div>
              <CardTitle>Notifications</CardTitle>
              <CardDescription>
                Configure admin notifications for the request portal.
              </CardDescription>
            </div>
            <Button onClick={handleOpenAddNotification}>
              <Plus className="size-4 mr-2" />
              Add Channel
            </Button>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>Notify on New Requests</Label>
                <p className="text-sm text-muted-foreground">
                  Send a notification to admins when a new request is submitted.
                </p>
              </div>
              <Switch
                checked={formData.adminNotifyNew ?? false}
                onCheckedChange={(checked) => handleChange('adminNotifyNew', checked)}
              />
            </div>

            {/* Notification Channels */}
            <div className="border-t pt-4 mt-4">
              <Label className="text-sm font-medium">Notification Channels</Label>
              <p className="text-sm text-muted-foreground mb-3">
                Channels configured here will receive request notifications.
              </p>
              {!notifications?.length ? (
                <EmptyState
                  icon={<Bell className="size-6" />}
                  title="No channels configured"
                  description="Add a notification channel to receive request alerts"
                  action={{ label: 'Add Channel', onClick: handleOpenAddNotification }}
                  className="py-6"
                />
              ) : (
                <div className="space-y-2">
                  {notifications.map((notification) => (
                    <div
                      key={notification.id}
                      className="flex items-center justify-between p-3 rounded-lg border bg-muted/40"
                    >
                      <div className="flex items-center gap-3">
                        <div className="flex size-8 items-center justify-center rounded bg-background">
                          <Bell className="size-4" />
                        </div>
                        <div>
                          <div className="flex items-center gap-2">
                            <span className="font-medium text-sm">{notification.name}</span>
                            <Badge variant="outline" className="text-xs">
                              {getTypeName(notification.type)}
                            </Badge>
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Switch
                          checked={notification.enabled}
                          onCheckedChange={(checked) =>
                            handleToggleNotificationEnabled(notification.id, checked)
                          }
                        />
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleTestNotification(notification.id)}
                          disabled={testNotificationMutation.isPending}
                        >
                          <TestTube className="size-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleOpenEditNotification(notification)}
                        >
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
                          onConfirm={() => handleDeleteNotification(notification.id)}
                        />
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Rate Limiting</CardTitle>
            <CardDescription>
              Control search rate limits to prevent abuse.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="rateLimit">Search Rate Limit</Label>
              <div className="flex items-center gap-2 max-w-md">
                <Input
                  id="rateLimit"
                  type="number"
                  min="1"
                  max="100"
                  value={formData.searchRateLimit ?? ''}
                  onChange={(e) => handleChange('searchRateLimit', parseInt(e.target.value, 10) || 10)}
                />
                <span className="text-sm text-muted-foreground whitespace-nowrap">requests per minute</span>
              </div>
              <p className="text-xs text-muted-foreground">
                Maximum number of search requests a user can make per minute. Applies globally to all portal users.
              </p>
            </div>
          </CardContent>
        </Card>
          </>
        )}
      </div>

      <NotificationDialog
        open={showNotificationDialog}
        onOpenChange={setShowNotificationDialog}
        notification={editingNotification}
      />
    </div>
  )
}
