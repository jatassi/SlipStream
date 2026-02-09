import { create } from 'zustand'
import type { QueueItem } from '@/types/queue'

interface DownloadingState {
  queueItems: QueueItem[]
  setQueueItems: (items: QueueItem[]) => void
  isMovieDownloading: (movieId: number) => boolean
  isSeriesDownloading: (seriesId: number) => boolean
  isSeasonDownloading: (seriesId: number, seasonNumber: number) => boolean
  isEpisodeDownloading: (episodeId: number, seriesId?: number, seasonNumber?: number) => boolean
  isSlotDownloading: (slotId: number) => boolean
}

export const useDownloadingStore = create<DownloadingState>((set, get) => ({
  queueItems: [],

  setQueueItems: (items) => set({ queueItems: items }),

  isMovieDownloading: (movieId) => {
    const { queueItems } = get()
    return queueItems.some(
      (item) =>
        item.movieId === movieId &&
        (item.status === 'downloading' || item.status === 'queued')
    )
  },

  isSeriesDownloading: (seriesId) => {
    const { queueItems } = get()
    return queueItems.some(
      (item) =>
        item.seriesId === seriesId &&
        item.isCompleteSeries &&
        (item.status === 'downloading' || item.status === 'queued')
    )
  },

  isSeasonDownloading: (seriesId, seasonNumber) => {
    const { queueItems } = get()
    return queueItems.some(
      (item) =>
        item.seriesId === seriesId &&
        ((item.seasonNumber === seasonNumber && item.isSeasonPack) ||
          item.isCompleteSeries) &&
        (item.status === 'downloading' || item.status === 'queued')
    )
  },

  isEpisodeDownloading: (episodeId, seriesId, seasonNumber) => {
    const { queueItems } = get()
    return queueItems.some((item) => {
      if (item.status !== 'downloading' && item.status !== 'queued') return false
      // Direct episode match
      if (item.episodeId === episodeId) return true
      // Season pack or complete series covering this episode
      if (seriesId && item.seriesId === seriesId) {
        if (item.isCompleteSeries) return true
        if (seasonNumber && item.seasonNumber === seasonNumber && item.isSeasonPack) return true
      }
      return false
    })
  },

  isSlotDownloading: (slotId) => {
    const { queueItems } = get()
    return queueItems.some(
      (item) =>
        item.targetSlotId === slotId &&
        (item.status === 'downloading' || item.status === 'queued' || item.status === 'paused')
    )
  },
}))
