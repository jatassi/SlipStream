import { useState } from 'react'

import { toast } from 'sonner'

import { useClearHistory, useHistory } from '@/hooks'
import { filterableEventTypes } from '@/lib/history-utils'
import { useUIStore } from '@/stores'
import type { HistoryEventType } from '@/types'

import type { DatePreset, MediaFilter } from './history-utils'
import { getAfterDate } from './history-utils'

const PAGE_SIZE = 50

function useHistoryFilters() {
  const [eventTypes, setEventTypes] = useState<HistoryEventType[]>(
    filterableEventTypes.map((et) => et.value),
  )
  const [mediaType, setMediaType] = useState<MediaFilter>('all')
  const [datePreset, setDatePreset] = useState<DatePreset>('all')
  const [limit, setLimit] = useState(PAGE_SIZE)

  const handleToggleEventType = (value: HistoryEventType) => {
    setEventTypes((prev) =>
      prev.includes(value) ? prev.filter((v) => v !== value) : [...prev, value],
    )
    setLimit(PAGE_SIZE)
  }

  const handleResetEventTypes = () => {
    setEventTypes(filterableEventTypes.map((e) => e.value))
    setLimit(PAGE_SIZE)
  }

  const handleMediaTypeChange = (value: MediaFilter) => {
    setMediaType(value)
    setLimit(PAGE_SIZE)
  }

  const handleDatePresetChange = (value: string | null) => {
    if (value) {
      setDatePreset(value as DatePreset)
      setLimit(PAGE_SIZE)
    }
  }

  return {
    eventTypes,
    mediaType,
    datePreset,
    limit,
    handleToggleEventType,
    handleResetEventTypes,
    handleMediaTypeChange,
    handleDatePresetChange,
    handleLoadMore: () => {
      setLimit((value) => value + PAGE_SIZE)
    },
  }
}

export function useHistoryPage() {
  const filters = useHistoryFilters()
  const allSelected = filters.eventTypes.length >= filterableEventTypes.length

  const globalLoading = useUIStore((s) => s.globalLoading)
  const {
    data: history,
    isLoading: queryLoading,
    isError,
    refetch,
  } = useHistory({
    eventType: allSelected ? undefined : filters.eventTypes.join(','),
    mediaType: filters.mediaType === 'all' ? undefined : filters.mediaType,
    after: getAfterDate(filters.datePreset),
    page: 1,
    pageSize: filters.limit,
  })

  const clearMutation = useClearHistory()
  const items = history?.items ?? []

  const handleClearHistory = async () => {
    try {
      await clearMutation.mutateAsync()
      toast.success('History cleared')
    } catch {
      toast.error('Failed to clear history')
    }
  }

  return {
    ...filters,
    items,
    hasMore: items.length < (history?.totalCount ?? 0),
    isLoading: queryLoading || globalLoading,
    isError,
    refetch,
    handleClearHistory,
  }
}
