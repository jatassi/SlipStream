import { Bell, Save } from 'lucide-react'

import { Group, IconTile } from '@/components/grouped-list'
import { usePushBack } from '@/components/layout/use-push-back'
import { NotificationDialog } from '@/components/notifications/notification-dialog'
import { Screen } from '@/components/screen/screen'
import type { SelectOption } from '@/components/settings/control-row'
import { InputRow, SelectRow, SwitchRow } from '@/components/settings/control-row'
import { SectionError, SectionLoading } from '@/components/settings/section-state'
import type { SettingsRowAction } from '@/components/settings/settings-item-row'
import { SettingsItemRow } from '@/components/settings/settings-item-row'
import { Button } from '@/components/ui/button'
import type { Notification, RootFolder } from '@/types'

import { useRequestSettingsPage } from './use-settings-page'

type PageState = ReturnType<typeof useRequestSettingsPage>

const NO_ROOT_FOLDER = ''

function moduleQuotaLabel(moduleType: string): string {
  return `${moduleType.charAt(0).toUpperCase() + moduleType.slice(1)} per Week`
}

function rootFolderOptions(rootFolders: RootFolder[] | undefined): SelectOption[] {
  return [
    { value: NO_ROOT_FOLDER, label: 'No default (use first available)' },
    ...(rootFolders ?? []).map((folder) => ({
      value: folder.id.toString(),
      label: folder.path,
    })),
  ]
}

function PortalAccessGroup({ page }: { page: PageState }) {
  return (
    <Group
      header="Portal Access"
      footer="When disabled, portal users cannot access the request system. Existing users and data are preserved."
    >
      <SwitchRow
        label="Enable External Requests Portal"
        checked={page.formData.enabled ?? true}
        onCheckedChange={(checked) => page.handleChange('enabled', checked)}
      />
    </Group>
  )
}

function QuotasGroup({ page }: { page: PageState }) {
  const quotas = page.formData.defaultQuotas ?? {}
  const moduleTypes = Object.keys(quotas).length > 0 ? Object.keys(quotas) : ['movie', 'tv']

  return (
    <Group
      header="Default Quotas"
      footer="Weekly limits applied to new users. Individual users can override them. Set a quota to 0 for unlimited."
    >
      {moduleTypes.map((moduleType) => (
        <InputRow
          key={moduleType}
          label={moduleQuotaLabel(moduleType)}
          type="number"
          min={0}
          value={quotas[moduleType] ?? ''}
          onChange={(e) =>
            page.handleQuotaChange(moduleType, Number.parseInt(e.target.value, 10) || 0)
          }
        />
      ))}
    </Group>
  )
}

function ContentGroup({ page }: { page: PageState }) {
  return (
    <Group
      header="Content"
      footer="The root folder where requested content is downloaded by default."
    >
      <SelectRow
        label="Default Root Folder"
        value={page.formData.defaultRootFolderId?.toString() ?? NO_ROOT_FOLDER}
        onChange={(value) =>
          page.handleChange(
            'defaultRootFolderId',
            value === NO_ROOT_FOLDER ? null : Number.parseInt(value, 10),
          )
        }
        options={rootFolderOptions(page.rootFolders)}
      />
    </Group>
  )
}

function ChannelRow({ notification, page }: { notification: Notification; page: PageState }) {
  const actions: SettingsRowAction[] = [
    { label: 'Test', onClick: () => page.handleTestNotification(notification.id) },
    {
      label: 'Delete channel',
      destructive: true,
      onClick: () => page.handleDeleteNotification(notification.id),
      confirm: {
        title: `Delete ${notification.name}?`,
        description: 'The channel stops receiving request notifications.',
      },
    },
  ]

  return (
    <SettingsItemRow
      leading={
        <IconTile className="bg-rose-600">
          <Bell />
        </IconTile>
      }
      title={notification.name}
      subtitle={page.getTypeName(notification.type)}
      onOpen={() => {
        page.handleOpenEditNotification(notification)
      }}
      openLabel={`Edit ${notification.name}`}
      toggle={{
        label: `${notification.name} enabled`,
        checked: notification.enabled,
        onCheckedChange: (checked) =>
          void page.handleToggleNotificationEnabled(notification.id, checked),
      }}
      actions={actions}
    />
  )
}

function NotificationsGroups({ page }: { page: PageState }) {
  const channels = page.notifications ?? []

  return (
    <>
      <Group
        header="Notifications"
        footer="Send a notification to admins when a new request is submitted."
      >
        <SwitchRow
          label="Notify on New Requests"
          checked={page.formData.adminNotifyNew ?? false}
          onCheckedChange={(checked) => page.handleChange('adminNotifyNew', checked)}
        />
      </Group>
      <Group
        header="Notification Channels"
        action={
          <button
            type="button"
            onClick={page.handleOpenAddNotification}
            className="press-dim text-footnote text-primary focus-visible:ring-ring rounded-md px-1 font-semibold outline-none focus-visible:ring-[3px]"
          >
            Add Channel
          </button>
        }
        footer="Channels configured here receive request notifications."
      >
        {channels.length === 0 ? (
          <div className="min-h-tap text-body text-muted-foreground flex items-center px-4 py-2.5">
            No channels configured
          </div>
        ) : (
          channels.map((notification) => (
            <ChannelRow key={notification.id} notification={notification} page={page} />
          ))
        )}
      </Group>
    </>
  )
}

function RateLimitGroup({ page }: { page: PageState }) {
  return (
    <Group
      header="Rate Limiting"
      footer="Maximum number of search requests a user can make per minute. Applies to all portal users."
    >
      <InputRow
        label="Searches per Minute"
        type="number"
        min={1}
        max={100}
        value={page.formData.searchRateLimit ?? ''}
        onChange={(e) =>
          page.handleChange('searchRateLimit', Number.parseInt(e.target.value, 10) || 10)
        }
      />
    </Group>
  )
}

function PortalGroups({ page }: { page: PageState }) {
  if (!page.portalEnabled) {
    return null
  }
  return (
    <>
      <QuotasGroup page={page} />
      <ContentGroup page={page} />
      <NotificationsGroups page={page} />
      <RateLimitGroup page={page} />
    </>
  )
}

function SaveAction({ page }: { page: PageState }) {
  return (
    <div className="px-screen flex justify-end">
      <Button
        className="h-11"
        onClick={() => void page.handleSave()}
        disabled={!page.hasChanges || page.updateMutation.isPending}
      >
        <Save className="mr-2 size-4" />
        Save Changes
      </Button>
    </div>
  )
}

export function RequestSettingsPage() {
  const page = useRequestSettingsPage()
  const back = usePushBack()

  if (page.isLoading) {
    return (
      <Screen title="Request Settings" back={back}>
        <SectionLoading count={3} />
      </Screen>
    )
  }

  if (page.isError) {
    return (
      <Screen title="Request Settings" back={back}>
        <SectionError onRetry={page.refetch} />
      </Screen>
    )
  }

  return (
    <Screen title="Request Settings" back={back}>
      <PortalAccessGroup page={page} />
      <PortalGroups page={page} />
      <SaveAction page={page} />
      <NotificationDialog
        open={page.showNotificationDialog}
        onOpenChange={page.setShowNotificationDialog}
        notification={page.editingNotification}
      />
    </Screen>
  )
}
