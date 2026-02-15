import { Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { QualityProfile } from '@/types'

import { AttributeFilters } from './attribute-filters'
import { QualityChecklist } from './quality-checklist'
import { UpgradeSettings } from './upgrade-settings'
import { UpgradeStrategyPreview } from './upgrade-strategy-preview'
import { useQualityProfileDialog } from './use-quality-profile-dialog'

type QualityProfileDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  profile?: QualityProfile | null
}

export function QualityProfileDialog({ open, onOpenChange, profile }: QualityProfileDialogProps) {
  const state = useQualityProfileDialog(open, onOpenChange, profile)
  const showPreview = state.formData.upgradesEnabled && state.allowedQualities.length >= 2

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>
            {state.isEditing ? 'Edit Quality Profile' : 'Add Quality Profile'}
          </DialogTitle>
          <DialogDescription>
            Configure quality preferences and attribute filters for downloads.
          </DialogDescription>
        </DialogHeader>

        <DialogBody state={state} showPreview={showPreview} />

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={state.handleSubmit}
            disabled={state.isPending || state.hasAttributeValidationError}
          >
            {state.isPending ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
            {state.isEditing ? 'Save' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

type DialogBodyProps = {
  state: ReturnType<typeof useQualityProfileDialog>
  showPreview: boolean
}

function DialogBody({ state, showPreview }: DialogBodyProps) {
  const { formData, cutoffOptions, updateField, toggleQuality, updateItemMode } = state

  return (
    <div className="space-y-6 py-4">
      <div className="space-y-2">
        <Label htmlFor="name">Name</Label>
        <Input
          id="name"
          placeholder="HD-1080p"
          value={formData.name}
          onChange={(e) => updateField('name', e.target.value)}
        />
      </div>

      <QualityChecklist items={formData.items} onToggle={toggleQuality} />

      <UpgradeSettings
        upgradesEnabled={formData.upgradesEnabled}
        upgradeStrategy={formData.upgradeStrategy}
        cutoffOverridesStrategy={formData.cutoffOverridesStrategy}
        allowAutoApprove={formData.allowAutoApprove}
        cutoff={formData.cutoff}
        cutoffOptions={cutoffOptions}
        onFieldChange={updateField}
      />

      {showPreview ? (
        <UpgradeStrategyPreview
          allowedQualities={formData.items}
          strategy={formData.upgradeStrategy}
          cutoffId={formData.cutoff}
          cutoffOverridesStrategy={formData.cutoffOverridesStrategy}
        />
      ) : null}

      <AttributeFilters
        hdrSettings={formData.hdrSettings}
        videoCodecSettings={formData.videoCodecSettings}
        audioCodecSettings={formData.audioCodecSettings}
        audioChannelSettings={formData.audioChannelSettings}
        hdrOptions={state.hdrOptions}
        disabledHdrItems={state.disabledHdrItems}
        attributeOptions={state.attributeOptions}
        attributeValidation={state.attributeValidation}
        onItemModeChange={updateItemMode}
      />
    </div>
  )
}
