import { Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
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
import type { PortalUserWithQuota, QualityProfile } from '@/types'

import { ProfileSelect } from './profile-select'
import { useUserEditDialog } from './use-user-edit-dialog'

type UserEditDialogProps = {
  user: PortalUserWithQuota
  open: boolean
  onOpenChange: (open: boolean) => void
  qualityProfiles: QualityProfile[]
}

export function UserEditDialog({ user, open, onOpenChange, qualityProfiles }: UserEditDialogProps) {
  const state = useUserEditDialog(user, onOpenChange)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Edit User</DialogTitle>
          <DialogDescription>
            Configure settings for {user.username}
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          <EditUserFormBody state={state} qualityProfiles={qualityProfiles} />
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={state.handleSave} disabled={state.isPending}>
            {state.isPending ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
            Save Changes
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

type EditFormProps = {
  state: ReturnType<typeof useUserEditDialog>
  qualityProfiles: QualityProfile[]
}

function EditUserFormBody({ state, qualityProfiles }: EditFormProps) {
  const moduleTypes = [...new Set(qualityProfiles.map((p) => p.moduleType))]

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="username">Username</Label>
        <Input
          id="username"
          type="text"
          value={state.username}
          onChange={(e) => state.setUsername(e.target.value)}
        />
      </div>

      {moduleTypes.map((moduleType) => {
        const filtered = qualityProfiles.filter((p) => p.moduleType === moduleType)
        const label = `${moduleType.charAt(0).toUpperCase() + moduleType.slice(1)} Quality Profile`
        return (
          <ProfileSelect
            key={moduleType}
            label={label}
            value={state.moduleProfileSettings[moduleType] ?? null}
            onChange={(id) => state.setModuleProfile(moduleType, id)}
            qualityProfiles={filtered}
          />
        )
      })}

      <div className="flex items-center space-x-2">
        <Checkbox
          id="autoApprove"
          checked={state.autoApprove}
          onCheckedChange={(checked) => state.setAutoApprove(checked)}
        />
        <Label htmlFor="autoApprove">Auto-approve requests</Label>
      </div>
    </div>
  )
}
