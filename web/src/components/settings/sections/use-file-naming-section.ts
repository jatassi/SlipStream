import { useCallback, useEffect, useRef, useState } from 'react'

import { toast } from 'sonner'

import { useImportSettings, useUpdateImportSettings } from '@/hooks'
import { useDebounce } from '@/hooks/use-debounce'
import type { ImportSettings } from '@/types'

export function useFileNamingSection() {
  const { data: settings, isLoading, isError, refetch } = useImportSettings()
  const updateMutation = useUpdateImportSettings()

  const [form, setForm] = useState<ImportSettings | null>(null)
  const [activeTab, setActiveTab] = useState('validation')
  const [prevSettings, setPrevSettings] = useState<typeof settings>(undefined)

  if (settings !== prevSettings) {
    setPrevSettings(settings)
    if (settings) {
      setForm({ ...settings })
    }
  }

  const updateField = useCallback(
    <K extends keyof ImportSettings>(field: K, value: ImportSettings[K]) => {
      setForm((prev) => (prev ? { ...prev, [field]: value } : null))
    },
    [],
  )

  const hasChanges = form && settings && JSON.stringify(form) !== JSON.stringify(settings)

  const debouncedForm = useDebounce(form, 1000)
  const lastSavedRef = useRef<string | null>(null)

  useEffect(() => {
    if (!debouncedForm || !settings) {
      return
    }
    const formJson = JSON.stringify(debouncedForm)
    const settingsJson = JSON.stringify(settings)
    if (formJson !== settingsJson && formJson !== lastSavedRef.current) {
      lastSavedRef.current = formJson
      updateMutation.mutate(debouncedForm, {
        onError: () => {
          toast.error('Failed to auto-save settings')
          lastSavedRef.current = null
        },
      })
    }
  }, [debouncedForm, settings, updateMutation])

  return {
    form,
    activeTab,
    setActiveTab,
    updateField,
    hasChanges,
    isLoading,
    isError,
    isSaving: updateMutation.isPending,
    refetch,
  }
}
