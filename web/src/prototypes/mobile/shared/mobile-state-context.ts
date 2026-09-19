import { createContext, useContext } from 'react'

import type { MediaItem, QueueItem } from './types'

export type MobileStore = {
  queue: QueueItem[]
  library: MediaItem[]
  togglePause: (id: string) => void
  removeFromQueue: (id: string) => void
  toggleMonitored: (id: number) => void
  media: (id: number) => MediaItem
}

export const MobileStateContext = createContext<MobileStore | null>(null)

export function useMobileState(): MobileStore {
  const store = useContext(MobileStateContext)
  if (!store) {
    throw new Error('useMobileState must be used inside MobileStateProvider')
  }
  return store
}
