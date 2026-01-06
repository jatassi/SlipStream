import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { defaultsApi } from '@/api'

export const defaultsKeys = {
  all: ['defaults'] as const,
  lists: () => [...defaultsKeys.all, 'list'] as const,
  list: () => [...defaultsKeys.all, 'list'] as const,
  listByEntityType: (entityType: string) => [...defaultsKeys.lists(), entityType] as const,
  details: () => [...defaultsKeys.all, 'detail'] as const,
  detail: (entityType: string, mediaType: string) => [...defaultsKeys.details(), entityType, mediaType] as const,
}

export const useDefaults = () =>
  useQuery({
    queryKey: defaultsKeys.all,
    queryFn: () => defaultsApi.getAll(),
  })

export const useDefaultsByEntityType = (entityType: string) =>
  useQuery({
    queryKey: defaultsKeys.listByEntityType(entityType),
    queryFn: () => defaultsApi.getByEntityType(entityType as any),
  })

export const useDefault = (entityType: string, mediaType: string) =>
  useQuery({
    queryKey: defaultsKeys.detail(entityType, mediaType),
    queryFn: () => defaultsApi.get(entityType as any, mediaType as any),
  })

export const useSetDefault = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ entityType, mediaType, entityId }: { entityType: string; mediaType: string; entityId: number }) =>
      defaultsApi.set(entityType as any, mediaType as any, entityId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: defaultsKeys.all })
    }
  })
}

export const useClearDefault = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ entityType, mediaType }: { entityType: string; mediaType: string }) =>
      defaultsApi.clear(entityType as any, mediaType as any),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: defaultsKeys.all })
    }
  })
}