import { FormActions, SheetPresenter } from '@/components/presenter'
import type { SlotConflict } from '@/types'

import { ProfileEditorCard } from './profile-editor-card'
import { useResolveConfigModal } from './use-resolve-config-modal'

type ResolveConfigModalProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  conflicts: SlotConflict[]
  onResolved: () => void
}

export function ResolveConfigModal(props: ResolveConfigModalProps) {
  const state = useResolveConfigModal(props)

  return (
    <SheetPresenter
      open={props.open}
      onOpenChange={props.onOpenChange}
      title="Resolve Profile Conflicts"
      description="Edit the conflicting profiles to make them mutually exclusive. Conflicting attributes are highlighted in orange."
      wideClassName="sm:max-w-6xl"
      footer={
        <SaveFooter
          saving={state.saving}
          onCancel={() => props.onOpenChange(false)}
          onSave={state.handleSaveAll}
        />
      }
    >
      <ProfileGrid {...state} />
    </SheetPresenter>
  )
}

function ProfileGrid({
  profilesToEdit,
  profileForms,
  slots,
  hdrOptions,
  attributeOptions,
  conflictingAttributes,
  updateProfileForm,
  updateItemMode,
  toggleQuality,
}: ReturnType<typeof useResolveConfigModal>) {
  return (
    <div
      className="grid gap-6 py-2 max-sm:!grid-cols-1"
      style={{ gridTemplateColumns: `repeat(${Math.min(profilesToEdit.length, 3)}, 1fr)` }}
    >
      {profilesToEdit.slice(0, 3).map((profile) => (
        <ProfileEditorCard
          key={profile.id}
          profile={profile}
          formData={profileForms[profile.id]}
          slots={slots}
          hdrOptions={hdrOptions}
          attributeOptions={attributeOptions}
          conflictingAttributes={conflictingAttributes}
          onUpdateField={(field, value) => updateProfileForm(profile.id, field, value)}
          onUpdateItemMode={(sf, value, mode) =>
            updateItemMode(profile.id, { settingsField: sf, value, mode })
          }
          onToggleQuality={(qualityId) => toggleQuality(profile.id, qualityId)}
        />
      ))}
    </div>
  )
}

function SaveFooter({
  saving,
  onCancel,
  onSave,
}: {
  saving: boolean
  onCancel: () => void
  onSave: () => void
}) {
  return (
    <FormActions onCancel={onCancel} confirmLabel="Save All" onConfirm={onSave} loading={saving} />
  )
}
