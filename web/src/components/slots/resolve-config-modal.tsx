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
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-6xl">
        <DialogHeader>
          <DialogTitle>Resolve Profile Conflicts</DialogTitle>
          <DialogDescription>
            Edit the conflicting profiles to make them mutually exclusive. Conflicting attributes
            are highlighted in orange.
          </DialogDescription>
        </DialogHeader>

        <ProfileGrid {...state} />

        <SaveFooter
          saving={state.saving}
          onCancel={() => props.onOpenChange(false)}
          onSave={state.handleSaveAll}
        />
      </DialogContent>
    </Dialog>
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
      className="grid gap-6"
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
    <DialogFooter>
      <Button variant="outline" onClick={onCancel}>
        Cancel
      </Button>
      <Button onClick={onSave} disabled={saving}>
        {saving ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
        Save All
      </Button>
    </DialogFooter>
  )
}
