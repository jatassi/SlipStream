import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { defaultsApi } from '@/api'
import type { EntityType, MediaType } from '@/api/defaults'

const defaultsKeys = {
  all: ['defaults'] as const,
  lists: () => [...defaultsKeys.all, 'list'] as const,
  list: () => [...defaultsKeys.all, 'list'] as const,
  listByEntityType: (entityType: string) => [...defaultsKeys.lists(), entityType] as const,
  details: () => [...defaultsKeys.all, 'detail'] as const,
  detail: (entityType: string, mediaType: string) =>
    [...defaultsKeys.details(), entityType, mediaType] as const,
}

export const useDefault = (entityType: EntityType, mediaType: MediaType) =>
  useQuery({
    queryKey: defaultsKeys.detail(entityType, mediaType),
    queryFn: () => defaultsApi.get(entityType, mediaType),
  })

export const useSetDefault = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      entityType,
      mediaType,
      entityId,
    }: {
      entityType: EntityType
      mediaType: MediaType
      entityId: number
    }) => defaultsApi.set(entityType, mediaType, entityId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: defaultsKeys.all })
    },
  })
}

export const useClearDefault = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ entityType, mediaType }: { entityType: EntityType; mediaType: MediaType }) =>
      defaultsApi.clear(entityType, mediaType),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: defaultsKeys.all })
    },
  })
}
