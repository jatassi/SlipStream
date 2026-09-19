import { Bell } from 'lucide-react'

import { IconTile } from '@/components/grouped-list'
import { usePushBack } from '@/components/layout/use-push-back'
import { NotificationDialog } from '@/components/notifications/notification-dialog'
import { Screen } from '@/components/screen/screen'
import type { SettingsRowAction } from '@/components/settings/settings-item-row'
import { SettingsItemRow } from '@/components/settings/settings-item-row'
import { AddAction, SettingsList } from '@/components/settings/settings-list'
import { useNotificationEventCatalog } from '@/hooks'
import type { Notification, NotificationEventGroup } from '@/types'

import { useNotificationsPage } from './use-notifications-page'

type PageState = ReturnType<typeof useNotificationsPage>

export function NotificationsPage() {
  const page = useNotificationsPage()
  const { data: eventCatalog } = useNotificationEventCatalog()
  const back = usePushBack()

  return (
    <Screen
      title="Notifications"
      back={back}
      trailing={<AddAction label="Add notification channel" onClick={page.handleOpenAdd} />}
    >
      <SettingsList
        state={{
          isLoading: page.isLoading,
          isError: page.isError,
          isEmpty: !page.notifications?.length,
          refetch: page.refetch,
        }}
        empty="No notification channels yet"
      >
        {page.notifications?.map((notification) => (
          <NotificationRow
            key={notification.id}
            notification={notification}
            catalog={eventCatalog ?? []}
            page={page}
          />
        ))}
      </SettingsList>
      <NotificationDialog
        open={page.showDialog}
        onOpenChange={page.setShowDialog}
        notification={page.editingNotification}
      />
    </Screen>
  )
}

function NotificationRow({
  notification,
  catalog,
  page,
}: {
  notification: Notification
  catalog: NotificationEventGroup[]
  page: PageState
}) {
  const actions: SettingsRowAction[] = [
    { label: 'Test', onClick: () => page.handleTest(notification.id) },
    {
      label: 'Delete',
      destructive: true,
      onClick: () => page.handleDelete(notification.id),
      confirm: {
        title: 'Delete notification',
        description: `Are you sure you want to delete "${notification.name}"?`,
      },
    },
  ]

  return (
    <SettingsItemRow
      leading={
        <IconTile className="bg-sky-600">
          <Bell />
        </IconTile>
      }
      title={notification.name}
      subtitle={`${page.getTypeName(notification.type)} · ${activeEvents(notification, catalog)}`}
      onOpen={() => page.handleOpenEdit(notification)}
      openLabel={`Edit ${notification.name}`}
      toggle={{
        label: `${notification.name} enabled`,
        checked: notification.enabled,
        onCheckedChange: (checked) => void page.handleToggleEnabled(notification.id, checked),
      }}
      actions={actions}
    />
  )
}

function activeEvents(notification: Notification, catalog: NotificationEventGroup[]): string {
  const labels = catalog
    .flatMap((group) => group.events)
    .filter((event) => notification.eventToggles[event.id])
    .map((event) => event.label)
  return labels.length > 0 ? labels.join(', ') : 'No events'
}
