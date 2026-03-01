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

import { ProfileSelect } from './profile-select'

type InviteDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  inviteName: string
  onNameChange: (name: string) => void
  movieQualityProfileId: number | null
  onMovieQualityProfileChange: (id: number | null) => void
  tvQualityProfileId: number | null
  onTvQualityProfileChange: (id: number | null) => void
  autoApprove: boolean
  onAutoApproveChange: (checked: boolean) => void
  qualityProfiles: { id: number; name: string }[] | undefined
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

      <ProfileSelect
        label="Movie Quality Profile"
        value={p.movieQualityProfileId}
        onChange={p.onMovieQualityProfileChange}
        qualityProfiles={p.qualityProfiles}
      />

      <ProfileSelect
        label="TV Quality Profile"
        value={p.tvQualityProfileId}
        onChange={p.onTvQualityProfileChange}
        qualityProfiles={p.qualityProfiles}
      />

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

