import { useState } from 'react'

import { toast } from 'sonner'

import { useClearHistory, useGlobalLoading, useHistory } from '@/hooks'
import { filterableEventTypes } from '@/lib/history-utils'
import type { HistoryEventType } from '@/types'

import type { DatePreset, MediaFilter } from './history-utils'
import { getAfterDate } from './history-utils'

function useHistoryFilters() {
  const [eventTypes, setEventTypes] = useState<HistoryEventType[]>(
    filterableEventTypes.map((et) => et.value),
  )
  const [mediaType, setMediaType] = useState<MediaFilter>('all')
  const [datePreset, setDatePreset] = useState<DatePreset>('all')
  const [page, setPage] = useState(1)
  const [expandedId, setExpandedId] = useState<number | null>(null)

  const handleToggleEventType = (value: HistoryEventType) => {
    setEventTypes((prev) =>
      prev.includes(value) ? prev.filter((v) => v !== value) : [...prev, value],
    )
    setPage(1)
  }
  const handleResetEventTypes = () => setEventTypes(filterableEventTypes.map((e) => e.value))
  const handleMediaTypeChange = (v: string) => { setMediaType(v as MediaFilter); setPage(1) }
  const handleDatePresetChange = (v: string | null) => {
    if (v) { setDatePreset(v as DatePreset); setPage(1) }
  }
  const handleToggleExpanded = (itemId: number, hasDetails: boolean) => {
    if (hasDetails) {setExpandedId(expandedId === itemId ? null : itemId)}
  }

  return {
    eventTypes, mediaType, datePreset, page, expandedId, setPage,
    handleToggleEventType, handleResetEventTypes, handleMediaTypeChange,
    handleDatePresetChange, handleToggleExpanded,
  }
}

export function useHistoryPage() {
  const filters = useHistoryFilters()
  const allSelected = filters.eventTypes.length >= filterableEventTypes.length
  const mediaTypeParam = filters.mediaType === 'all' ? undefined : filters.mediaType

  const globalLoading = useGlobalLoading()
  const { data: history, isLoading: queryLoading, isError, refetch } = useHistory({
    eventType: allSelected ? undefined : filters.eventTypes.join(','),
    mediaType: mediaTypeParam,
    after: getAfterDate(filters.datePreset),
    page: filters.page,
    pageSize: 50,
  })

  const isLoading = queryLoading || globalLoading
  const clearMutation = useClearHistory()

  const handleClearHistory = async () => {
    try {
      await clearMutation.mutateAsync()
      toast.success('History cleared')
    } catch {
      toast.error('Failed to clear history')
    }
  }

  const handlePreviousPage = () => filters.setPage((p) => Math.max(1, p - 1))
  const handleNextPage = () => {
    if (history) {
      filters.setPage((p) => Math.min(history.totalPages, p + 1))
    }
  }
  const handlePageSelect = (p: number) => filters.setPage(p)

  return {
    ...filters,
    history,
    isLoading,
    isError,
    refetch,
    handleClearHistory,
    handlePreviousPage,
    handleNextPage,
    handlePageSelect,
  }
}
