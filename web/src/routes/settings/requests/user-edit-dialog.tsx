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
import type { PortalUserWithQuota } from '@/types'

import { useUserEditDialog } from './use-user-edit-dialog'

type UserEditDialogProps = {
  user: PortalUserWithQuota
  open: boolean
  onOpenChange: (open: boolean) => void
  qualityProfiles: { id: number; name: string }[]
}

export function UserEditDialog({ user, open, onOpenChange, qualityProfiles }: UserEditDialogProps) {
  const state = useUserEditDialog(user, onOpenChange)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Edit User</DialogTitle>
          <DialogDescription>
            Configure settings for {user.displayName ?? user.username}
          </DialogDescription>
        </DialogHeader>
        <EditUserFormBody state={state} qualityProfiles={qualityProfiles} />
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
  qualityProfiles: { id: number; name: string }[]
}

function EditUserFormBody({ state, qualityProfiles }: EditFormProps) {
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

      <EditProfileSelect state={state} qualityProfiles={qualityProfiles} />

      <div className="flex items-center space-x-2">
        <Checkbox
          id="autoApprove"
          checked={state.autoApprove}
          onCheckedChange={(checked) => state.setAutoApprove(checked)}
        />
        <Label htmlFor="autoApprove">Auto-approve requests</Label>
      </div>

      <QuotaOverrideSection state={state} />
    </div>
  )
}

function EditProfileSelect({ state, qualityProfiles }: EditFormProps) {
  const profileLabel = state.qualityProfileId
    ? (qualityProfiles.find((p) => p.id === state.qualityProfileId)?.name ?? 'Select profile')
    : 'Default (use global)'

  return (
    <div className="space-y-2">
      <Label>Quality Profile</Label>
      <Select
        value={state.qualityProfileId?.toString() ?? ''}
        onValueChange={(value) =>
          state.setQualityProfileId(value ? Number.parseInt(value, 10) : null)
        }
      >
        <SelectTrigger>{profileLabel}</SelectTrigger>
        <SelectContent>
          <SelectItem value="">Default (use global)</SelectItem>
          {qualityProfiles.map((profile) => (
            <SelectItem key={profile.id} value={profile.id.toString()}>
              {profile.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

function QuotaOverrideSection({ state }: { state: ReturnType<typeof useUserEditDialog> }) {
  return (
    <div className="space-y-2">
      <div className="flex items-center space-x-2">
        <Checkbox
          id="quotaOverride"
          checked={state.useQuotaOverride}
          onCheckedChange={(checked) => state.setUseQuotaOverride(checked)}
        />
        <Label htmlFor="quotaOverride">Override quota limits</Label>
      </div>

      {state.useQuotaOverride ? (
        <QuotaLimitsGrid
          moviesLimit={state.moviesLimit}
          onMoviesLimitChange={state.setMoviesLimit}
          seasonsLimit={state.seasonsLimit}
          onSeasonsLimitChange={state.setSeasonsLimit}
          episodesLimit={state.episodesLimit}
          onEpisodesLimitChange={state.setEpisodesLimit}
        />
      ) : null}
    </div>
  )
}

type QuotaLimitsGridProps = {
  moviesLimit: string
  onMoviesLimitChange: (value: string) => void
  seasonsLimit: string
  onSeasonsLimitChange: (value: string) => void
  episodesLimit: string
  onEpisodesLimitChange: (value: string) => void
}

function QuotaLimitsGrid({
  moviesLimit,
  onMoviesLimitChange,
  seasonsLimit,
  onSeasonsLimitChange,
  episodesLimit,
  onEpisodesLimitChange,
}: QuotaLimitsGridProps) {
  return (
    <div className="ml-6 space-y-2 pt-2">
      <div className="grid grid-cols-3 gap-2">
        <div className="space-y-1">
          <Label className="text-xs">Movies</Label>
          <Input
            type="number"
            placeholder="Default"
            value={moviesLimit}
            onChange={(e) => onMoviesLimitChange(e.target.value)}
          />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">Seasons</Label>
          <Input
            type="number"
            placeholder="Default"
            value={seasonsLimit}
            onChange={(e) => onSeasonsLimitChange(e.target.value)}
          />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">Episodes</Label>
          <Input
            type="number"
            placeholder="Default"
            value={episodesLimit}
            onChange={(e) => onEpisodesLimitChange(e.target.value)}
          />
        </div>
      </div>
      <p className="text-muted-foreground text-xs">
        Leave empty to use the global default. Set to 0 for no limit.
      </p>
    </div>
  )
}
