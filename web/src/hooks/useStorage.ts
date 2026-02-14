import { useQuery } from '@tanstack/react-query'

import { storageApi } from '@/api/storage'
import type { StorageInfo } from '@/types/storage'

export function useStorage() {
  return useQuery<StorageInfo[]>({
    queryKey: ['storage'],
    queryFn: storageApi.getStorage,
    refetchInterval: 60_000, // Refresh every minute
    staleTime: 30_000, // Consider data fresh for 30 seconds
  })
}
