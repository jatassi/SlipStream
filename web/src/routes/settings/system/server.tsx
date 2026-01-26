import { useState, useCallback } from 'react'
import { Save } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { ServerSection } from '@/components/settings'
import { SystemNav } from './SystemNav'
import { useUpdateSettings } from '@/hooks'
import { toast } from 'sonner'

interface LogRotationSettings {
  maxSizeMB: number
  maxBackups: number
  maxAgeDays: number
  compress: boolean
}

export function ServerPage() {
  const updateMutation = useUpdateSettings()

  const [port, setPort] = useState('')
  const [logLevel, setLogLevel] = useState('')
  const [logRotation, setLogRotation] = useState<LogRotationSettings>({
    maxSizeMB: 10,
    maxBackups: 5,
    maxAgeDays: 30,
    compress: false,
  })
  const [externalAccessEnabled, setExternalAccessEnabled] = useState(false)
  const [initialPort, setInitialPort] = useState('')
  const [initialLogLevel, setInitialLogLevel] = useState('')
  const [initialLogRotation, setInitialLogRotation] = useState<LogRotationSettings | null>(null)
  const [initialExternalAccess, setInitialExternalAccess] = useState<boolean | null>(null)

  const handlePortChange = useCallback((value: string) => {
    if (!initialPort) setInitialPort(value)
    setPort(value)
  }, [initialPort])

  const handleLogLevelChange = useCallback((value: string) => {
    if (!initialLogLevel) setInitialLogLevel(value)
    setLogLevel(value)
  }, [initialLogLevel])

  const handleLogRotationChange = useCallback((value: LogRotationSettings) => {
    if (!initialLogRotation) setInitialLogRotation(value)
    setLogRotation(value)
  }, [initialLogRotation])

  const handleExternalAccessChange = useCallback((value: boolean) => {
    if (initialExternalAccess === null) setInitialExternalAccess(value)
    setExternalAccessEnabled(value)
  }, [initialExternalAccess])

  const hasChanges = (port !== initialPort && initialPort !== '') ||
                     (logLevel !== initialLogLevel && initialLogLevel !== '') ||
                     (initialLogRotation !== null && (
                       logRotation.maxSizeMB !== initialLogRotation.maxSizeMB ||
                       logRotation.maxBackups !== initialLogRotation.maxBackups ||
                       logRotation.maxAgeDays !== initialLogRotation.maxAgeDays ||
                       logRotation.compress !== initialLogRotation.compress
                     )) ||
                     (initialExternalAccess !== null && externalAccessEnabled !== initialExternalAccess)

  const handleSave = async () => {
    try {
      await updateMutation.mutateAsync({
        serverPort: parseInt(port),
        logLevel,
        logMaxSizeMB: logRotation.maxSizeMB,
        logMaxBackups: logRotation.maxBackups,
        logMaxAgeDays: logRotation.maxAgeDays,
        logCompress: logRotation.compress,
        externalAccessEnabled,
      })
      setInitialPort(port)
      setInitialLogLevel(logLevel)
      setInitialLogRotation(logRotation)
      setInitialExternalAccess(externalAccessEnabled)
      toast.success('Settings saved')
    } catch {
      toast.error('Failed to save settings')
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="System"
        description="Server configuration and authentication settings"
        breadcrumbs={[
          { label: 'Settings', href: '/settings/media' },
          { label: 'System' },
        ]}
        actions={
          <Button onClick={handleSave} disabled={updateMutation.isPending || !hasChanges}>
            <Save className="size-4 mr-2" />
            Save Changes
          </Button>
        }
      />

      <SystemNav />

      <div className="max-w-2xl">
        <ServerSection
          port={port}
          onPortChange={handlePortChange}
          logLevel={logLevel}
          onLogLevelChange={handleLogLevelChange}
          logRotation={logRotation}
          onLogRotationChange={handleLogRotationChange}
          externalAccessEnabled={externalAccessEnabled}
          onExternalAccessChange={handleExternalAccessChange}
        />
      </div>
    </div>
  )
}
