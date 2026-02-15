import { useState } from 'react'

import { Download, Edit, TestTube, Trash2 } from 'lucide-react'
import { toast } from 'sonner'

import { DownloadClientDialog } from '@/components/downloadclients/download-client-dialog'
import { ConfirmDialog } from '@/components/forms/confirm-dialog'
import { ListSection } from '@/components/settings/list-section'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import {
  useDeleteDownloadClient,
  useDownloadClients,
  useTestDownloadClient,
  useUpdateDownloadClient,
} from '@/hooks'
import type { DownloadClient } from '@/types'

const clientTypeLabels: Record<string, string> = {
  qbittorrent: 'qBittorrent',
  transmission: 'Transmission',
  sabnzbd: 'SABnzbd',
  nzbget: 'NZBGet',
}

type ClientCardActions = {
  onToggleEnabled: (id: number, enabled: boolean) => void
  onTest: (id: number) => void
  onEdit: (client: DownloadClient) => void
  onDelete: (id: number) => void
  isTestPending: boolean
}

function ClientCardInfo({ client }: { client: DownloadClient }) {
  return (
    <div className="flex items-center gap-4">
      <div className="bg-muted flex size-10 items-center justify-center rounded-lg">
        <Download className="size-5" />
      </div>
      <div>
        <div className="flex items-center gap-2">
          <CardTitle className="text-base">{client.name}</CardTitle>
          <Badge variant="outline">{clientTypeLabels[client.type] || client.type}</Badge>
          <Badge variant="secondary">Priority {client.priority}</Badge>
        </div>
        <CardDescription className="text-xs">
          {client.useSsl ? 'https' : 'http'}://{client.host}:{client.port}
        </CardDescription>
      </div>
    </div>
  )
}

function ClientCard({ client, actions }: { client: DownloadClient; actions: ClientCardActions }) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between py-4">
        <ClientCardInfo client={client} />
        <div className="flex items-center gap-4">
          <Switch checked={client.enabled} onCheckedChange={(checked) => actions.onToggleEnabled(client.id, checked)} />
          <Button variant="outline" size="sm" onClick={() => actions.onTest(client.id)} disabled={actions.isTestPending}>
            <TestTube className="mr-1 size-4" />
            Test
          </Button>
          <Button variant="ghost" size="icon" onClick={() => actions.onEdit(client)}>
            <Edit className="size-4" />
          </Button>
          <ConfirmDialog
            trigger={<Button variant="ghost" size="icon"><Trash2 className="size-4" /></Button>}
            title="Delete download client"
            description={`Are you sure you want to delete "${client.name}"?`}
            confirmLabel="Delete"
            variant="destructive"
            onConfirm={() => actions.onDelete(client.id)}
          />
        </div>
      </CardHeader>
    </Card>
  )
}

function useDownloadClientActions() {
  const [showDialog, setShowDialog] = useState(false)
  const [editingClient, setEditingClient] = useState<DownloadClient | null>(null)

  const query = useDownloadClients()
  const deleteMutation = useDeleteDownloadClient()
  const testMutation = useTestDownloadClient()
  const updateMutation = useUpdateDownloadClient()

  const handleToggleEnabled = (id: number, enabled: boolean) => {
    void (async () => {
      try { await updateMutation.mutateAsync({ id, data: { enabled } }); toast.success(enabled ? 'Client enabled' : 'Client disabled') }
      catch { toast.error('Failed to update client') }
    })()
  }

  const handleTest = (id: number) => {
    void (async () => {
      try {
        const result = await testMutation.mutateAsync(id)
        toast[result.success ? 'success' : 'error'](result.success ? 'Connection successful' : (result.message || 'Connection failed'))
      } catch { toast.error('Failed to test connection') }
    })()
  }

  const handleDelete = (id: number) => {
    void (async () => {
      try { await deleteMutation.mutateAsync(id); toast.success('Client deleted') }
      catch { toast.error('Failed to delete client') }
    })()
  }

  return {
    query, showDialog, setShowDialog, editingClient,
    handleOpenAdd: () => { setEditingClient(null); setShowDialog(true) },
    cardActions: {
      onToggleEnabled: handleToggleEnabled,
      onTest: handleTest,
      onEdit: (client: DownloadClient) => { setEditingClient(client); setShowDialog(true) },
      onDelete: handleDelete,
      isTestPending: testMutation.isPending,
    } satisfies ClientCardActions,
  }
}

export function DownloadClientsSection() {
  const s = useDownloadClientActions()
  const { data: clients, isLoading, isError, refetch } = s.query

  return (
    <>
      <ListSection
        data={clients}
        isLoading={isLoading}
        isError={isError}
        refetch={refetch}
        emptyIcon={<Download className="size-8" />}
        emptyTitle="No download clients configured"
        emptyDescription="Add a download client to start downloading"
        emptyAction={{ label: 'Add Client', onClick: s.handleOpenAdd }}
        renderItem={(client) => <ClientCard client={client} actions={s.cardActions} />}
        keyExtractor={(client) => client.id}
        addPlaceholder={{ label: 'Add Download Client', onClick: s.handleOpenAdd }}
      />
      <DownloadClientDialog open={s.showDialog} onOpenChange={s.setShowDialog} client={s.editingClient} />
    </>
  )
}
