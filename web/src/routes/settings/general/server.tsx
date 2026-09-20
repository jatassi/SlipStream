import { Save } from 'lucide-react'

import { Group } from '@/components/grouped-list'
import { usePushBack } from '@/components/layout/use-push-back'
import { Screen } from '@/components/screen/screen'
import { ServerSection } from '@/components/settings'
import { InputRow, SwitchRow } from '@/components/settings/control-row'
import { Button } from '@/components/ui/button'

import { useHistoryRetention } from './use-history-retention'
import { useServerPage } from './use-server-page'

function HistoryRetentionGroup() {
  const h = useHistoryRetention()

  return (
    <Group
      header="History Retention"
      footer="History entries older than the retention period are automatically deleted daily at 2 AM."
    >
      <SwitchRow
        label="Auto-cleanup old history entries"
        checked={h.currentEnabled}
        onCheckedChange={(v) => h.setEnabled(v)}
      />
      {h.currentEnabled ? (
        <InputRow
          label="Retention period (days)"
          type="number"
          min={1}
          max={3650}
          value={h.currentDays}
          // `||` intentional: parseInt of an emptied/invalid field is NaN, which `??` would not catch
          onChange={(e) => h.setDays(Number.parseInt(e.target.value) || 1)}
        />
      ) : undefined}
      {h.hasChanges ? (
        <div className="px-4 py-2">
          <Button className="h-11" size="sm" onClick={h.handleSave} disabled={h.isSaving}>
            <Save className="mr-1.5 size-3" />
            Save
          </Button>
        </div>
      ) : undefined}
    </Group>
  )
}

export function ServerPage() {
  const page = useServerPage()
  const back = usePushBack()

  return (
    <Screen
      title="Server"
      back={back}
      trailing={
        <Button size="sm" onClick={page.handleSave} disabled={page.isSaving || !page.hasChanges}>
          <Save className="mr-2 size-4" />
          Save
        </Button>
      }
    >
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
      <HistoryRetentionGroup />
    </Screen>
  )
}
