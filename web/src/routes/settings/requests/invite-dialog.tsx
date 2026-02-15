import { Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
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
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'

type InviteDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  inviteName: string
  onNameChange: (name: string) => void
  qualityProfileId: number | null
  onQualityProfileChange: (id: number | null) => void
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
        <InviteFormBody {...props} />
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
  const profileLabel = p.qualityProfileId
    ? (p.qualityProfiles?.find((pr) => pr.id === p.qualityProfileId)?.name ?? 'Select profile')
    : 'Default (use global)'

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
        qualityProfileId={p.qualityProfileId}
        onQualityProfileChange={p.onQualityProfileChange}
        qualityProfiles={p.qualityProfiles}
        profileLabel={profileLabel}
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

type ProfileSelectProps = {
  qualityProfileId: number | null
  onQualityProfileChange: (id: number | null) => void
  qualityProfiles: { id: number; name: string }[] | undefined
  profileLabel: string
}

function ProfileSelect({ qualityProfileId, onQualityProfileChange, qualityProfiles, profileLabel }: ProfileSelectProps) {
  return (
    <div className="space-y-2">
      <Label>Quality Profile</Label>
      <Select
        value={qualityProfileId?.toString() ?? ''}
        onValueChange={(value) =>
          onQualityProfileChange(value ? Number.parseInt(value, 10) : null)
        }
      >
        <SelectTrigger>{profileLabel}</SelectTrigger>
        <SelectContent>
          <SelectItem value="">Default (use global)</SelectItem>
          {qualityProfiles?.map((profile) => (
            <SelectItem key={profile.id} value={profile.id.toString()}>
              {profile.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
