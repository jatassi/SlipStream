import { useState } from 'react'

import { AlertTriangle, Loader2, Play, Save } from 'lucide-react'
import { toast } from 'sonner'

import { ErrorState } from '@/components/data/ErrorState'
import { LoadingState } from '@/components/data/LoadingState'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Slider } from '@/components/ui/slider'
import { Switch } from '@/components/ui/switch'
import {
  useRssSyncSettings,
  useRssSyncStatus,
  useTriggerRssSync,
  useUpdateRssSyncSettings,
} from '@/hooks'

const formatElapsed = (ms: number) => {
  if (ms < 1000) {
    return `${ms}ms`
  }
  return `${(ms / 1000).toFixed(1)}s`
}

const formatRelativeTime = (dateStr: string) => {
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMin = Math.floor(diffMs / 60_000)
  if (diffMin < 1) {
    return 'Just now'
  }
  if (diffMin < 60) {
    return `${diffMin} minute${diffMin === 1 ? '' : 's'} ago`
  }
  const diffHours = Math.floor(diffMin / 60)
  if (diffHours < 24) {
    return `${diffHours} hour${diffHours === 1 ? '' : 's'} ago`
  }
  const diffDays = Math.floor(diffHours / 24)
  return `${diffDays} day${diffDays === 1 ? '' : 's'} ago`
}

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

  const hasChanges =
    settings && (enabled !== settings.enabled || intervalMin !== settings.intervalMin)

  if (isLoading) {
    return <LoadingState variant="list" count={2} />
  }

  if (isError) {
    return <ErrorState onRetry={refetch} />
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
              <p className="text-muted-foreground text-sm">
                Periodically fetch RSS feeds from indexers and grab matching releases
              </p>
            </div>
            <Switch checked={enabled} onCheckedChange={setEnabled} />
          </div>

          <div className="space-y-4">
            <div className="space-y-2">
              <div className="flex justify-between">
                <Label>Sync Interval</Label>
                <span className="text-muted-foreground text-sm">Every {intervalMin} minutes</span>
              </div>
              <Slider
                value={[intervalMin]}
                onValueChange={(value) => {
                  const v =
                    Array.isArray(value) && typeof value[0] === 'number' ? value[0] : intervalMin
                  setIntervalMin(v)
                }}
                min={10}
                max={120}
                step={5}
                disabled={!enabled}
              />
              <p className="text-muted-foreground text-xs">
                How often to check RSS feeds for new releases (10-120 minutes)
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Last Sync</CardTitle>
          <CardDescription>Status of the most recent RSS sync run</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {status?.lastRun ? (
            <>
              <div className="text-muted-foreground text-sm">
                {formatRelativeTime(status.lastRun)}
              </div>
              {status.error ? (
                <div className="flex items-center gap-2 rounded-md border border-red-500/30 bg-red-500/5 p-3">
                  <AlertTriangle className="size-4 shrink-0 text-red-500" />
                  <p className="text-sm text-red-400">{status.error}</p>
                </div>
              ) : (
                <div className="grid grid-cols-4 gap-4">
                  <div className="text-center">
                    <div className="text-lg font-semibold">{status.totalReleases}</div>
                    <div className="text-muted-foreground text-xs">Releases</div>
                  </div>
                  <div className="text-center">
                    <div className="text-lg font-semibold">{status.matched}</div>
                    <div className="text-muted-foreground text-xs">Matched</div>
                  </div>
                  <div className="text-center">
                    <div className="text-lg font-semibold">{status.grabbed}</div>
                    <div className="text-muted-foreground text-xs">Grabbed</div>
                  </div>
                  <div className="text-center">
                    <div className="text-lg font-semibold">{formatElapsed(status.elapsed)}</div>
                    <div className="text-muted-foreground text-xs">Elapsed</div>
                  </div>
                </div>
              )}
            </>
          ) : (
            <p className="text-muted-foreground text-sm">No sync has been run yet</p>
          )}

          <Button variant="outline" onClick={handleTrigger} disabled={triggerMutation.isPending}>
            {triggerMutation.isPending ? (
              <Loader2 className="mr-2 size-4 animate-spin" />
            ) : (
              <Play className="mr-2 size-4" />
            )}
            Run Now
          </Button>
        </CardContent>
      </Card>

      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={updateMutation.isPending || !hasChanges}>
          <Save className="mr-2 size-4" />
          Save Changes
        </Button>
      </div>
    </div>
  )
}
