import { useState } from 'react'

import type { Request } from '@/types'

export function useRequestSelection(requests: Request[]) {
  const [activeTab, setActiveTab] = useState<string>('pending')
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())

  const filteredRequests = requests.filter((r) => {
    if (activeTab === 'all') {
      return true
    }
    return r.status === activeTab
  })

  const pendingCount = requests.filter((r) => r.status === 'pending').length
  const isAllSelected =
    filteredRequests.length > 0 && filteredRequests.every((r) => selectedIds.has(r.id))
  const isSomeSelected = selectedIds.size > 0

  const handleTabChange = (value: string) => {
    setActiveTab(value)
    setSelectedIds(new Set())
  }

  const toggleSelectAll = () => {
    if (isAllSelected) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(filteredRequests.map((r) => r.id)))
    }
  }

  const toggleSelect = (id: number) => {
    const newSelected = new Set(selectedIds)
    if (newSelected.has(id)) {
      newSelected.delete(id)
    } else {
      newSelected.add(id)
    }
    setSelectedIds(newSelected)
  }

  const clearSelection = () => {
    setSelectedIds(new Set())
  }

  return {
    activeTab,
    handleTabChange,
    selectedIds,
    toggleSelectAll,
    toggleSelect,
    clearSelection,
    filteredRequests,
    pendingCount,
    isAllSelected,
    isSomeSelected,
  }
}
