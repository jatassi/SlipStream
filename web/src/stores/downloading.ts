import { create } from 'zustand'

import type { QueueItem } from '@/types/queue'

function isEpisodeInDownload(
  item: QueueItem,
  params: { episodeId: number; seriesId?: number; seasonNumber?: number },
): boolean {
  if (item.status !== 'downloading' && item.status !== 'queued') {
    return false
  }
  if (item.episodeId === params.episodeId) {
    return true
  }
  if (params.seriesId && item.seriesId === params.seriesId) {
    if (item.isCompleteSeries) {
      return true
    }
    if (params.seasonNumber && item.seasonNumber === params.seasonNumber && item.isSeasonPack) {
      return true
    }
  }
  return false
}

type DownloadingState = {
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
        item.movieId === movieId && (item.status === 'downloading' || item.status === 'queued'),
    )
  },

  isSeriesDownloading: (seriesId) => {
    const { queueItems } = get()
    return queueItems.some(
      (item) =>
        item.seriesId === seriesId &&
        item.isCompleteSeries &&
        (item.status === 'downloading' || item.status === 'queued'),
    )
  },

  isSeasonDownloading: (seriesId, seasonNumber) => {
    const { queueItems } = get()
    return queueItems.some(
      (item) =>
        item.seriesId === seriesId &&
        ((item.seasonNumber === seasonNumber && item.isSeasonPack === true) || item.isCompleteSeries === true) &&
        (item.status === 'downloading' || item.status === 'queued'),
    )
  },

  isEpisodeDownloading: (episodeId, seriesId, seasonNumber) => {
    const { queueItems } = get()
    return queueItems.some((item) =>
      isEpisodeInDownload(item, { episodeId, seriesId, seasonNumber }),
    )
  },

  isSlotDownloading: (slotId) => {
    const { queueItems } = get()
    return queueItems.some(
      (item) =>
        item.targetSlotId === slotId &&
        (item.status === 'downloading' || item.status === 'queued' || item.status === 'paused'),
    )
  },
}))
