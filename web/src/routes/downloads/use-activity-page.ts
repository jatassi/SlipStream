import { useState } from 'react'

import { useQueue } from '@/hooks'
import { formatSpeed } from '@/lib/formatters'
import { useUIStore } from '@/stores'
import type { QueueItem } from '@/types'

export type MediaFilter = 'all' | 'movies' | 'series'

function filterItems(items: QueueItem[], filter: MediaFilter): QueueItem[] {
  if (filter === 'all') {
    return items
  }
  const mediaType = filter === 'movies' ? 'movie' : 'series'
  return items.filter((item) => item.mediaType === mediaType)
}

function summaryLine(items: QueueItem[]): string {
  const downloading = items.filter((item) => item.status === 'downloading')
  const speed = downloading.reduce((total, item) => total + item.downloadSpeed, 0)
  return `${downloading.length} downloading · ${formatSpeed(speed)} · ${items.length} in queue`
}

export function useActivityPage() {
  const [filter, setFilter] = useState<MediaFilter>('all')
  const globalLoading = useUIStore((s) => s.globalLoading)
  const { data: queueResponse, isLoading: queryLoading, isError, refetch } = useQueue()
  const isLoading = queryLoading || globalLoading

  const items = queueResponse?.items ?? []
  const clientErrors = queueResponse?.errors ?? []

  const filteredItems = filterItems(items, filter).toSorted((a, b) =>
    a.title.localeCompare(b.title),
  )

  return {
    filter,
    setFilter,
    isLoading,
    isError,
    refetch,
    filteredItems,
    clientErrors,
    summary: summaryLine(items),
  }
}
