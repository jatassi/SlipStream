import { useState } from 'react'
import { Plus, Edit, Trash2, Rss, TestTube, Globe, Lock, Unlock } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { LoadingState } from '@/components/data/LoadingState'
import { EmptyState } from '@/components/data/EmptyState'
import { ErrorState } from '@/components/data/ErrorState'
import { ConfirmDialog } from '@/components/forms/ConfirmDialog'
import { IndexerDialog } from '@/components/indexers/IndexerDialog'
import {
  useIndexers,
  useDeleteIndexer,
  useTestIndexer,
  useUpdateIndexer,
} from '@/hooks'
import { toast } from 'sonner'
import type { Indexer, Privacy, Protocol } from '@/types'

const privacyIcons: Record<Privacy, React.ReactNode> = {
  public: <Globe className="size-3" />,
  'semi-private': <Unlock className="size-3" />,
  private: <Lock className="size-3" />,
}

const privacyColors: Record<Privacy, string> = {
  public: 'bg-green-500/10 text-green-500 hover:bg-green-500/20',
  'semi-private': 'bg-yellow-500/10 text-yellow-500 hover:bg-yellow-500/20',
  private: 'bg-red-500/10 text-red-500 hover:bg-red-500/20',
}

const protocolColors: Record<Protocol, string> = {
  torrent: 'bg-blue-500/10 text-blue-500 hover:bg-blue-500/20',
  usenet: 'bg-purple-500/10 text-purple-500 hover:bg-purple-500/20',
}

export function IndexersPage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingIndexer, setEditingIndexer] = useState<Indexer | null>(null)

  const { data: indexers, isLoading, isError, refetch } = useIndexers()
  const deleteMutation = useDeleteIndexer()
  const testMutation = useTestIndexer()
  const updateMutation = useUpdateIndexer()

  const handleAdd = () => {
    setEditingIndexer(null)
    setDialogOpen(true)
  }

  const handleEdit = (indexer: Indexer) => {
    setEditingIndexer(indexer)
    setDialogOpen(true)
  }

  const handleToggleEnabled = async (id: number, enabled: boolean) => {
    try {
      await updateMutation.mutateAsync({ id, data: { enabled } })
      toast.success(enabled ? 'Indexer enabled' : 'Indexer disabled')
    } catch {
      toast.error('Failed to update indexer')
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
      toast.success('Indexer deleted')
    } catch {
      toast.error('Failed to delete indexer')
    }
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Indexers" />
        <LoadingState variant="list" count={3} />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Indexers" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="Indexers"
        description="Configure search providers for finding releases"
        breadcrumbs={[
          { label: 'Settings', href: '/settings' },
          { label: 'Indexers' },
        ]}
        actions={
          <Button onClick={handleAdd}>
            <Plus className="size-4 mr-2" />
            Add Indexer
          </Button>
        }
      />

      {!indexers?.length ? (
        <EmptyState
          icon={<Rss className="size-8" />}
          title="No indexers configured"
          description="Add an indexer to search for releases"
          action={{ label: 'Add Indexer', onClick: handleAdd }}
        />
      ) : (
        <div className="space-y-4">
          {indexers.map((indexer) => (
            <Card key={indexer.id}>
              <CardHeader className="flex flex-row items-center justify-between py-4">
                <div className="flex items-center gap-4">
                  <div className="flex size-10 items-center justify-center rounded-lg bg-muted">
                    <Rss className="size-5" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <CardTitle className="text-base">{indexer.name}</CardTitle>
                      <Badge variant="secondary" className={protocolColors[indexer.protocol]}>
                        {indexer.protocol}
                      </Badge>
                      <Badge variant="secondary" className={privacyColors[indexer.privacy]}>
                        <span className="mr-1">{privacyIcons[indexer.privacy]}</span>
                        {indexer.privacy}
                      </Badge>
                    </div>
                    <CardDescription className="text-xs flex items-center gap-2">
                      <span>{indexer.definitionId}</span>
                      <span className="text-muted-foreground/50">|</span>
                      {indexer.supportsMovies && <span>Movies</span>}
                      {indexer.supportsMovies && indexer.supportsTv && (
                        <span className="text-muted-foreground/50">/</span>
                      )}
                      {indexer.supportsTv && <span>TV</span>}
                      <span className="text-muted-foreground/50">|</span>
                      <span>Priority: {indexer.priority}</span>
                      {!indexer.autoSearchEnabled && (
                        <>
                          <span className="text-muted-foreground/50">|</span>
                          <span className="text-yellow-500">Manual search only</span>
                        </>
                      )}
                    </CardDescription>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <Switch
                    checked={indexer.enabled}
                    onCheckedChange={(checked) => handleToggleEnabled(indexer.id, checked)}
                  />
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleTest(indexer.id)}
                    disabled={testMutation.isPending}
                  >
                    <TestTube className="size-4 mr-1" />
                    Test
                  </Button>
                  <Button variant="ghost" size="icon" onClick={() => handleEdit(indexer)}>
                    <Edit className="size-4" />
                  </Button>
                  <ConfirmDialog
                    trigger={
                      <Button variant="ghost" size="icon">
                        <Trash2 className="size-4" />
                      </Button>
                    }
                    title="Delete indexer"
                    description={`Are you sure you want to delete "${indexer.name}"?`}
                    confirmLabel="Delete"
                    variant="destructive"
                    onConfirm={() => handleDelete(indexer.id)}
                  />
                </div>
              </CardHeader>
            </Card>
          ))}
        </div>
      )}

      <IndexerDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        indexer={editingIndexer}
      />
    </div>
  )
}
