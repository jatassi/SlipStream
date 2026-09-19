import { Download } from 'lucide-react'

import { DownloadClientDialog } from '@/components/downloadclients/download-client-dialog'
import { clientTypeConfigs } from '@/components/downloadclients/use-download-client-dialog'
import { IconTile } from '@/components/grouped-list'
import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'
import type { SettingsRowAction } from '@/components/settings/settings-item-row'
import { SettingsItemRow } from '@/components/settings/settings-item-row'
import { AddAction, SettingsList } from '@/components/settings/settings-list'
import type { DownloadClient } from '@/types'

import { useDownloadClientsPage } from './use-download-clients-page'

type PageState = ReturnType<typeof useDownloadClientsPage>

export function DownloadClientsPage() {
  const page = useDownloadClientsPage()
  const back = usePushBack()
  const { data: clients, isLoading, isError, refetch } = page.query

  return (
    <Screen
      title="Download Clients"
      back={back}
      trailing={<AddAction label="Add download client" onClick={page.handleAdd} />}
    >
      <SettingsList
        state={{ isLoading, isError, isEmpty: !clients?.length, refetch }}
        empty="No download clients yet"
      >
        {clients?.map((client) => (
          <ClientRow key={client.id} client={client} page={page} />
        ))}
      </SettingsList>
      <DownloadClientDialog
        open={page.dialogOpen}
        onOpenChange={page.setDialogOpen}
        client={page.editingClient}
      />
    </Screen>
  )
}

function ClientRow({ client, page }: { client: DownloadClient; page: PageState }) {
  const actions: SettingsRowAction[] = [
    { label: 'Test', onClick: () => page.handleTest(client.id) },
    {
      label: 'Delete',
      destructive: true,
      onClick: () => page.handleDelete(client.id),
      confirm: {
        title: 'Delete download client',
        description: `Are you sure you want to delete "${client.name}"?`,
      },
    },
  ]

  return (
    <SettingsItemRow
      leading={
        <IconTile className="bg-emerald-600">
          <Download />
        </IconTile>
      }
      title={client.name}
      subtitle={clientSubtitle(client)}
      onOpen={() => page.handleEdit(client)}
      openLabel={`Edit ${client.name}`}
      toggle={{
        label: `${client.name} enabled`,
        checked: client.enabled,
        onCheckedChange: (checked) => page.handleToggleEnabled(client, checked),
      }}
      actions={actions}
    />
  )
}

function clientSubtitle(client: DownloadClient): string {
  const urlBase = client.urlBase && client.urlBase !== '/' ? client.urlBase : ''
  const address = `${client.useSsl ? 'https' : 'http'}://${client.host}:${client.port}${urlBase}`
  return `${clientTypeConfigs[client.type].label} · ${address}`
}
