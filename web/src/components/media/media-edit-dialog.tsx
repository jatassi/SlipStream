import { SheetPresenter } from '@/components/presenter'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { LoadingButton } from '@/components/ui/loading-button'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

import { useMediaEditDialog } from './use-media-edit-dialog'

type MediaEditItem = { id: number; title: string; monitored: boolean; qualityProfileId: number }

type MediaEditDialogProps<T extends MediaEditItem> = {
  open: boolean
  onOpenChange: (open: boolean) => void
  item: T
  updateMutation: {
    mutateAsync: (args: { id: number; data: { monitored: boolean; qualityProfileId: number } }) => Promise<unknown>
    isPending: boolean
  }
  mediaLabel: string
  moduleType: string
  monitoredDescription: string
}

export function MediaEditDialog<T extends MediaEditItem>({
  open,
  onOpenChange,
  item,
  updateMutation,
  mediaLabel,
  moduleType,
  monitoredDescription,
}: MediaEditDialogProps<T>) {
  const state = useMediaEditDialog({ item, updateMutation, mediaLabel, moduleType, onOpenChange })

  return (
    <SheetPresenter
      open={open}
      onOpenChange={onOpenChange}
      title={`Edit ${mediaLabel}`}
      description={item.title}
      footer={
        <EditFooter onCancel={state.handleCancel} onSubmit={state.handleSubmit} isPending={state.isPending} />
      }
    >
      <EditForm
        profiles={state.profiles}
        qualityProfileId={state.qualityProfileId}
        onProfileChange={state.handleProfileChange}
        monitored={state.monitored}
        onMonitoredChange={state.setMonitored}
        monitoredDescription={monitoredDescription}
      />
    </SheetPresenter>
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
    <div className="flex justify-end gap-2">
      <Button variant="outline" className="min-h-tap" onClick={onCancel}>
        Cancel
      </Button>
      <LoadingButton className="min-h-tap" loading={isPending} onClick={onSubmit}>
        Save
      </LoadingButton>
    </div>
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
    <div className="space-y-4 py-2">
      <div className="space-y-2">
        <Label htmlFor="quality-profile">Quality Profile</Label>
        <Select value={qualityProfileId.toString()} onValueChange={(v) => v && onProfileChange(v)}>
          <SelectTrigger id="quality-profile" className="min-h-tap w-full">
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

      <div className="min-h-tap flex items-center justify-between gap-4">
        <div className="space-y-0.5">
          <Label htmlFor="monitored">Monitored</Label>
          <p className="text-muted-foreground text-footnote">{monitoredDescription}</p>
        </div>
        <Switch id="monitored" checked={monitored} onCheckedChange={onMonitoredChange} />
      </div>
    </div>
  )
}
