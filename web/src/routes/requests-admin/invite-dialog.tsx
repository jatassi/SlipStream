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
import type { QualityProfile } from '@/types'

import { ProfileSelect } from './profile-select'

type InviteDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  inviteName: string
  onNameChange: (name: string) => void
  moduleSettings: Record<string, number | null>
  onModuleProfileChange: (moduleType: string, profileId: number | null) => void
  autoApprove: boolean
  onAutoApproveChange: (checked: boolean) => void
  qualityProfiles: QualityProfile[] | undefined
  isPending: boolean
  onSubmit: () => void
}

export function InviteDialog(props: InviteDialogProps) {
  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Invite User</DialogTitle>
          <DialogDescription>
            Create an invitation for a new user to join the request portal. The name you enter will
            become their username.
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          <InviteFormBody {...props} />
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={() => props.onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={props.onSubmit} disabled={props.isPending}>
            {props.isPending ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
            Create Invitation
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function InviteFormBody(p: InviteDialogProps) {
  const profiles = p.qualityProfiles ?? []
  const moduleTypes = [...new Set(profiles.map((prof) => prof.moduleType))]

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="name">Name</Label>
        <Input
          id="name"
          type="text"
          placeholder="John"
          value={p.inviteName}
          onChange={(e) => p.onNameChange(e.target.value)}
        />
      </div>

      {moduleTypes.map((moduleType) => {
        const filtered = profiles.filter((prof) => prof.moduleType === moduleType)
        const label = `${moduleType.charAt(0).toUpperCase() + moduleType.slice(1)} Quality Profile`
        return (
          <ProfileSelect
            key={moduleType}
            label={label}
            value={p.moduleSettings[moduleType] ?? null}
            onChange={(id) => p.onModuleProfileChange(moduleType, id)}
            qualityProfiles={filtered}
          />
        )
      })}

      <div className="flex items-center space-x-2">
        <Checkbox
          id="inviteAutoApprove"
          checked={p.autoApprove}
          onCheckedChange={(checked) => p.onAutoApproveChange(checked)}
        />
        <Label htmlFor="inviteAutoApprove">Auto-approve requests</Label>
      </div>
    </div>
  )
}
