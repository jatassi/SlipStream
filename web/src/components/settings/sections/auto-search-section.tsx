import { useState } from 'react'

import { Save } from 'lucide-react'
import { toast } from 'sonner'

import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Slider } from '@/components/ui/slider'
import { Switch } from '@/components/ui/switch'
import { useAutoSearchSettings, useUpdateAutoSearchSettings } from '@/hooks'

type SearchSettingsCardProps = {
  enabled: boolean
  onEnabledChange: (v: boolean) => void
  intervalHours: number
  onIntervalChange: (v: number) => void
}

function SearchSettingsCard({ enabled, onEnabledChange, intervalHours, onIntervalChange }: SearchSettingsCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Automatic Search</CardTitle>
        <CardDescription>Scheduled task that searches for missing monitored items</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex items-center justify-between">
          <div className="space-y-0.5">
            <Label>Enable Automatic Search</Label>
            <p className="text-muted-foreground text-sm">Periodically search for missing movies and episodes</p>
          </div>
          <Switch checked={enabled} onCheckedChange={onEnabledChange} />
        </div>
        <div className="space-y-2">
          <div className="flex justify-between">
            <Label>Search Interval</Label>
            <span className="text-muted-foreground text-sm">
              {intervalHours === 1 ? 'Every hour' : `Every ${intervalHours} hours`}
            </span>
          </div>
          <Slider
            value={[intervalHours]}
            onValueChange={(value) => {
              const v = Array.isArray(value) && typeof value[0] === 'number' ? value[0] : intervalHours
              onIntervalChange(v)
            }}
            min={1}
            max={24}
            step={1}
            disabled={!enabled}
          />
          <p className="text-muted-foreground text-xs">How often to search for missing items (1-24 hours)</p>
        </div>
      </CardContent>
    </Card>
  )
}

function BackoffSettingsCard({
  enabled,
  backoffThreshold,
  onBackoffChange,
}: {
  enabled: boolean
  backoffThreshold: number
  onBackoffChange: (v: number) => void
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Backoff Settings</CardTitle>
        <CardDescription>Reduce search frequency for items that consistently fail to find releases</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="backoffThreshold">Backoff Threshold</Label>
          <Input
            id="backoffThreshold"
            type="number"
            value={backoffThreshold}
            onChange={(e) => onBackoffChange(Math.max(1, Number.parseInt(e.target.value) || 1))}
            min={1}
            disabled={!enabled}
          />
          <p className="text-muted-foreground text-xs">
            After this many consecutive failed searches, the item will be searched less frequently. Default: 12 failures before backoff.
          </p>
        </div>
      </CardContent>
    </Card>
  )
}

export function AutoSearchSection() {
  const { data: settings, isLoading, isError, refetch } = useAutoSearchSettings()
  const updateMutation = useUpdateAutoSearchSettings()

  const [enabled, setEnabled] = useState(true)
  const [intervalHours, setIntervalHours] = useState(1)
  const [backoffThreshold, setBackoffThreshold] = useState(12)
  const [prevSettings, setPrevSettings] = useState<typeof settings>(undefined)

  if (settings !== prevSettings) {
    setPrevSettings(settings)
    if (settings) { setEnabled(settings.enabled); setIntervalHours(settings.intervalHours); setBackoffThreshold(settings.backoffThreshold) }
  }

  const handleSave = async () => {
    try { await updateMutation.mutateAsync({ enabled, intervalHours, backoffThreshold }); toast.success('Settings saved') }
    catch { toast.error('Failed to save settings') }
  }

  const hasChanges = settings && (enabled !== settings.enabled || intervalHours !== settings.intervalHours || backoffThreshold !== settings.backoffThreshold)

  if (isLoading) {return <LoadingState variant="list" count={2} />}
  if (isError) {return <ErrorState onRetry={refetch} />}

  return (
    <div className="space-y-6">
      <SearchSettingsCard enabled={enabled} onEnabledChange={setEnabled} intervalHours={intervalHours} onIntervalChange={setIntervalHours} />
      <BackoffSettingsCard enabled={enabled} backoffThreshold={backoffThreshold} onBackoffChange={setBackoffThreshold} />
      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={updateMutation.isPending || !hasChanges}>
          <Save className="mr-2 size-4" />
          Save Changes
        </Button>
      </div>
    </div>
  )
}
