import { useState } from 'react'

import { Bell, Lock } from 'lucide-react'

import { NotificationDialog } from '@/components/notifications/notification-dialog'
import { ChangePinDialog, PasskeyManager } from '@/components/portal'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { NotificationChannelsCard } from './notification-channels-card'
import { SettingsHeader } from './settings-header'
import { useNotificationsSection } from './use-request-settings'

export function PortalSettingsPage() {
  const [pinDialogOpen, setPinDialogOpen] = useState(false)

  return (
    <div className="mx-auto max-w-4xl space-y-6 px-6 pt-6">
      <SettingsHeader />

      <Tabs defaultValue="security">
        <TabsList>
          <TabsTrigger value="security" className="text-xs md:text-sm">
            <Lock className="mr-1 size-3 md:mr-2 md:size-4" />
            Security
          </TabsTrigger>
          <TabsTrigger value="notifications" className="text-xs md:text-sm">
            <Bell className="mr-1 size-3 md:mr-2 md:size-4" />
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
                <Lock className="mr-1 size-3 md:mr-2 md:size-4" />
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
  const state = useNotificationsSection()

  return (
    <div className="space-y-6">
      <NotificationChannelsCard
        notifications={state.notifications}
        isLoading={state.isLoading}
        isTestPending={state.isTestPending}
        getTypeName={state.getTypeName}
        onCreate={state.handleCreate}
        onEdit={state.handleEdit}
        onDelete={state.handleDelete}
        onTest={state.handleTest}
        onToggleEnabled={state.handleToggleEnabled}
      />

      <NotificationDialog
        open={state.dialogOpen}
        onOpenChange={state.setDialogOpen}
        notification={state.notificationForDialog}
        eventTriggers={state.portalEventTriggers}
        schemas={state.schemas}
        onCreate={state.handleCreateNotification}
        onUpdate={state.handleUpdateNotification}
      />
    </div>
  )
}
