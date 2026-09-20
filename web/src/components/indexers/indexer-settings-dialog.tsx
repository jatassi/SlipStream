import { RotateCcw, Settings, TrendingDown, TrendingUp } from 'lucide-react'

import { FormActions, SheetPresenter } from '@/components/presenter'
import { ControlStack, InputRow, SelectRow } from '@/components/settings/control-row'
import { Button } from '@/components/ui/button'
import type { ContentType, ProwlarrIndexerWithSettings } from '@/types'

import { useIndexerSettingsDialog } from './use-indexer-settings-dialog'

const CONTENT_TYPE_OPTIONS = [
  { value: 'both', label: 'Both' },
  { value: 'movies', label: 'Movies Only' },
  { value: 'series', label: 'Series Only' },
]

export function IndexerSettingsDialog({ indexer }: { indexer: ProwlarrIndexerWithSettings }) {
  const dialog = useIndexerSettingsDialog(indexer)

  return (
    <>
      <Button
        variant="ghost"
        size="icon"
        aria-label={`Settings for ${indexer.name}`}
        className="size-8"
        onClick={() => dialog.handleOpenChange(true)}
      >
        <Settings className="size-4" />
      </Button>
      <SheetPresenter
        open={dialog.open}
        onOpenChange={dialog.handleOpenChange}
        title={`Settings for ${indexer.name}`}
        description="Configure priority and content type filtering for this indexer"
        footer={
          <FormActions
            onCancel={() => dialog.handleOpenChange(false)}
            confirmLabel="Save"
            onConfirm={dialog.handleSave}
            loading={dialog.isSaving}
          />
        }
      >
        <SettingsForm dialog={dialog} />
      </SheetPresenter>
    </>
  )
}

function SettingsForm({ dialog }: { dialog: ReturnType<typeof useIndexerSettingsDialog> }) {
  return (
    <div className="space-y-4 py-2">
      <ControlStack footer="Lower priority indexers are preferred during deduplication.">
        <InputRow
          label="Priority (1-50)"
          aria-label="Priority (1-50)"
          type="number"
          min={1}
          max={50}
          value={dialog.priority}
          onChange={(e) => dialog.handlePriorityChange(e.target.value)}
        />
      </ControlStack>
      <ControlStack footer="Filter this indexer to only be used for specific content types.">
        <SelectRow
          label="Content Type"
          value={dialog.contentType}
          onChange={(value) => dialog.setContentType(value as ContentType)}
          options={CONTENT_TYPE_OPTIONS}
        />
      </ControlStack>
      <StatisticsSection
        settings={dialog.settings}
        onReset={dialog.handleResetStats}
        isResetting={dialog.isResetting}
      />
    </div>
  )
}

type IndexerSettings = ProwlarrIndexerWithSettings['settings']

function StatisticsSection({
  settings,
  onReset,
  isResetting,
}: {
  settings: IndexerSettings
  onReset: () => void
  isResetting: boolean
}) {
  if (!settings || (settings.successCount === 0 && settings.failureCount === 0)) {
    return null
  }

  return (
    <div className="space-y-2 rounded-card border p-3">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium">Statistics</span>
        <Button variant="ghost" size="sm" onClick={onReset} disabled={isResetting}>
          <RotateCcw className="mr-1 size-3" />
          Reset
        </Button>
      </div>
      <div className="flex gap-4 text-sm">
        <div className="flex items-center gap-1">
          <TrendingUp className="size-4 text-green-500" />
          <span>{settings.successCount} successful</span>
        </div>
        <div className="flex items-center gap-1">
          <TrendingDown className="size-4 text-red-500" />
          <span>{settings.failureCount} failed</span>
        </div>
      </div>
      {Boolean(settings.lastFailureReason) && (
        <p className="text-muted-foreground text-xs">Last failure: {settings.lastFailureReason}</p>
      )}
    </div>
  )
}
