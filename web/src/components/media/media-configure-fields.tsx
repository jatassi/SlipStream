import { Check } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import type { QualityProfile, RootFolder } from '@/types'

export type FolderSelectProps = {
  rootFolderId: string
  rootFolders: RootFolder[] | undefined
  onChange: (v: string) => void
}

export function FolderSelect({ rootFolderId, rootFolders, onChange }: FolderSelectProps) {
  const label =
    rootFolders?.find((f) => f.id === Number.parseInt(rootFolderId))?.name ??
    'Select a root folder'

  return (
    <div className="space-y-2">
      <Label htmlFor="rootFolder">Root Folder *</Label>
      <Select value={rootFolderId} onValueChange={(v) => { if (v) { onChange(v) } }}>
        <SelectTrigger>{label}</SelectTrigger>
        <SelectContent>
          {rootFolders?.map((folder) => (
            <SelectItem key={folder.id} value={String(folder.id)}>
              {folder.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

export type ProfileSelectProps = {
  qualityProfileId: string
  qualityProfiles: QualityProfile[] | undefined
  onChange: (v: string) => void
}

export function ProfileSelect({ qualityProfileId, qualityProfiles, onChange }: ProfileSelectProps) {
  const label =
    qualityProfiles?.find((p) => p.id === Number.parseInt(qualityProfileId))?.name ??
    'Select a quality profile'

  return (
    <div className="space-y-2">
      <Label htmlFor="qualityProfile">Quality Profile *</Label>
      <Select value={qualityProfileId} onValueChange={(v) => { if (v) { onChange(v) } }}>
        <SelectTrigger>{label}</SelectTrigger>
        <SelectContent>
          {qualityProfiles?.map((profile) => (
            <SelectItem key={profile.id} value={String(profile.id)}>
              {profile.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

export type ToggleFieldProps = {
  label: string
  description: string
  checked: boolean
  onChange: (checked: boolean) => void
}

export function ToggleField({ label, description, checked, onChange }: ToggleFieldProps) {
  return (
    <div className="flex items-center justify-between">
      <div className="space-y-0.5">
        <Label>{label}</Label>
        <p className="text-muted-foreground text-sm">{description}</p>
      </div>
      <Switch checked={checked} onCheckedChange={onChange} />
    </div>
  )
}

export type FormActionsProps = {
  rootFolderId: string
  qualityProfileId: string
  isPending: boolean
  onBack: () => void
  onAdd: () => void
  addLabel: string
}

export function FormActions({ rootFolderId, qualityProfileId, isPending, onBack, onAdd, addLabel }: FormActionsProps) {
  return (
    <div className="flex justify-end gap-2">
      <Button variant="outline" onClick={onBack}>
        Back
      </Button>
      <Button onClick={onAdd} disabled={!rootFolderId || !qualityProfileId || isPending}>
        <Check className="mr-2 size-4" />
        {addLabel}
      </Button>
    </div>
  )
}
