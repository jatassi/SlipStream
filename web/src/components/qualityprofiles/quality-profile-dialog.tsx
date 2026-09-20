import { FormActions, SheetPresenter } from '@/components/presenter'
import { ControlStack, InputRow, SelectRow } from '@/components/settings/control-row'
import { getEnabledModules } from '@/modules'
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
  defaultModuleType?: string
}

export function QualityProfileDialog({ open, onOpenChange, profile, defaultModuleType }: QualityProfileDialogProps) {
  const state = useQualityProfileDialog({ open, onOpenChange, profile, defaultModuleType })
  const showPreview = state.formData.upgradesEnabled && state.allowedQualities.length >= 2

  return (
    <SheetPresenter
      open={open}
      onOpenChange={onOpenChange}
      title={state.isEditing ? 'Edit Quality Profile' : 'Add Quality Profile'}
      description="Configure quality preferences and attribute filters for downloads."
      wideClassName="sm:max-w-3xl"
      footer={
        <FormActions
          onCancel={() => onOpenChange(false)}
          confirmLabel={state.isEditing ? 'Save' : 'Create'}
          onConfirm={state.handleSubmit}
          confirmDisabled={state.hasAttributeValidationError}
          loading={state.isPending}
        />
      }
    >
      <ProfileFormBody state={state} showPreview={showPreview} />
    </SheetPresenter>
  )
}

type ProfileFormBodyProps = {
  state: ReturnType<typeof useQualityProfileDialog>
  showPreview: boolean
}

function ModuleTypeSelector({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const options = getEnabledModules().map((mod) => ({ value: mod.id, label: mod.name }))
  return <SelectRow label="Module Type" value={value} onChange={onChange} options={options} />
}

function ProfileFormBody({ state, showPreview }: ProfileFormBodyProps) {
  const { formData, cutoffOptions, updateField, toggleQuality, updateItemMode, isEditing } = state

  return (
    <div className="space-y-6 py-2">
      <ControlStack>
        <InputRow
          stacked
          label="Name"
          aria-label="Name"
          placeholder="HD-1080p"
          value={formData.name}
          onChange={(e) => updateField('name', e.target.value)}
        />
        {!isEditing && (
          <ModuleTypeSelector value={formData.moduleType} onChange={(v) => updateField('moduleType', v)} />
        )}
      </ControlStack>

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

      {/* eslint-disable-next-line react/jsx-no-leaked-render -- condition is already a boolean */}
      {showPreview && <UpgradeStrategyPreview
          allowedQualities={formData.items}
          strategy={formData.upgradeStrategy}
          cutoffId={formData.cutoff}
          cutoffOverridesStrategy={formData.cutoffOverridesStrategy}
        />}

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
