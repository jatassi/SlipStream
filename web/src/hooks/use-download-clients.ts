import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { downloadClientsApi } from '@/api'
import { createQueryKeys } from '@/lib/query-keys'
import type { CreateDownloadClientInput, DownloadClient, UpdateDownloadClientInput } from '@/types'

const downloadClientKeys = createQueryKeys('downloadClients')

export function useDownloadClients() {
  return useQuery({
    queryKey: downloadClientKeys.list(),
    queryFn: () => downloadClientsApi.list(),
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
