import { History, Save } from 'lucide-react'

import { PageHeader } from '@/components/layout/page-header'
import { ServerSection } from '@/components/settings'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

import { SystemNav } from './system-nav'
import { useHistoryRetention } from './use-history-retention'
import { useServerPage } from './use-server-page'

function HistoryRetentionCard() {
  const h = useHistoryRetention()

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
            checked={h.currentEnabled}
            onCheckedChange={(v) => h.setEnabled(v)}
          />
        </div>
        {h.currentEnabled ? (
          <RetentionDaysInput days={h.currentDays} onChange={h.setDays} />
        ) : null}
        {h.hasChanges ? (
          <Button size="sm" onClick={h.handleSave} disabled={h.isSaving}>
            <Save className="mr-1.5 size-3" />
            Save
          </Button>
        ) : null}
      </CardContent>
    </Card>
  )
}

function RetentionDaysInput({
  days,
  onChange,
}: {
  days: number
  onChange: (v: number) => void
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor="retention-days">Retention period (days)</Label>
      <Input
        id="retention-days"
        type="number"
        min={1}
        max={3650}
        value={days}
        onChange={(e) => onChange(Number.parseInt(e.target.value) || 1)}
        className="w-32"
      />
      <p className="text-muted-foreground text-xs">
        History entries older than this will be automatically deleted daily at 2 AM.
      </p>
    </div>
  )
}

export function ServerPage() {
  const page = useServerPage()

  return (
    <div className="space-y-6">
      <PageHeader
        title="System"
        description="Server configuration and authentication settings"
        breadcrumbs={[{ label: 'Settings', href: '/settings/media' }, { label: 'System' }]}
        actions={
          <Button onClick={page.handleSave} disabled={page.isSaving || !page.hasChanges}>
            <Save className="mr-2 size-4" />
            Save Changes
          </Button>
        }
      />
      <SystemNav />
      <div className="max-w-2xl space-y-6">
        <ServerSection
          port={page.port}
          onPortChange={page.onPortChange}
          logLevel={page.logLevel}
          onLogLevelChange={page.onLogLevelChange}
          logRotation={page.logRotation}
          onLogRotationChange={page.onLogRotationChange}
          externalAccessEnabled={page.externalAccessEnabled}
          onExternalAccessChange={page.onExternalAccessChange}
        />
        <HistoryRetentionCard />
      </div>
    </div>
  )
}
