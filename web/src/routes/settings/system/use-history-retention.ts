import { useState } from 'react'

import { toast } from 'sonner'

import { useHistorySettings, useUpdateHistorySettings } from '@/hooks'

function resolveCurrentValues(
  enabled: boolean | null,
  days: number | null,
  settings?: { enabled: boolean; retentionDays: number },
) {
  return {
    currentEnabled: enabled ?? settings?.enabled ?? true,
    currentDays: days ?? settings?.retentionDays ?? 365,
  }
}

function detectChanges(
  currentEnabled: boolean,
  currentDays: number,
  settings?: { enabled: boolean; retentionDays: number },
) {
  if (!settings) {return false}
  return currentEnabled !== settings.enabled || currentDays !== settings.retentionDays
}

export function useHistoryRetention() {
  const { data: settings } = useHistorySettings()
  const updateMutation = useUpdateHistorySettings()

  const [enabled, setEnabled] = useState<boolean | null>(null)
  const [days, setDays] = useState<number | null>(null)
  const [prevSettings, setPrevSettings] = useState(settings)

  if (settings !== prevSettings) {
    setPrevSettings(settings)
    if (settings) {
      setEnabled(settings.enabled)
      setDays(settings.retentionDays)
    }
  }

  const { currentEnabled, currentDays } = resolveCurrentValues(enabled, days, settings)
  const hasChanges = detectChanges(currentEnabled, currentDays, settings)

  const handleSave = async () => {
    try {
      await updateMutation.mutateAsync({
        enabled: currentEnabled,
        retentionDays: currentDays,
      })
      toast.success('History retention settings saved')
    } catch {
      toast.error('Failed to save history retention settings')
    }
  }

  return {
    currentEnabled,
    currentDays,
    hasChanges,
    isSaving: updateMutation.isPending,
    setEnabled,
    setDays,
    handleSave,
  }
}
