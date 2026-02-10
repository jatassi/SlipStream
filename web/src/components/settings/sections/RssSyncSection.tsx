import { useState } from 'react'
import { Save, Play, Loader2, AlertTriangle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Slider } from '@/components/ui/slider'
import { LoadingState } from '@/components/data/LoadingState'
import { ErrorState } from '@/components/data/ErrorState'
import { useRssSyncSettings, useUpdateRssSyncSettings, useRssSyncStatus, useTriggerRssSync } from '@/hooks'
import { toast } from 'sonner'

export function RssSyncSection() {
  const { data: settings, isLoading, isError, refetch } = useRssSyncSettings()
  const updateMutation = useUpdateRssSyncSettings()
  const { data: status } = useRssSyncStatus()
  const triggerMutation = useTriggerRssSync()

  const [enabled, setEnabled] = useState(true)
  const [intervalMin, setIntervalMin] = useState(15)
  const [prevSettings, setPrevSettings] = useState(settings)

  if (settings !== prevSettings) {
    setPrevSettings(settings)
    if (settings) {
      setEnabled(settings.enabled)
      setIntervalMin(settings.intervalMin)
    }
  }

  const handleSave = async () => {
    try {
      await updateMutation.mutateAsync({ enabled, intervalMin })
      toast.success('Settings saved')
    } catch {
      toast.error('Failed to save settings')
    }
  }

  const handleTrigger = async () => {
    try {
      await triggerMutation.mutateAsync()
      toast.success('RSS sync started')
    } catch {
      toast.error('Failed to trigger RSS sync')
    }
  }

  const hasChanges = settings && (
    enabled !== settings.enabled ||
    intervalMin !== settings.intervalMin
  )

  if (isLoading) {
    return <LoadingState variant="list" count={2} />
  }

  if (isError) {
    return <ErrorState onRetry={refetch} />
  }

  const formatElapsed = (ms: number) => {
    if (ms < 1000) return `${ms}ms`
    return `${(ms / 1000).toFixed(1)}s`
  }

  const formatRelativeTime = (dateStr: string) => {
    const date = new Date(dateStr)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMin = Math.floor(diffMs / 60000)
    if (diffMin < 1) return 'Just now'
    if (diffMin < 60) return `${diffMin} minute${diffMin === 1 ? '' : 's'} ago`
    const diffHours = Math.floor(diffMin / 60)
    if (diffHours < 24) return `${diffHours} hour${diffHours === 1 ? '' : 's'} ago`
    const diffDays = Math.floor(diffHours / 24)
    return `${diffDays} day${diffDays === 1 ? '' : 's'} ago`
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>RSS Sync</CardTitle>
          <CardDescription>
            Periodically fetch RSS feeds from indexers and grab matching releases
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>Enable RSS Sync</Label>
              <p className="text-sm text-muted-foreground">
                Periodically fetch RSS feeds from indexers and grab matching releases
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
                <Label>Sync Interval</Label>
                <span className="text-sm text-muted-foreground">
                  Every {intervalMin} minutes
                </span>
              </div>
              <Slider
                value={[intervalMin]}
                onValueChange={(value) => {
                  const v = Array.isArray(value) ? value[0] : value
                  setIntervalMin(v)
                }}
                min={10}
                max={120}
                step={5}
                disabled={!enabled}
              />
              <p className="text-xs text-muted-foreground">
                How often to check RSS feeds for new releases (10-120 minutes)
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Last Sync</CardTitle>
          <CardDescription>
            Status of the most recent RSS sync run
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {status?.lastRun ? (
            <>
              <div className="text-sm text-muted-foreground">
                {formatRelativeTime(status.lastRun)}
              </div>
              {status.error ? (
                <div className="flex items-center gap-2 rounded-md border border-red-500/30 bg-red-500/5 p-3">
                  <AlertTriangle className="size-4 text-red-500 shrink-0" />
                  <p className="text-sm text-red-400">{status.error}</p>
                </div>
              ) : (
                <div className="grid grid-cols-4 gap-4">
                  <div className="text-center">
                    <div className="text-lg font-semibold">{status.totalReleases}</div>
                    <div className="text-xs text-muted-foreground">Releases</div>
                  </div>
                  <div className="text-center">
                    <div className="text-lg font-semibold">{status.matched}</div>
                    <div className="text-xs text-muted-foreground">Matched</div>
                  </div>
                  <div className="text-center">
                    <div className="text-lg font-semibold">{status.grabbed}</div>
                    <div className="text-xs text-muted-foreground">Grabbed</div>
                  </div>
                  <div className="text-center">
                    <div className="text-lg font-semibold">{formatElapsed(status.elapsed)}</div>
                    <div className="text-xs text-muted-foreground">Elapsed</div>
                  </div>
                </div>
              )}
            </>
          ) : (
            <p className="text-sm text-muted-foreground">No sync has been run yet</p>
          )}

          <Button
            variant="outline"
            onClick={handleTrigger}
            disabled={triggerMutation.isPending}
          >
            {triggerMutation.isPending ? (
              <Loader2 className="size-4 mr-2 animate-spin" />
            ) : (
              <Play className="size-4 mr-2" />
            )}
            Run Now
          </Button>
        </CardContent>
      </Card>

      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={updateMutation.isPending || !hasChanges}>
          <Save className="size-4 mr-2" />
          Save Changes
        </Button>
      </div>
    </div>
  )
}
