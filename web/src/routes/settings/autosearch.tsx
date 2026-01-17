import { useEffect, useState } from 'react'
import { Save } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Slider } from '@/components/ui/slider'
import { LoadingState } from '@/components/data/LoadingState'
import { ErrorState } from '@/components/data/ErrorState'
import { useAutoSearchSettings, useUpdateAutoSearchSettings } from '@/hooks'
import { toast } from 'sonner'

export function AutoSearchSettingsPage() {
  const { data: settings, isLoading, isError, refetch } = useAutoSearchSettings()
  const updateMutation = useUpdateAutoSearchSettings()

  const [enabled, setEnabled] = useState(true)
  const [intervalHours, setIntervalHours] = useState(1)
  const [backoffThreshold, setBackoffThreshold] = useState(12)

  useEffect(() => {
    if (settings) {
      setEnabled(settings.enabled)
      setIntervalHours(settings.intervalHours)
      setBackoffThreshold(settings.backoffThreshold)
    }
  }, [settings])

  const handleSave = async () => {
    try {
      await updateMutation.mutateAsync({
        enabled,
        intervalHours,
        backoffThreshold,
      })
      toast.success('Settings saved')
    } catch {
      toast.error('Failed to save settings')
    }
  }

  const hasChanges = settings && (
    enabled !== settings.enabled ||
    intervalHours !== settings.intervalHours ||
    backoffThreshold !== settings.backoffThreshold
  )

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Release Searching" />
        <LoadingState variant="list" count={2} />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Release Searching" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title="Release Searching"
        description="Configure automatic release searching behavior"
        breadcrumbs={[
          { label: 'Settings', href: '/settings' },
          { label: 'Release Searching' },
        ]}
        actions={
          <Button onClick={handleSave} disabled={updateMutation.isPending || !hasChanges}>
            <Save className="size-4 mr-2" />
            Save Changes
          </Button>
        }
      />

      <div className="space-y-6 max-w-2xl">
        <Card>
          <CardHeader>
            <CardTitle>Automatic Search</CardTitle>
            <CardDescription>
              Scheduled task that searches for missing monitored items
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>Enable Automatic Search</Label>
                <p className="text-sm text-muted-foreground">
                  Periodically search for missing movies and episodes
                </p>
              </div>
              <Switch
                checked={enabled}
                onCheckedChange={setEnabled}
              />
            </div>

            <div className="space-y-4">
              <div className="space-y-2">
                <div className="flex justify-between">
                  <Label>Search Interval</Label>
                  <span className="text-sm text-muted-foreground">
                    {intervalHours === 1 ? 'Every hour' : `Every ${intervalHours} hours`}
                  </span>
                </div>
                <Slider
                  value={[intervalHours]}
                  onValueChange={(value) => {
                    const v = Array.isArray(value) ? value[0] : value
                    setIntervalHours(v)
                  }}
                  min={1}
                  max={24}
                  step={1}
                  disabled={!enabled}
                />
                <p className="text-xs text-muted-foreground">
                  How often to search for missing items (1-24 hours)
                </p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Backoff Settings</CardTitle>
            <CardDescription>
              Reduce search frequency for items that consistently fail to find releases
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="backoffThreshold">Backoff Threshold</Label>
              <Input
                id="backoffThreshold"
                type="number"
                value={backoffThreshold}
                onChange={(e) => setBackoffThreshold(Math.max(1, parseInt(e.target.value) || 1))}
                min={1}
                disabled={!enabled}
              />
              <p className="text-xs text-muted-foreground">
                After this many consecutive failed searches, the item will be searched less frequently.
                Default: 12 failures before backoff.
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
