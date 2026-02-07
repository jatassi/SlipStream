import { create } from 'zustand'

export interface ArtworkReadyPayload {
  mediaType: 'movie' | 'series'
  mediaId: number
  artworkType: 'poster' | 'backdrop' | 'logo'
}

interface ArtworkState {
  // Track artwork versions by key (mediaType:mediaId:artworkType)
  versions: Map<string, number>
  // Notify that artwork is ready
  notifyReady: (payload: ArtworkReadyPayload) => void
  // Get current version for a specific artwork
  getVersion: (mediaType: string, mediaId: number, artworkType: string) => number
}

function makeKey(mediaType: string, mediaId: number, artworkType: string): string {
  return `${mediaType}:${mediaId}:${artworkType}`
}

export const useArtworkStore = create<ArtworkState>((set, get) => ({
  versions: new Map(),

  notifyReady: (payload) => {
    const key = makeKey(payload.mediaType, payload.mediaId, payload.artworkType)
    set((state) => {
      const newVersions = new Map(state.versions)
      newVersions.set(key, Date.now())
      return { versions: newVersions }
    })
  },

  getVersion: (mediaType, mediaId, artworkType) => {
    const key = makeKey(mediaType, mediaId, artworkType)
    return get().versions.get(key) || 0
  },
}))
