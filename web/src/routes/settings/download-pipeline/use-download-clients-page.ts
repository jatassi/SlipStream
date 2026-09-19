import { useState } from 'react'

import { toast } from 'sonner'

import {
  useDeleteDownloadClient,
  useDownloadClients,
  useTestDownloadClient,
  useUpdateDownloadClient,
} from '@/hooks'
import type { DownloadClient } from '@/types'

function useClientMutations() {
  const deleteMutation = useDeleteDownloadClient()
  const testMutation = useTestDownloadClient()
  const updateMutation = useUpdateDownloadClient()

  const handleToggleEnabled = (client: DownloadClient, enabled: boolean) => {
    const { id, createdAt: _createdAt, updatedAt: _updatedAt, ...data } = client
    void (async () => {
      try {
        await updateMutation.mutateAsync({ id, data: { ...data, enabled } })
        toast.success(enabled ? 'Client enabled' : 'Client disabled')
      } catch {
        toast.error('Failed to update client')
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
        toast.success('Client deleted')
      } catch {
        toast.error('Failed to delete client')
      }
    })()
  }

  return { handleToggleEnabled, handleTest, handleDelete }
}

export function useDownloadClientsPage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingClient, setEditingClient] = useState<DownloadClient | null>(null)
  const query = useDownloadClients()
  const mutations = useClientMutations()

  return {
    query,
    dialogOpen,
    setDialogOpen,
    editingClient,
    handleAdd: () => {
      setEditingClient(null)
      setDialogOpen(true)
    },
    handleEdit: (client: DownloadClient) => {
      setEditingClient(client)
      setDialogOpen(true)
    },
    ...mutations,
  }
}
