import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { indexersApi } from '@/api'
import type {
  CreateIndexerInput,
  DefinitionFilters,
  Indexer,
  TestConfigInput,
  UpdateIndexerInput,
} from '@/types'

export const indexerKeys = {
  all: ['indexers'] as const,
  lists: () => [...indexerKeys.all, 'list'] as const,
  list: () => [...indexerKeys.lists()] as const,
  details: () => [...indexerKeys.all, 'detail'] as const,
  detail: (id: number) => [...indexerKeys.details(), id] as const,
  statuses: () => [...indexerKeys.all, 'status'] as const,
  status: (id: number) => [...indexerKeys.statuses(), id] as const,
}

export const definitionKeys = {
  all: ['definitions'] as const,
  lists: () => [...definitionKeys.all, 'list'] as const,
  list: () => [...definitionKeys.lists()] as const,
  search: (query?: string, filters?: DefinitionFilters) =>
    [...definitionKeys.all, 'search', { query, filters }] as const,
  details: () => [...definitionKeys.all, 'detail'] as const,
  detail: (id: string) => [...definitionKeys.details(), id] as const,
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

export function useTestIndexerConfig() {
  return useMutation({
    mutationFn: (data: TestConfigInput) => indexersApi.testConfig(data),
  })
}

// Indexer status hooks
export function useIndexerStatus(id: number) {
  return useQuery({
    queryKey: indexerKeys.status(id),
    queryFn: () => indexersApi.getStatus(id),
    enabled: !!id,
  })
}

export function useIndexerStatuses() {
  return useQuery({
    queryKey: indexerKeys.statuses(),
    queryFn: () => indexersApi.getAllStatuses(),
  })
}

// Definition hooks
export function useDefinitions() {
  return useQuery({
    queryKey: definitionKeys.list(),
    queryFn: () => indexersApi.listDefinitions(),
  })
}

export function useSearchDefinitions(query?: string, filters?: DefinitionFilters) {
  return useQuery({
    queryKey: definitionKeys.search(query, filters),
    queryFn: () => indexersApi.searchDefinitions(query, filters),
  })
}

export function useDefinition(id: string) {
  return useQuery({
    queryKey: definitionKeys.detail(id),
    queryFn: () => indexersApi.getDefinition(id),
    enabled: !!id,
  })
}

export function useDefinitionSchema(id: string) {
  return useQuery({
    queryKey: definitionKeys.schema(id),
    queryFn: () => indexersApi.getDefinitionSchema(id),
    enabled: !!id,
  })
}

export function useUpdateDefinitions() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => indexersApi.updateDefinitions(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: definitionKeys.all })
    },
  })
}
