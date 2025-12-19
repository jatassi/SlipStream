import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { indexersApi } from '@/api'
import type { Indexer, CreateIndexerInput, UpdateIndexerInput } from '@/types'

export const indexerKeys = {
  all: ['indexers'] as const,
  lists: () => [...indexerKeys.all, 'list'] as const,
  list: () => [...indexerKeys.lists()] as const,
  details: () => [...indexerKeys.all, 'detail'] as const,
  detail: (id: number) => [...indexerKeys.details(), id] as const,
}

export function useIndexers() {
  return useQuery({
    queryKey: indexerKeys.list(),
    queryFn: () => indexersApi.list(),
  })
}

export function useIndexer(id: number) {
  return useQuery({
    queryKey: indexerKeys.detail(id),
    queryFn: () => indexersApi.get(id),
    enabled: !!id,
  })
}

export function useCreateIndexer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateIndexerInput) => indexersApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: indexerKeys.all })
    },
  })
}

export function useUpdateIndexer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateIndexerInput }) =>
      indexersApi.update(id, data),
    onSuccess: (indexer: Indexer) => {
      queryClient.invalidateQueries({ queryKey: indexerKeys.all })
      queryClient.setQueryData(indexerKeys.detail(indexer.id), indexer)
    },
  })
}

export function useDeleteIndexer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => indexersApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: indexerKeys.all })
    },
  })
}

export function useTestIndexer() {
  return useMutation({
    mutationFn: (id: number) => indexersApi.test(id),
  })
}

export function useTestNewIndexer() {
  return useMutation({
    mutationFn: (data: CreateIndexerInput) => indexersApi.testNew(data),
  })
}
