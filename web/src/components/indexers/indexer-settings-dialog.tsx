import { Loader2, RotateCcw, Settings, TrendingDown, TrendingUp } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import type { ContentType, ProwlarrIndexerWithSettings } from '@/types'
import { ContentTypeLabels } from '@/types'

import { useIndexerSettingsDialog } from './use-indexer-settings-dialog'

export function IndexerSettingsDialog({ indexer }: { indexer: ProwlarrIndexerWithSettings }) {
  const dialog = useIndexerSettingsDialog(indexer)

  return (
    <Dialog open={dialog.open} onOpenChange={dialog.handleOpenChange}>
      <DialogTrigger render={<Button variant="ghost" size="icon" className="size-8" />}>
        <Settings className="size-4" />
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Settings for {indexer.name}</DialogTitle>
          <DialogDescription>
            Configure priority and content type filtering for this indexer
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <PriorityField priority={dialog.priority} onChange={dialog.handlePriorityChange} />
          <ContentTypeField
            contentType={dialog.contentType}
            onChange={(v) => dialog.setContentType(v as ContentType)}
          />
          <StatisticsSection
            settings={dialog.settings}
            onReset={dialog.handleResetStats}
            isResetting={dialog.isResetting}
          />
        </div>

        <DialogFooter>
          <DialogClose render={<Button variant="outline" />}>Cancel</DialogClose>
          <Button onClick={dialog.handleSave} disabled={dialog.isSaving}>
            {dialog.isSaving ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function PriorityField({ priority, onChange }: { priority: number; onChange: (value: string) => void }) {
  return (
    <div className="space-y-2">
      <Label htmlFor="priority">Priority (1-50)</Label>
      <Input
        id="priority"
        type="number"
        min={1}
        max={50}
        value={priority}
        onChange={(e) => onChange(e.target.value)}
      />
      <p className="text-muted-foreground text-xs">
        Lower priority indexers are preferred during deduplication
      </p>
    </div>
  )
}

function ContentTypeField({
  contentType,
  onChange,
}: {
  contentType: string
  onChange: (value: string) => void
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor="contentType">Content Type</Label>
      <Select value={contentType} onValueChange={(v) => v && onChange(v)}>
        <SelectTrigger id="contentType">
          {ContentTypeLabels[contentType as keyof typeof ContentTypeLabels]}
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="both">Both</SelectItem>
          <SelectItem value="movies">Movies Only</SelectItem>
          <SelectItem value="series">Series Only</SelectItem>
        </SelectContent>
      </Select>
      <p className="text-muted-foreground text-xs">
        Filter this indexer to only be used for specific content types
      </p>
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
    <div className="space-y-2 rounded-lg border p-3">
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
      {settings.lastFailureReason ? (
        <p className="text-muted-foreground text-xs">
          Last failure: {settings.lastFailureReason}
        </p>
      ) : null}
    </div>
  )
}
