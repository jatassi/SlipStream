import { useState } from 'react'

import { Edit, Globe, Lock, Rss, TestTube, Trash2, Unlock } from 'lucide-react'
import { toast } from 'sonner'

import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { ConfirmDialog } from '@/components/forms/confirm-dialog'
import {
  IndexerDialog,
  IndexerModeToggle,
  ProwlarrConfigForm,
  ProwlarrIndexerList,
} from '@/components/indexers'
import { AddPlaceholderCard } from '@/components/settings/add-placeholder-card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Switch } from '@/components/ui/switch'
import {
  useDeleteIndexer,
  useIndexerMode,
  useIndexers,
  useTestIndexer,
  useUpdateIndexer,
} from '@/hooks'
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

function useIndexerActions() {
  const deleteMutation = useDeleteIndexer()
  const testMutation = useTestIndexer()
  const updateMutation = useUpdateIndexer()

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

  return { handleToggleEnabled, handleTest, handleDelete, isTestPending: testMutation.isPending }
}

export function IndexersSection() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingIndexer, setEditingIndexer] = useState<Indexer | null>(null)
  const { data: modeInfo, isLoading: modeLoading } = useIndexerMode()
  const { data: indexers, isLoading, isError, refetch } = useIndexers()
  const actions = useIndexerActions()
  const isProwlarrMode = modeInfo?.effectiveMode === 'prowlarr'

  const handleAdd = () => {
    setEditingIndexer(null)
    setDialogOpen(true)
  }

  const handleEdit = (indexer: Indexer) => {
    setEditingIndexer(indexer)
    setDialogOpen(true)
  }

  if (modeLoading) {
    return <LoadingState variant="list" count={3} />
  }

  return (
    <div className="space-y-6">
      <IndexerModeToggle />
      {isProwlarrMode ? (
        <ProwlarrModeContent />
      ) : (
        <SlipStreamModeContent
          indexers={indexers}
          isLoading={isLoading}
          isError={isError}
          refetch={refetch}
          onAdd={handleAdd}
          onEdit={handleEdit}
          actions={actions}
        />
      )}
      <IndexerDialog open={dialogOpen} onOpenChange={setDialogOpen} indexer={editingIndexer} />
    </div>
  )
}

function ProwlarrModeContent() {
  return (
    <div className="space-y-6">
      <ProwlarrConfigForm />
      <ProwlarrIndexerList />
    </div>
  )
}

type IndexerActions = {
  handleToggleEnabled: (id: number, enabled: boolean) => Promise<void>
  handleTest: (id: number) => Promise<void>
  handleDelete: (id: number) => Promise<void>
  isTestPending: boolean
}

type SlipStreamModeContentProps = {
  indexers: Indexer[] | undefined
  isLoading: boolean
  isError: boolean
  refetch: () => void
  onAdd: () => void
  onEdit: (indexer: Indexer) => void
  actions: IndexerActions
}

function SlipStreamModeContent(props: SlipStreamModeContentProps) {
  const { indexers, isLoading, isError, refetch, onAdd, onEdit, actions } = props

  if (isLoading) {
    return <LoadingState variant="list" count={3} />
  }
  if (isError) {
    return <ErrorState onRetry={refetch} />
  }
  if (!indexers?.length) {
    return <AddPlaceholderCard label="Add Indexer" onClick={onAdd} />
  }

  return (
    <div className="space-y-4">
      {indexers.map((indexer) => (
        <IndexerCard key={indexer.id} indexer={indexer} onEdit={onEdit} actions={actions} />
      ))}
      <AddPlaceholderCard label="Add Indexer" onClick={onAdd} />
    </div>
  )
}

type IndexerCardProps = {
  indexer: Indexer
  onEdit: (indexer: Indexer) => void
  actions: IndexerActions
}

function IndexerCard({ indexer, onEdit, actions }: IndexerCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between py-4">
        <div className="flex items-center gap-4">
          <div className="bg-muted flex size-10 items-center justify-center rounded-lg">
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
            <IndexerDescription indexer={indexer} />
          </div>
        </div>
        <IndexerActions indexer={indexer} onEdit={onEdit} actions={actions} />
      </CardHeader>
    </Card>
  )
}

function IndexerDescription({ indexer }: { indexer: Indexer }) {
  return (
    <CardDescription className="flex items-center gap-2 text-xs">
      <span>{indexer.definitionId}</span>
      <span className="text-muted-foreground/50">|</span>
      {indexer.supportsMovies ? <span>Movies</span> : null}
      {indexer.supportsMovies && indexer.supportsTv ? (
        <span className="text-muted-foreground/50">/</span>
      ) : null}
      {indexer.supportsTv ? <span>TV</span> : null}
      <span className="text-muted-foreground/50">|</span>
      <span>Priority: {indexer.priority}</span>
      {!indexer.autoSearchEnabled && (
        <>
          <span className="text-muted-foreground/50">|</span>
          <span className="text-yellow-500">Manual search only</span>
        </>
      )}
      {!indexer.rssEnabled && (
        <>
          <span className="text-muted-foreground/50">|</span>
          <span className="text-yellow-500">No RSS</span>
        </>
      )}
    </CardDescription>
  )
}

function IndexerActions(props: {
  indexer: Indexer
  onEdit: (indexer: Indexer) => void
  actions: IndexerActions
}) {
  const { indexer, onEdit, actions } = props

  return (
    <div className="flex items-center gap-4">
      <Switch
        checked={indexer.enabled}
        onCheckedChange={(checked) => actions.handleToggleEnabled(indexer.id, checked)}
      />
      <Button
        variant="outline"
        size="sm"
        onClick={() => actions.handleTest(indexer.id)}
        disabled={actions.isTestPending}
      >
        <TestTube className="mr-1 size-4" />
        Test
      </Button>
      <Button variant="ghost" size="icon" onClick={() => onEdit(indexer)}>
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
        onConfirm={() => actions.handleDelete(indexer.id)}
      />
    </div>
  )
}
