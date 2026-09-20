import { Loader2 } from 'lucide-react'

import { SheetPresenter } from '@/components/presenter'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import type { PortalUserWithQuota, QualityProfile } from '@/types'

import { ProfileSelect } from './profile-select'
import { useUserEditForm } from './use-user-edit-form'

type UserEditSheetProps = {
  user: PortalUserWithQuota
  open: boolean
  onOpenChange: (open: boolean) => void
  qualityProfiles: QualityProfile[]
}

function moduleLabel(moduleType: string): string {
  return `${moduleType.charAt(0).toUpperCase() + moduleType.slice(1)} Quality Profile`
}

function EditUserForm({
  form,
  qualityProfiles,
}: {
  form: ReturnType<typeof useUserEditForm>
  qualityProfiles: QualityProfile[]
}) {
  const moduleTypes = [...new Set(qualityProfiles.map((profile) => profile.moduleType))]

  return (
    <div className="space-y-4 pb-2">
      <div className="space-y-2">
        <Label htmlFor="edit-username">Username</Label>
        <Input
          id="edit-username"
          type="text"
          className="h-11 text-base"
          value={form.username}
          onChange={(e) => form.setUsername(e.target.value)}
        />
      </div>

      {moduleTypes.map((moduleType) => (
        <ProfileSelect
          key={moduleType}
          label={moduleLabel(moduleType)}
          value={form.moduleProfileSettings[moduleType] ?? null}
          onChange={(id) => form.setModuleProfile(moduleType, id)}
          qualityProfiles={qualityProfiles.filter((profile) => profile.moduleType === moduleType)}
        />
      ))}

      <div className="flex min-h-tap items-center justify-between gap-3">
        <Label htmlFor="edit-auto-approve">Auto-approve requests</Label>
        <Switch
          id="edit-auto-approve"
          aria-label="Auto-approve requests"
          checked={form.autoApprove}
          onCheckedChange={form.setAutoApprove}
        />
      </div>
    </div>
  )
}

export function UserEditSheet({ user, open, onOpenChange, qualityProfiles }: UserEditSheetProps) {
  const form = useUserEditForm(user, onOpenChange)

  return (
    <SheetPresenter
      open={open}
      onOpenChange={onOpenChange}
      title="Edit User"
      description={`Configure settings for ${user.username}`}
      footer={
        <Button className="h-11 w-full" onClick={form.handleSave} disabled={form.isPending}>
          {form.isPending ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
          Save Changes
        </Button>
      }
    >
      <EditUserForm form={form} qualityProfiles={qualityProfiles} />
    </SheetPresenter>
  )
}
