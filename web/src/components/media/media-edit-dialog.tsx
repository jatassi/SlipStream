import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { LoadingButton } from '@/components/ui/loading-button'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

import { useMediaEditDialog } from './use-media-edit-dialog'

type MediaEditDialogProps<T extends { id: number; title: string; monitored: boolean; qualityProfileId: number }> = {
  open: boolean
  onOpenChange: (open: boolean) => void
  item: T
  updateMutation: {
    mutateAsync: (args: { id: number; data: { monitored: boolean; qualityProfileId: number } }) => Promise<unknown>
    isPending: boolean
  }
  mediaLabel: string
  monitoredDescription: string
}

export function MediaEditDialog<T extends { id: number; title: string; monitored: boolean; qualityProfileId: number }>({
  open,
  onOpenChange,
  item,
  updateMutation,
  mediaLabel,
  monitoredDescription,
}: MediaEditDialogProps<T>) {
  const state = useMediaEditDialog({ item, updateMutation, mediaLabel, onOpenChange })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Edit {mediaLabel}</DialogTitle>
          <DialogDescription>{item.title}</DialogDescription>
        </DialogHeader>
        <EditForm
          profiles={state.profiles}
          qualityProfileId={state.qualityProfileId}
          onProfileChange={state.handleProfileChange}
          monitored={state.monitored}
          onMonitoredChange={state.setMonitored}
          monitoredDescription={monitoredDescription}
        />
        <EditFooter onCancel={state.handleCancel} onSubmit={state.handleSubmit} isPending={state.isPending} />
      </DialogContent>
    </Dialog>
  )
}

function EditFooter({
  onCancel,
  onSubmit,
  isPending,
}: {
  onCancel: () => void
  onSubmit: () => void
  isPending: boolean
}) {
  return (
    <DialogFooter>
      <Button variant="outline" onClick={onCancel}>
        Cancel
      </Button>
      <LoadingButton loading={isPending} onClick={onSubmit}>
        Save
      </LoadingButton>
    </DialogFooter>
  )
}

function EditForm({
  profiles,
  qualityProfileId,
  onProfileChange,
  monitored,
  onMonitoredChange,
  monitoredDescription,
}: {
  profiles?: { id: number; name: string }[]
  qualityProfileId: number
  onProfileChange: (value: string) => void
  monitored: boolean
  onMonitoredChange: (value: boolean) => void
  monitoredDescription: string
}) {
  return (
    <div className="space-y-4 py-4">
      <div className="space-y-2">
        <Label htmlFor="quality-profile">Quality Profile</Label>
        <Select value={qualityProfileId.toString()} onValueChange={(v) => v && onProfileChange(v)}>
          <SelectTrigger id="quality-profile">
            {profiles?.find((p) => p.id === qualityProfileId)?.name ?? 'Select profile...'}
          </SelectTrigger>
          <SelectContent>
            {profiles?.map((profile) => (
              <SelectItem key={profile.id} value={profile.id.toString()}>
                {profile.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="flex items-center justify-between">
        <div className="space-y-0.5">
          <Label htmlFor="monitored">Monitored</Label>
          <p className="text-muted-foreground text-sm">{monitoredDescription}</p>
        </div>
        <Switch id="monitored" checked={monitored} onCheckedChange={onMonitoredChange} />
      </div>
    </div>
  )
}
