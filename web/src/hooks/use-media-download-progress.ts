import { useEffect, useMemo, useState } from 'react'

import { useDownloadingStore } from '@/stores'
import type { QueueItem } from '@/types/queue'

export type MediaTarget =
  | { mediaType: 'movie'; movieId: number }
  | { mediaType: 'series'; seriesId: number }
  | { mediaType: 'season'; seriesId: number; seasonNumber: number }
  | { mediaType: 'episode'; episodeId: number; seriesId?: number; seasonNumber?: number }
  | { mediaType: 'movie-slot'; movieId: number; slotId: number }
  | {
      mediaType: 'episode-slot'
      episodeId: number
      slotId: number
      seriesId?: number
      seasonNumber?: number
    }

export type MediaDownloadProgress = {
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

function isActiveItem(item: QueueItem): boolean {
  return item.status === 'downloading' || item.status === 'queued' || item.status === 'paused'
}

function matchesSeriesBroadly(
  item: QueueItem,
  seriesId: number,
  seasonNumber?: number,
): boolean {
  if (item.seriesId !== seriesId) {
    return false
  }
  if (item.isCompleteSeries) {
    return true
  }
  return !!seasonNumber && item.seasonNumber === seasonNumber && !!item.isSeasonPack
}

function matchMovie(item: QueueItem, target: { movieId: number }): boolean {
  return item.movieId === target.movieId
}

function matchSeries(item: QueueItem, target: { seriesId: number }): boolean {
  return item.seriesId === target.seriesId && !!item.isCompleteSeries
}

function matchSeason(
  item: QueueItem,
  target: { seriesId: number; seasonNumber: number },
): boolean {
  if (item.seriesId !== target.seriesId) {
    return false
  }
  if (item.isCompleteSeries) {
    return true
  }
  return item.seasonNumber === target.seasonNumber && !!item.isSeasonPack
}

function matchEpisode(
  item: QueueItem,
  target: { episodeId: number; seriesId?: number; seasonNumber?: number },
): boolean {
  if (item.episodeId === target.episodeId) {
    return true
  }
  if (!target.seriesId) {
    return false
  }
  return matchesSeriesBroadly(item, target.seriesId, target.seasonNumber)
}

function matchMovieSlot(
  item: QueueItem,
  target: { movieId: number; slotId: number },
): boolean {
  return item.movieId === target.movieId && item.targetSlotId === target.slotId
}

function matchEpisodeSlot(
  item: QueueItem,
  target: { episodeId: number; slotId: number; seriesId?: number; seasonNumber?: number },
): boolean {
  if (item.episodeId === target.episodeId && item.targetSlotId === target.slotId) {
    return true
  }
  if (!target.seriesId || item.seriesId !== target.seriesId || item.targetSlotId !== target.slotId) {
    return false
  }
  return matchesSeriesBroadly(item, target.seriesId, target.seasonNumber)
}

const matchers: Record<MediaTarget['mediaType'], (item: QueueItem, target: never) => boolean> = {
  'movie': matchMovie as (item: QueueItem, target: never) => boolean,
  'series': matchSeries as (item: QueueItem, target: never) => boolean,
  'season': matchSeason as (item: QueueItem, target: never) => boolean,
  'episode': matchEpisode as (item: QueueItem, target: never) => boolean,
  'movie-slot': matchMovieSlot as (item: QueueItem, target: never) => boolean,
  'episode-slot': matchEpisodeSlot as (item: QueueItem, target: never) => boolean,
}

function matchItems(queueItems: QueueItem[], target: MediaTarget): QueueItem[] {
  const matcher = matchers[target.mediaType]
  return queueItems.filter((item) => isActiveItem(item) && matcher(item, target as never))
}

const COMPLETION_DURATION = 2500

export function useMediaDownloadProgress(target: MediaTarget): MediaDownloadProgress {
  const queueItems = useDownloadingStore((state) => state.queueItems)
  const [justCompleted, setJustCompleted] = useState(false)
  const [prevItemCount, setPrevItemCount] = useState(0)

  const items = useMemo(() => matchItems(queueItems, target), [queueItems, target])

  const itemCount = items.length

  if (itemCount !== prevItemCount) {
    setPrevItemCount(itemCount)
    if (prevItemCount > 0 && itemCount === 0) {
      setJustCompleted(true)
    }
  }

  useEffect(() => {
    if (!justCompleted) {
      return
    }
    const timer = setTimeout(() => {
      setJustCompleted(false)
    }, COMPLETION_DURATION)
    return () => clearTimeout(timer)
  }, [justCompleted])

  return useMemo(() => {
    const isDownloading = items.length > 0
    const isPaused = isDownloading && items.every((i) => i.status === 'paused')
    const size = items.reduce((acc, i) => acc + (i.size || 0), 0)
    const downloadedSize = items.reduce((acc, i) => acc + (i.downloadedSize || 0), 0)

    return {
      isDownloading,
      isPaused,
      progress: size > 0 ? (downloadedSize / size) * 100 : 0,
      speed: items.reduce((acc, i) => acc + (i.downloadSpeed || 0), 0),
      eta: items.reduce((acc, i) => Math.max(acc, i.eta || 0), 0),
      size,
      downloadedSize,
      items,
      releaseName: items[0]?.releaseName || items[0]?.title || '',
      justCompleted,
    }
  }, [items, justCompleted])
}

export function clearCompletionEarly(setJustCompleted: (v: boolean) => void) {
  setJustCompleted(false)
}
