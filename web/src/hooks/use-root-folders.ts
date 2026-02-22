import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { rootFoldersApi } from '@/api'
import { createQueryKeys } from '@/lib/query-keys'
import type { CreateRootFolderInput } from '@/types'

const baseKeys = createQueryKeys('rootFolders')
const rootFolderKeys = {
  ...baseKeys,
  listByType: (mediaType: 'movie' | 'tv') => [...baseKeys.lists(), mediaType] as const,
}

export function useRootFolders() {
  return useQuery({
    queryKey: rootFolderKeys.list(),
    queryFn: () => rootFoldersApi.list(),
  })
}

export function useRootFoldersByType(mediaType: 'movie' | 'tv') {
  return useQuery({
    queryKey: rootFolderKeys.listByType(mediaType),
    queryFn: () => rootFoldersApi.listByType(mediaType),
  })
}

export function useCreateRootFolder() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateRootFolderInput) => rootFoldersApi.create(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: rootFolderKeys.all })
    },
  })
}

export function useDeleteRootFolder() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => rootFoldersApi.delete(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: rootFolderKeys.all })
      void queryClient.invalidateQueries({ queryKey: ['defaults'] })
    },
  })
}

