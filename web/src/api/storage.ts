import { apiFetch } from './client'
import type { StorageInfo } from '@/types/storage'

export const storageApi = {
  getStorage: async (): Promise<StorageInfo[]> => {
    return apiFetch<StorageInfo[]>('/filesystem/storage')
  }
}