import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { indexersApi } from '@/api'
import { createQueryKeys } from '@/lib/query-keys'
import type {
  CreateIndexerInput,
  Indexer,
  TestConfigInput,
  UpdateIndexerInput,
} from '@/types'

const indexerKeys = createQueryKeys('indexers')

const definitionKeys = {
  all: ['definitions'] as const,
  list: () => ['definitions', 'list'] as const,
  schemas: () => [...definitionKeys.all, 'schema'] as const,
  schema: (id: string) => [...definitionKeys.schemas(), id] as const,
}

// Indexer hooks
export function useIndexers() {
  return useQuery({
    queryKey: indexerKeys.list(),
    queryFn: () => indexersApi.list(),
  })
}

export function useCreateIndexer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateIndexerInput) => indexersApi.create(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: indexerKeys.all })
    },
  })
}

export function useUpdateIndexer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateIndexerInput }) =>
      indexersApi.update(id, data),
    onSuccess: (indexer: Indexer) => {
      void queryClient.invalidateQueries({ queryKey: indexerKeys.all })
      queryClient.setQueryData(indexerKeys.detail(indexer.id), indexer)

    },
  })
}

export function useDeleteIndexer() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => indexersApi.delete(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: indexerKeys.all })
    },
  })
}

export function useTestIndexer() {
  return useMutation({
    mutationFn: (id: number) => indexersApi.test(id),
  })
}

export function useTestIndexerConfig() {
  return useMutation({
    mutationFn: (data: TestConfigInput) => indexersApi.testConfig(data),
  })
}

// Definition hooks
export function useDefinitions() {
  return useQuery({
    queryKey: definitionKeys.list(),
    queryFn: () => indexersApi.listDefinitions(),
  })
}

export function useDefinitionSchema(id: string) {
  return useQuery({
    queryKey: definitionKeys.schema(id),
    queryFn: () => indexersApi.getDefinitionSchema(id),
    enabled: !!id,
  })
}

