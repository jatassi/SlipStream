import { useQuery } from '@tanstack/react-query'

import { storageApi } from '@/api/storage'
import { createQueryKeys } from '@/lib/query-keys'
import type { StorageInfo } from '@/types/storage'

const storageKeys = createQueryKeys('storage')

export function useStorage() {
  return useQuery<StorageInfo[]>({
    queryKey: storageKeys.all,
    queryFn: storageApi.getStorage,
    refetchInterval: 60_000, // Refresh every minute
    staleTime: 30_000, // Consider data fresh for 30 seconds
  })
}
