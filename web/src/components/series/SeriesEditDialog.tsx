import { useState } from 'react'

import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'

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
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { useQualityProfiles, useUpdateSeries } from '@/hooks'
import type { Series } from '@/types'

type SeriesEditDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  series: Series
}

export function SeriesEditDialog({ open, onOpenChange, series }: SeriesEditDialogProps) {
  const [monitored, setMonitored] = useState(series.monitored)
  const [qualityProfileId, setQualityProfileId] = useState(series.qualityProfileId)
  const [prevSeries, setPrevSeries] = useState(series)

  if (series.id !== prevSeries.id) {
    setPrevSeries(series)
    setMonitored(series.monitored)
    setQualityProfileId(series.qualityProfileId)
  }

  const updateMutation = useUpdateSeries()
  const { data: profiles } = useQualityProfiles()

  const hasChanges = monitored !== series.monitored || qualityProfileId !== series.qualityProfileId

  const handleSubmit = async () => {
    if (!hasChanges) {
      onOpenChange(false)
      return
    }

    try {
      await updateMutation.mutateAsync({
        id: series.id,
        data: { monitored, qualityProfileId },
      })
      toast.success('Series updated')
      onOpenChange(false)
    } catch {
      toast.error('Failed to update series')
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Edit Series</DialogTitle>
          <DialogDescription>{series.title}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="quality-profile">Quality Profile</Label>
            <Select
              value={qualityProfileId?.toString() ?? ''}
              onValueChange={(v) => v && setQualityProfileId(Number.parseInt(v, 10))}
            >
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
              <p className="text-muted-foreground text-sm">
                Search for releases and upgrade quality for all monitored episodes
              </p>
            </div>
            <Switch id="monitored" checked={monitored} onCheckedChange={setMonitored} />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={updateMutation.isPending}>
            {updateMutation.isPending ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
