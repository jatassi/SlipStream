import { useState } from 'react'

import { Loader2, Play, Save } from 'lucide-react'
import { toast } from 'sonner'

import { Group, Row } from '@/components/grouped-list'
import { SliderRow, SwitchRow } from '@/components/settings/control-row'
import { SectionError, SectionLoading } from '@/components/settings/section-state'
import { Button } from '@/components/ui/button'
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

type SyncStatus = {
  lastRun?: string
  error?: string
  totalReleases?: number
  matched?: number
  grabbed?: number
  elapsed?: number
}

function LastSyncRows({ status }: { status: SyncStatus | undefined }) {
  if (!status?.lastRun) {
    return <Row title="Last run" trailing="Never" />
  }

  if (status.error) {
    return (
      <>
        <Row title="Last run" trailing={formatRelativeTime(status.lastRun)} />
        <Row title="Error" subtitle={status.error} tone="warning" />
      </>
    )
  }

  return (
    <>
      <Row title="Last run" trailing={formatRelativeTime(status.lastRun)} />
      <Row title="Releases" trailing={String(status.totalReleases ?? 0)} />
      <Row title="Matched" trailing={String(status.matched ?? 0)} />
      <Row title="Grabbed" trailing={String(status.grabbed ?? 0)} />
      <Row title="Elapsed" trailing={formatElapsed(status.elapsed ?? 0)} />
    </>
  )
}

function useRssSyncForm() {
  const { data: settings, isLoading, isError, refetch } = useRssSyncSettings()
  const updateMutation = useUpdateRssSyncSettings()

  const [enabled, setEnabled] = useState(true)
  const [intervalMin, setIntervalMin] = useState(15)
  const [prevSettings, setPrevSettings] = useState<typeof settings>(undefined)

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

  const hasChanges = Boolean(
    settings && (enabled !== settings.enabled || intervalMin !== settings.intervalMin),
  )

  return {
    enabled,
    setEnabled,
    intervalMin,
    setIntervalMin,
    handleSave,
    hasChanges,
    isSaving: updateMutation.isPending,
    isLoading,
    isError,
    refetch,
  }
}

function RunNowRow() {
  const triggerMutation = useTriggerRssSync()

  const handleTrigger = async () => {
    try {
      await triggerMutation.mutateAsync()
      toast.success('RSS sync started')
    } catch {
      toast.error('Failed to trigger RSS sync')
    }
  }

  return (
    <Row
      title={<span className="text-primary">Run Now</span>}
      onClick={() => void handleTrigger()}
      trailing={
        triggerMutation.isPending ? (
          <Loader2 className="size-4 animate-spin" />
        ) : (
          <Play className="size-4" />
        )
      }
    />
  )
}

export function RssSyncSection() {
  const form = useRssSyncForm()
  const { data: status } = useRssSyncStatus()

  if (form.isLoading) {
    return <SectionLoading count={2} />
  }
  if (form.isError) {
    return <SectionError onRetry={form.refetch} />
  }

  return (
    <>
      <Group
        header="Feed Schedule"
        footer="Periodically fetch RSS feeds from indexers and grab matching releases, every 10 to 120 minutes."
      >
        <SwitchRow label="Enable RSS Sync" checked={form.enabled} onCheckedChange={form.setEnabled} />
        <SliderRow
          label="Sync Interval"
          value={form.intervalMin}
          display={`Every ${form.intervalMin} minutes`}
          onChange={form.setIntervalMin}
          min={10}
          max={120}
          step={5}
          disabled={!form.enabled}
        />
      </Group>
      <Group header="Last Sync" footer="Status of the most recent RSS sync run.">
        <LastSyncRows status={status} />
        <RunNowRow />
      </Group>
      <div className="px-screen flex justify-end">
        <Button className="h-11" onClick={form.handleSave} disabled={form.isSaving || !form.hasChanges}>
          <Save className="mr-2 size-4" />
          Save Changes
        </Button>
      </div>
    </>
  )
}
