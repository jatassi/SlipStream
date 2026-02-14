import type { StorageInfo } from '@/types/storage'

import { apiFetch } from './client'

export const storageApi = {
  getStorage: async (): Promise<StorageInfo[]> => {
    return apiFetch<StorageInfo[]>('/filesystem/storage')
  },
}
