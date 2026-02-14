import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { rootFoldersApi } from '@/api'
import type { CreateRootFolderInput } from '@/types'

export const rootFolderKeys = {
  all: ['rootFolders'] as const,
  lists: () => [...rootFolderKeys.all, 'list'] as const,
  list: () => [...rootFolderKeys.lists()] as const,
  listByType: (mediaType: 'movie' | 'tv') => [...rootFolderKeys.lists(), mediaType] as const,
  details: () => [...rootFolderKeys.all, 'detail'] as const,
  detail: (id: number) => [...rootFolderKeys.details(), id] as const,
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

export function useRootFolder(id: number) {
  return useQuery({
    queryKey: rootFolderKeys.detail(id),
    queryFn: () => rootFoldersApi.get(id),
    enabled: !!id,
  })
}

export function useCreateRootFolder() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateRootFolderInput) => rootFoldersApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: rootFolderKeys.all })
    },
  })
}

export function useDeleteRootFolder() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => rootFoldersApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: rootFolderKeys.all })
      queryClient.invalidateQueries({ queryKey: ['defaults'] })
    },
  })
}

export function useRefreshRootFolder() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => rootFoldersApi.refresh(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: rootFolderKeys.all })
    },
  })
}
