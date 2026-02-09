import { useState, useCallback } from 'react'
import { Save, History } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { ServerSection } from '@/components/settings'
import { SystemNav } from './SystemNav'
import { useUpdateSettings, useHistorySettings, useUpdateHistorySettings } from '@/hooks'
import { toast } from 'sonner'

interface LogRotationSettings {
  maxSizeMB: number
  maxBackups: number
  maxAgeDays: number
  compress: boolean
}

function HistoryRetentionCard() {
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

  const currentEnabled = enabled ?? settings?.enabled ?? true
  const currentDays = days ?? settings?.retentionDays ?? 365

  const hasChanges = settings && (
    currentEnabled !== settings.enabled ||
    currentDays !== settings.retentionDays
  )

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

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <History className="size-4" />
          History Retention
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center justify-between">
          <Label htmlFor="retention-enabled">Auto-cleanup old history entries</Label>
          <Switch
            id="retention-enabled"
            checked={currentEnabled}
            onCheckedChange={(v) => setEnabled(v)}
          />
        </div>
        {currentEnabled && (
          <div className="space-y-2">
            <Label htmlFor="retention-days">Retention period (days)</Label>
            <Input
              id="retention-days"
              type="number"
              min={1}
              max={3650}
              value={currentDays}
              onChange={(e) => setDays(parseInt(e.target.value) || 1)}
              className="w-32"
            />
            <p className="text-xs text-muted-foreground">
              History entries older than this will be automatically deleted daily at 2 AM.
            </p>
          </div>
        )}
        {hasChanges && (
          <Button size="sm" onClick={handleSave} disabled={updateMutation.isPending}>
            <Save className="size-3 mr-1.5" />
            Save
          </Button>
        )}
      </CardContent>
    </Card>
  )
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

      <div className="max-w-2xl space-y-6">
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
        <HistoryRetentionCard />
      </div>
    </div>
  )
}
