import { useState } from 'react'

import { toast } from 'sonner'

import { useCreateRootFolder, useDeleteRootFolder, useRootFolders } from '@/hooks'
import { useClearDefault, useSetDefault } from '@/hooks/use-defaults'
import { withToast } from '@/lib/with-toast'
import type { MediaType } from '@/types'

export type NewFolderState = {
  name: string
  path: string
  mediaType: 'movie' | 'tv'
}

function useNewFolder() {
  const [state, setState] = useState<NewFolderState>({ name: '', path: '', mediaType: 'movie' })
  return {
    state,
    setName: (name: string) => setState((prev) => ({ ...prev, name })),
    setPath: (path: string) => setState((prev) => ({ ...prev, path })),
    setMediaType: (mediaType: 'movie' | 'tv') => setState((prev) => ({ ...prev, mediaType })),
    reset: () => setState({ name: '', path: '', mediaType: 'movie' }),
  }
}

function useFolderMutations() {
  const createMutation = useCreateRootFolder()
  const deleteMutation = useDeleteRootFolder()
  const setDefaultMutation = useSetDefault()
  const clearDefaultMutation = useClearDefault()

  return {
    createMutation,
    handleDelete: (id: number) =>
      void withToast(async () => {
        await deleteMutation.mutateAsync(id)
        toast.success('Root folder deleted')
      })(),
    handleSetDefault: (id: number, mediaType: MediaType) =>
      void withToast(async () => {
        await setDefaultMutation.mutateAsync({ entityType: 'root_folder', mediaType, entityId: id })
        toast.success(`Default ${mediaType} root folder set`)
      })(),
    handleClearDefault: (mediaType: MediaType) =>
      void withToast(async () => {
        await clearDefaultMutation.mutateAsync({ entityType: 'root_folder', mediaType })
        toast.success(`Default ${mediaType} root folder cleared`)
      })(),
  }
}

export function useRootFoldersPage() {
  const [showAddDialog, setShowAddDialog] = useState(false)
  const [showBrowser, setShowBrowser] = useState(false)
  const newFolder = useNewFolder()
  const query = useRootFolders()
  const mutations = useFolderMutations()

  const handleAdd = () => {
    if (!newFolder.state.path.trim()) {
      toast.error('Please enter a path')
      return
    }
    void withToast(async () => {
      await mutations.createMutation.mutateAsync({
        path: newFolder.state.path,
        name: newFolder.state.name.trim(),
        mediaType: newFolder.state.mediaType,
      })
      toast.success('Root folder added')
      setShowAddDialog(false)
      newFolder.reset()
    })()
  }

  return {
    query,
    showAddDialog,
    setShowAddDialog,
    showBrowser,
    setShowBrowser,
    newFolder,
    handleAdd,
    isPending: mutations.createMutation.isPending,
    handleDelete: mutations.handleDelete,
    handleSetDefault: mutations.handleSetDefault,
    handleClearDefault: mutations.handleClearDefault,
  }
}
