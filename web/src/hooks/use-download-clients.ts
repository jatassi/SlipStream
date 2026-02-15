import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { downloadClientsApi } from '@/api'
import type { CreateDownloadClientInput, DownloadClient, UpdateDownloadClientInput } from '@/types'

export const downloadClientKeys = {
  all: ['downloadClients'] as const,
  lists: () => [...downloadClientKeys.all, 'list'] as const,
  list: () => [...downloadClientKeys.lists()] as const,
  details: () => [...downloadClientKeys.all, 'detail'] as const,
  detail: (id: number) => [...downloadClientKeys.details(), id] as const,
}

export function useDownloadClients() {
  return useQuery({
    queryKey: downloadClientKeys.list(),
    queryFn: () => downloadClientsApi.list(),
  })
}

export function useDownloadClient(id: number) {
  return useQuery({
    queryKey: downloadClientKeys.detail(id),
    queryFn: () => downloadClientsApi.get(id),
    enabled: !!id,
  })
}

export function useCreateDownloadClient() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateDownloadClientInput) => downloadClientsApi.create(data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: downloadClientKeys.all })
    },
  })
}

export function useUpdateDownloadClient() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateDownloadClientInput }) =>
      downloadClientsApi.update(id, data),
    onSuccess: (client: DownloadClient) => {
      void queryClient.invalidateQueries({ queryKey: downloadClientKeys.all })
      queryClient.setQueryData(downloadClientKeys.detail(client.id), client)
    },
  })
}

export function useDeleteDownloadClient() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => downloadClientsApi.delete(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: downloadClientKeys.all })
    },
  })
}

export function useTestDownloadClient() {
  return useMutation({
    mutationFn: (id: number) => downloadClientsApi.test(id),
  })
}

export function useTestNewDownloadClient() {
  return useMutation({
    mutationFn: (data: CreateDownloadClientInput) => downloadClientsApi.testNew(data),
  })
}
