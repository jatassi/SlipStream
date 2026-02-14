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
import { useQualityProfiles, useUpdateMovie } from '@/hooks'
import type { Movie } from '@/types'

type MovieEditDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  movie: Movie
}

export function MovieEditDialog({ open, onOpenChange, movie }: MovieEditDialogProps) {
  const [monitored, setMonitored] = useState(movie.monitored)
  const [qualityProfileId, setQualityProfileId] = useState(movie.qualityProfileId)
  const [prevMovie, setPrevMovie] = useState(movie)

  if (movie.id !== prevMovie.id) {
    setPrevMovie(movie)
    setMonitored(movie.monitored)
    setQualityProfileId(movie.qualityProfileId)
  }

  const updateMutation = useUpdateMovie()
  const { data: profiles } = useQualityProfiles()

  const hasChanges = monitored !== movie.monitored || qualityProfileId !== movie.qualityProfileId

  const handleSubmit = async () => {
    if (!hasChanges) {
      onOpenChange(false)
      return
    }

    try {
      await updateMutation.mutateAsync({
        id: movie.id,
        data: { monitored, qualityProfileId },
      })
      toast.success('Movie updated')
      onOpenChange(false)
    } catch {
      toast.error('Failed to update movie')
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Edit Movie</DialogTitle>
          <DialogDescription>{movie.title}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="quality-profile">Quality Profile</Label>
            <Select
              value={qualityProfileId.toString()}
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
                Search for releases and upgrade quality
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
