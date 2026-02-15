import { useEffect, useMemo, useRef, useState } from 'react'

import { useRouterState } from '@tanstack/react-router'

import { useDownloadingStore } from '@/stores'

import type { DownloadTheme } from './downloads-nav-types'

type DownloadStats = {
  progress: number
  theme: DownloadTheme
  movieCount: number
  tvCount: number
  hasDownloads: boolean
  allPaused: boolean
}

function computeDownloadStats(queueItems: { mediaType: string; size?: number; downloadedSize?: number; status?: string }[]): DownloadStats {
  const movieCount = queueItems.filter((item) => item.mediaType === 'movie').length
  const tvCount = queueItems.filter((item) => item.mediaType === 'series').length

  let theme: DownloadTheme = 'none'
  if (movieCount > 0 && tvCount > 0) {
    theme = 'both'
  } else if (movieCount > 0) {
    theme = 'movie'
  } else if (tvCount > 0) {
    theme = 'tv'
  }

  const totalSize = queueItems.reduce((acc, item) => acc + (item.size ?? 0), 0)
  const downloadedSize = queueItems.reduce((acc, item) => acc + (item.downloadedSize ?? 0), 0)
  const progress = totalSize > 0 ? (downloadedSize / totalSize) * 100 : 0
  const hasDownloads = queueItems.length > 0
  const allPaused = hasDownloads && queueItems.every((i) => i.status === 'paused')

  return { progress, theme, movieCount, tvCount, hasDownloads, allPaused }
}

function determineFlashTheme(removedTypes: Set<'movie' | 'series'>): DownloadTheme {
  if (removedTypes.has('movie') && removedTypes.has('series')) {
    return 'both'
  }
  if (removedTypes.has('movie')) {
    return 'movie'
  }
  return 'tv'
}

function detectRemovedTypes(
  prevItems: Map<string, string>,
  currentItems: Map<string, string>,
): Set<'movie' | 'series'> {
  const removedTypes = new Set<'movie' | 'series'>()
  for (const [id, mediaType] of prevItems) {
    if (currentItems.has(id)) {
      continue
    }
    if (mediaType === 'movie') {
      removedTypes.add('movie')
    }
    if (mediaType === 'series') {
      removedTypes.add('series')
    }
  }
  return removedTypes
}

function useCompletionFlash(queueItems: { id: string; mediaType: string }[]): DownloadTheme | null {
  const [completionFlash, setCompletionFlash] = useState<DownloadTheme | null>(null)
  const prevItemsRef = useRef<Map<string, string>>(new Map())

  const itemIds = useMemo(
    () =>
      queueItems
        .map((i) => `${i.id}:${i.mediaType}`)
        .toSorted()
        .join(','),
    [queueItems],
  )

  useEffect(() => {
    const currentItems = new Map(queueItems.map((i) => [i.id, i.mediaType]))
    const prevItems = prevItemsRef.current

    if (prevItems.size > 0 && currentItems.size < prevItems.size) {
      const removedTypes = detectRemovedTypes(prevItems, currentItems)
      if (removedTypes.size > 0) {
        queueMicrotask(() => setCompletionFlash(determineFlashTheme(removedTypes)))
      }
    }

    prevItemsRef.current = currentItems
  }, [itemIds]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!completionFlash) {
      return
    }
    const timer = setTimeout(() => setCompletionFlash(null), 800)
    return () => clearTimeout(timer)
  }, [completionFlash])

  return completionFlash
}

export function useDownloadsNav() {
  const router = useRouterState()
  const queueItems = useDownloadingStore((state) => state.queueItems)

  const isActive =
    router.location.pathname === '/activity' || router.location.pathname.startsWith('/activity/')

  const stats = useMemo(() => computeDownloadStats(queueItems), [queueItems])
  const completionFlash = useCompletionFlash(queueItems)

  return { ...stats, isActive, completionFlash }
}
