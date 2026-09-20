import { Loader2 } from 'lucide-react'

import { SheetPresenter } from '@/components/presenter'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import type { QualityProfile } from '@/types'

import { ProfileSelect } from './profile-select'

type InviteSheetProps = {
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

function moduleLabel(moduleType: string): string {
  return `${moduleType.charAt(0).toUpperCase() + moduleType.slice(1)} Quality Profile`
}

function InviteForm(props: InviteSheetProps) {
  const profiles = props.qualityProfiles ?? []
  const moduleTypes = [...new Set(profiles.map((profile) => profile.moduleType))]

  return (
    <div className="space-y-4 pb-2">
      <div className="space-y-2">
        <Label htmlFor="invite-name">Name</Label>
        <Input
          id="invite-name"
          type="text"
          placeholder="John"
          className="h-11 text-base"
          value={props.inviteName}
          onChange={(e) => props.onNameChange(e.target.value)}
        />
      </div>

      {moduleTypes.map((moduleType) => (
        <ProfileSelect
          key={moduleType}
          label={moduleLabel(moduleType)}
          value={props.moduleSettings[moduleType] ?? null}
          onChange={(id) => props.onModuleProfileChange(moduleType, id)}
          qualityProfiles={profiles.filter((profile) => profile.moduleType === moduleType)}
        />
      ))}

      <div className="flex min-h-tap items-center justify-between gap-3">
        <Label htmlFor="invite-auto-approve">Auto-approve requests</Label>
        <Switch
          id="invite-auto-approve"
          aria-label="Auto-approve requests"
          checked={props.autoApprove}
          onCheckedChange={props.onAutoApproveChange}
        />
      </div>
    </div>
  )
}

export function InviteSheet(props: InviteSheetProps) {
  return (
    <SheetPresenter
      open={props.open}
      onOpenChange={props.onOpenChange}
      title="Invite User"
      description="The name you enter becomes their portal username."
      footer={
        <Button className="h-11 w-full" onClick={props.onSubmit} disabled={props.isPending}>
          {props.isPending ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
          Create Invitation
        </Button>
      }
    >
      <InviteForm {...props} />
    </SheetPresenter>
  )
}
