import { useState } from 'react'

import { toast } from 'sonner'

import {
  useDeleteIndexer,
  useIndexerMode,
  useIndexers,
  useTestIndexer,
  useUpdateIndexer,
} from '@/hooks'
import type { Indexer } from '@/types'

function useIndexerMutations() {
  const deleteMutation = useDeleteIndexer()
  const testMutation = useTestIndexer()
  const updateMutation = useUpdateIndexer()

  const handleToggleEnabled = (id: number, enabled: boolean) => {
    void (async () => {
      try {
        await updateMutation.mutateAsync({ id, data: { enabled } })
        toast.success(enabled ? 'Indexer enabled' : 'Indexer disabled')
      } catch {
        toast.error('Failed to update indexer')
      }
    })()
  }

  const handleTest = (id: number) => {
    void (async () => {
      try {
        const result = await testMutation.mutateAsync(id)
        toast[result.success ? 'success' : 'error'](
          result.success ? 'Connection successful' : result.message || 'Connection failed',
        )
      } catch {
        toast.error('Failed to test connection')
      }
    })()
  }

  const handleDelete = (id: number) => {
    void (async () => {
      try {
        await deleteMutation.mutateAsync(id)
        toast.success('Indexer deleted')
      } catch {
        toast.error('Failed to delete indexer')
      }
    })()
  }

  return { handleToggleEnabled, handleTest, handleDelete }
}

export function useIndexersPage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingIndexer, setEditingIndexer] = useState<Indexer | null>(null)
  const { data: modeInfo, isLoading: modeLoading } = useIndexerMode()
  const query = useIndexers()
  const mutations = useIndexerMutations()

  return {
    query,
    modeLoading,
    isProwlarrMode: modeInfo?.effectiveMode === 'prowlarr',
    dialogOpen,
    setDialogOpen,
    editingIndexer,
    handleAdd: () => {
      setEditingIndexer(null)
      setDialogOpen(true)
    },
    handleEdit: (indexer: Indexer) => {
      setEditingIndexer(indexer)
      setDialogOpen(true)
    },
    ...mutations,
  }
}
