import { useState } from 'react'

import { toast } from 'sonner'

import { useResetProwlarrIndexerStats, useUpdateProwlarrIndexerSettings } from '@/hooks'
import type { ContentType, ProwlarrIndexerSettingsInput, ProwlarrIndexerWithSettings } from '@/types'

export function useIndexerSettingsDialog(indexer: ProwlarrIndexerWithSettings) {
  const updateSettings = useUpdateProwlarrIndexerSettings()
  const resetStats = useResetProwlarrIndexerStats()

  const [priority, setPriority] = useState(indexer.settings?.priority ?? 25)
  const [contentType, setContentType] = useState<ContentType>(
    indexer.settings?.contentType ?? 'both',
  )
  const [open, setOpen] = useState(false)

  const handleSave = async () => {
    const data: ProwlarrIndexerSettingsInput = { priority, contentType }
    try {
      await updateSettings.mutateAsync({ indexerId: indexer.id, data })
      toast.success(`Settings updated for ${indexer.name}`)
      setOpen(false)
    } catch {
      toast.error('Failed to update settings')
    }
  }

  const handleResetStats = async () => {
    try {
      await resetStats.mutateAsync(indexer.id)
      toast.success('Stats reset')
    } catch {
      toast.error('Failed to reset stats')
    }
  }

  const handleOpenChange = (newOpen: boolean) => {
    setOpen(newOpen)
    if (newOpen) {
      setPriority(indexer.settings?.priority ?? 25)
      setContentType(indexer.settings?.contentType ?? 'both')
    }
  }

  const handlePriorityChange = (value: string) => {
    setPriority(Math.min(50, Math.max(1, Number.parseInt(value) || 1)))
  }

  return {
    open,
    priority,
    contentType,
    setContentType,
    handleOpenChange,
    handleSave,
    handleResetStats,
    handlePriorityChange,
    isSaving: updateSettings.isPending,
    isResetting: resetStats.isPending,
    settings: indexer.settings,
  }
}
