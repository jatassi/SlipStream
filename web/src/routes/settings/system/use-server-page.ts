import { useCallback, useState } from 'react'

import { toast } from 'sonner'

import { useUpdateSettings } from '@/hooks'

type LogRotationSettings = {
  maxSizeMB: number
  maxBackups: number
  maxAgeDays: number
  compress: boolean
}

function useTrackedField<T>(defaultValue: T) {
  const [value, setValue] = useState(defaultValue)
  const [initial, setInitial] = useState<T | null>(null)
  const handler = useCallback(
    (v: T) => {
      if (initial === null) {
        setInitial(v)
      }
      setValue(v)
    },
    [initial],
  )
  return { value, initial, handler }
}

function hasLogRotationChanges(
  current: LogRotationSettings,
  initial: LogRotationSettings | null,
): boolean {
  if (!initial) {return false}
  return (
    current.maxSizeMB !== initial.maxSizeMB ||
    current.maxBackups !== initial.maxBackups ||
    current.maxAgeDays !== initial.maxAgeDays ||
    current.compress !== initial.compress
  )
}

export function useServerPage() {
  const updateMutation = useUpdateSettings()

  const port = useTrackedField('')
  const logLevel = useTrackedField('')
  const logRotation = useTrackedField<LogRotationSettings>({
    maxSizeMB: 10,
    maxBackups: 5,
    maxAgeDays: 30,
    compress: false,
  })
  const externalAccess = useTrackedField(false)

  const hasChanges =
    (port.value !== port.initial && port.initial !== null && port.initial !== '') ||
    (logLevel.value !== logLevel.initial && logLevel.initial !== null && logLevel.initial !== '') ||
    hasLogRotationChanges(logRotation.value, logRotation.initial) ||
    (externalAccess.initial !== null && externalAccess.value !== externalAccess.initial)

  const handleSave = async () => {
    try {
      await updateMutation.mutateAsync({
        serverPort: Number.parseInt(port.value),
        logLevel: logLevel.value,
        logMaxSizeMB: logRotation.value.maxSizeMB,
        logMaxBackups: logRotation.value.maxBackups,
        logMaxAgeDays: logRotation.value.maxAgeDays,
        logCompress: logRotation.value.compress,
        externalAccessEnabled: externalAccess.value,
      })
      toast.success('Settings saved')
    } catch {
      toast.error('Failed to save settings')
    }
  }

  return {
    port: port.value,
    onPortChange: port.handler,
    logLevel: logLevel.value,
    onLogLevelChange: logLevel.handler,
    logRotation: logRotation.value,
    onLogRotationChange: logRotation.handler,
    externalAccessEnabled: externalAccess.value,
    onExternalAccessChange: externalAccess.handler,
    hasChanges,
    isSaving: updateMutation.isPending,
    handleSave,
  }
}
