import { useState } from 'react'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/components/ui/dialog'
import { toast } from 'sonner'
import { useUpdateMovie } from '@/hooks'
import type { Movie } from '@/types'

interface MovieEditDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  movie: Movie
}

export function MovieEditDialog({ open, onOpenChange, movie }: MovieEditDialogProps) {
  const [monitored, setMonitored] = useState(movie.monitored)
  const [prevMovie, setPrevMovie] = useState(movie)

  if (movie.id !== prevMovie.id) {
    setPrevMovie(movie)
    setMonitored(movie.monitored)
  }

  const updateMutation = useUpdateMovie()

  const handleSubmit = async () => {
    if (monitored === movie.monitored) {
      onOpenChange(false)
      return
    }

    try {
      await updateMutation.mutateAsync({
        id: movie.id,
        data: { monitored },
      })
      toast.success(monitored ? 'Movie monitored' : 'Movie unmonitored')
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
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label htmlFor="monitored">Monitored</Label>
              <p className="text-sm text-muted-foreground">
                Search for releases and upgrade quality
              </p>
            </div>
            <Switch
              id="monitored"
              checked={monitored}
              onCheckedChange={setMonitored}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={updateMutation.isPending}>
            {updateMutation.isPending && <Loader2 className="size-4 mr-2 animate-spin" />}
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
