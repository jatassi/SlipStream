import { Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import type { QualityProfile } from '@/types'

import { AttributeFilters } from './attribute-filters'
import { QualityChecklist } from './quality-checklist'
import { UpgradeSettings } from './upgrade-settings'
import { UpgradeStrategyPreview } from './upgrade-strategy-preview'
import { useQualityProfileDialog } from './use-quality-profile-dialog'

const MODULE_TYPE_OPTIONS = [
  { value: 'movie', label: 'Movie' },
  { value: 'tv', label: 'TV' },
]

type QualityProfileDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  profile?: QualityProfile | null
  defaultModuleType?: string
}

export function QualityProfileDialog({ open, onOpenChange, profile, defaultModuleType }: QualityProfileDialogProps) {
  const state = useQualityProfileDialog({ open, onOpenChange, profile, defaultModuleType })
  const showPreview = state.formData.upgradesEnabled && state.allowedQualities.length >= 2

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>
            {state.isEditing ? 'Edit Quality Profile' : 'Add Quality Profile'}
          </DialogTitle>
          <DialogDescription>
            Configure quality preferences and attribute filters for downloads.
          </DialogDescription>
        </DialogHeader>

        <DialogBody>
          <ProfileFormBody state={state} showPreview={showPreview} />
        </DialogBody>

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

type ProfileFormBodyProps = {
  state: ReturnType<typeof useQualityProfileDialog>
  showPreview: boolean
}

function ModuleTypeSelector({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <div className="space-y-2">
      <Label htmlFor="module-type">Module Type</Label>
      <Select value={value} onValueChange={(v) => v && onChange(v)}>
        <SelectTrigger id="module-type">
          {MODULE_TYPE_OPTIONS.find((o) => o.value === value)?.label ?? 'Select module type...'}
        </SelectTrigger>
        <SelectContent>
          {MODULE_TYPE_OPTIONS.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

function ProfileFormBody({ state, showPreview }: ProfileFormBodyProps) {
  const { formData, cutoffOptions, updateField, toggleQuality, updateItemMode, isEditing } = state

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

      {!isEditing && (
        <ModuleTypeSelector value={formData.moduleType} onChange={(v) => updateField('moduleType', v)} />
      )}

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

      {showPreview ? <UpgradeStrategyPreview
          allowedQualities={formData.items}
          strategy={formData.upgradeStrategy}
          cutoffId={formData.cutoff}
          cutoffOverridesStrategy={formData.cutoffOverridesStrategy}
        /> : null}

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
