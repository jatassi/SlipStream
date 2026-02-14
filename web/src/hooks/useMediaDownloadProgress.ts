import { useEffect, useMemo, useState } from 'react'
import { useDownloadingStore } from '@/stores'
import type { QueueItem } from '@/types/queue'

export type MediaTarget =
  | { mediaType: 'movie'; movieId: number }
  | { mediaType: 'series'; seriesId: number }
  | { mediaType: 'season'; seriesId: number; seasonNumber: number }
  | { mediaType: 'episode'; episodeId: number; seriesId?: number; seasonNumber?: number }
  | { mediaType: 'movie-slot'; movieId: number; slotId: number }
  | { mediaType: 'episode-slot'; episodeId: number; slotId: number; seriesId?: number; seasonNumber?: number }

export interface MediaDownloadProgress {
  isDownloading: boolean
  isPaused: boolean
  progress: number
  speed: number
  eta: number
  size: number
  downloadedSize: number
  items: QueueItem[]
  releaseName: string
  justCompleted: boolean
}

function matchItems(queueItems: QueueItem[], target: MediaTarget): QueueItem[] {
  return queueItems.filter((item) => {
    const active = item.status === 'downloading' || item.status === 'queued' || item.status === 'paused'
    if (!active) return false

    switch (target.mediaType) {
      case 'movie':
        return item.movieId === target.movieId
      case 'series':
        return item.seriesId === target.seriesId && item.isCompleteSeries
      case 'season':
        return (
          item.seriesId === target.seriesId &&
          (item.isCompleteSeries ||
            (item.seasonNumber === target.seasonNumber && item.isSeasonPack))
        )
      case 'episode':
        if (item.episodeId === target.episodeId) return true
        if (target.seriesId && item.seriesId === target.seriesId) {
          if (item.isCompleteSeries) return true
          if (target.seasonNumber && item.seasonNumber === target.seasonNumber && item.isSeasonPack) return true
        }
        return false
      case 'movie-slot':
        return item.movieId === target.movieId && item.targetSlotId === target.slotId
      case 'episode-slot':
        if (item.episodeId === target.episodeId && item.targetSlotId === target.slotId) return true
        if (target.seriesId && item.seriesId === target.seriesId && item.targetSlotId === target.slotId) {
          if (item.isCompleteSeries) return true
          if (target.seasonNumber && item.seasonNumber === target.seasonNumber && item.isSeasonPack) return true
        }
        return false
    }
  })
}

const COMPLETION_DURATION = 2500

export function useMediaDownloadProgress(target: MediaTarget): MediaDownloadProgress {
  const queueItems = useDownloadingStore((state) => state.queueItems)
  const [justCompleted, setJustCompleted] = useState(false)
  const [prevItemCount, setPrevItemCount] = useState(0)

  const items = useMemo(() => matchItems(queueItems, target), [queueItems, target])

  const itemCount = items.length

  // Detect completion: items we were tracking disappeared (render-time state adjustment)
  if (itemCount !== prevItemCount) {
    setPrevItemCount(itemCount)
    if (prevItemCount > 0 && itemCount === 0) {
      setJustCompleted(true)
    }
  }

  // Auto-reset completion flag after delay
  useEffect(() => {
    if (!justCompleted) return
    const timer = setTimeout(() => {
      setJustCompleted(false)
    }, COMPLETION_DURATION)
    return () => clearTimeout(timer)
  }, [justCompleted])

  const isDownloading = items.length > 0
  const isPaused = isDownloading && items.every((i) => i.status === 'paused')

  const size = items.reduce((acc, i) => acc + (i.size || 0), 0)
  const downloadedSize = items.reduce((acc, i) => acc + (i.downloadedSize || 0), 0)
  const progress = size > 0 ? (downloadedSize / size) * 100 : 0
  const speed = items.reduce((acc, i) => acc + (i.downloadSpeed || 0), 0)
  const eta = items.reduce((acc, i) => Math.max(acc, i.eta || 0), 0)
  const releaseName = items[0]?.releaseName || items[0]?.title || ''

  return {
    isDownloading,
    isPaused,
    progress,
    speed,
    eta,
    size,
    downloadedSize,
    items,
    releaseName,
    justCompleted,
  }
}

export function clearCompletionEarly(setJustCompleted: (v: boolean) => void) {
  setJustCompleted(false)
}
