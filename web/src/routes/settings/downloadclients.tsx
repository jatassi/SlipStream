import { Plus, Edit, Trash2, Download, TestTube } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import {
  useDownloadClients,
  useDeleteDownloadClient,
  useTestDownloadClient,
  useUpdateDownloadClient,
} from '@/hooks'
import { toast } from 'sonner'

const clientTypeLabels: Record<string, string> = {
  qbittorrent: 'qBittorrent',
  transmission: 'Transmission',
  sabnzbd: 'SABnzbd',
  nzbget: 'NZBGet',
}

export function DownloadClientsPage() {
  const { data: clients, isLoading, isError, refetch } = useDownloadClients()
  const deleteMutation = useDeleteDownloadClient()
  const testMutation = useTestDownloadClient()
  const updateMutation = useUpdateDownloadClient()

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

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Download Clients" />
        <LoadingState variant="list" count={3} />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Download Clients" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="Download Clients"
        description="Configure torrent and usenet clients"
        breadcrumbs={[
          { label: 'Settings', href: '/settings' },
          { label: 'Download Clients' },
        ]}
        actions={
          <Button>
            <Plus className="size-4 mr-2" />
            Add Client
          </Button>
        }
      />

      {!clients?.length ? (
        <EmptyState
          icon={<Download className="size-8" />}
          title="No download clients configured"
          description="Add a download client to start downloading"
          action={{ label: 'Add Client', onClick: () => {} }}
        />
      ) : (
        <div className="space-y-4">
          {clients.map((client) => (
            <Card key={client.id}>
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
                  <Button variant="ghost" size="icon">
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
          ))}
        </div>
      )}
    </div>
  )
}
