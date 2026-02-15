import { useState } from 'react'

import { useGlobalLoading, useQueue } from '@/hooks'
import type { QueueItem } from '@/types'

export type MediaFilter = 'all' | 'movies' | 'series'

function filterItems(items: QueueItem[], filter: MediaFilter): QueueItem[] {
  if (filter === 'all') {
    return items
  }
  const mediaType = filter === 'movies' ? 'movie' : 'series'
  return items.filter((item) => item.mediaType === mediaType)
}

export function useActivityPage() {
  const [filter, setFilter] = useState<MediaFilter>('all')
  const globalLoading = useGlobalLoading()
  const { data: queueResponse, isLoading: queryLoading, isError, isFetching, refetch } = useQueue()
  const isLoading = queryLoading || globalLoading

  const items = queueResponse?.items ?? []
  const clientErrors = queueResponse?.errors ?? []

  const filteredItems = filterItems(items, filter).toSorted((a, b) =>
    a.title.localeCompare(b.title),
  )

  const movieCount = items.filter((q) => q.mediaType === 'movie').length
  const seriesCount = items.filter((q) => q.mediaType === 'series').length
  const totalCount = items.length

  return {
    filter,
    setFilter,
    isLoading,
    isError,
    isFetching,
    refetch,
    filteredItems,
    clientErrors,
    movieCount,
    seriesCount,
    totalCount,
  }
}
