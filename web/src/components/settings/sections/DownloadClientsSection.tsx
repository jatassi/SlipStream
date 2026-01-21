import { useState } from 'react'
import { Edit, Trash2, Download, TestTube } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { DownloadClientDialog } from '@/components/downloadclients/DownloadClientDialog'
import { ListSection } from '@/components/settings/ListSection'
import {
  useDownloadClients,
  useDeleteDownloadClient,
  useTestDownloadClient,
  useUpdateDownloadClient,
} from '@/hooks'
import { toast } from 'sonner'
import type { DownloadClient } from '@/types'

const clientTypeLabels: Record<string, string> = {
  qbittorrent: 'qBittorrent',
  transmission: 'Transmission',
  sabnzbd: 'SABnzbd',
  nzbget: 'NZBGet',
}

export function DownloadClientsSection() {
  const [showDialog, setShowDialog] = useState(false)
  const [editingClient, setEditingClient] = useState<DownloadClient | null>(null)

  const { data: clients, isLoading, isError, refetch } = useDownloadClients()
  const deleteMutation = useDeleteDownloadClient()
  const testMutation = useTestDownloadClient()
  const updateMutation = useUpdateDownloadClient()

  const handleOpenAdd = () => {
    setEditingClient(null)
    setShowDialog(true)
  }

  const handleOpenEdit = (client: DownloadClient) => {
    setEditingClient(client)
    setShowDialog(true)
  }

  const handleToggleEnabled = async (id: number, enabled: boolean) => {
    try {
      await updateMutation.mutateAsync({ id, data: { enabled } })
      toast.success(enabled ? 'Client enabled' : 'Client disabled')
    } catch {
      toast.error('Failed to update client')
    }
  }

  const handleTest = async (id: number) => {
    try {
      const result = await testMutation.mutateAsync(id)
      if (result.success) {
        toast.success('Connection successful')
      } else {
        toast.error(result.message || 'Connection failed')
      }
    } catch {
      toast.error('Failed to test connection')
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await deleteMutation.mutateAsync(id)
      toast.success('Client deleted')
    } catch {
      toast.error('Failed to delete client')
    }
  }

  const renderClient = (client: DownloadClient) => (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between py-4">
        <div className="flex items-center gap-4">
          <div className="flex size-10 items-center justify-center rounded-lg bg-muted">
            <Download className="size-5" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <CardTitle className="text-base">{client.name}</CardTitle>
              <Badge variant="outline">
                {clientTypeLabels[client.type] || client.type}
              </Badge>
              <Badge variant="secondary">Priority {client.priority}</Badge>
            </div>
            <CardDescription className="text-xs">
              {client.useSsl ? 'https' : 'http'}://{client.host}:{client.port}
            </CardDescription>
          </div>
        </div>
        <div className="flex items-center gap-4">
          <Switch
            checked={client.enabled}
            onCheckedChange={(checked) => handleToggleEnabled(client.id, checked)}
          />
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleTest(client.id)}
            disabled={testMutation.isPending}
          >
            <TestTube className="size-4 mr-1" />
            Test
          </Button>
          <Button variant="ghost" size="icon" onClick={() => handleOpenEdit(client)}>
            <Edit className="size-4" />
          </Button>
          <ConfirmDialog
            trigger={
              <Button variant="ghost" size="icon">
                <Trash2 className="size-4" />
              </Button>
            }
            title="Delete download client"
            description={`Are you sure you want to delete "${client.name}"?`}
            confirmLabel="Delete"
            variant="destructive"
            onConfirm={() => handleDelete(client.id)}
          />
        </div>
      </CardHeader>
    </Card>
  )

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
        emptyAction={{ label: 'Add Client', onClick: handleOpenAdd }}
        renderItem={renderClient}
        keyExtractor={(client) => client.id}
        addPlaceholder={{ label: 'Add Download Client', onClick: handleOpenAdd }}
      />

      <DownloadClientDialog
        open={showDialog}
        onOpenChange={setShowDialog}
        client={editingClient}
      />
    </>
  )
}
