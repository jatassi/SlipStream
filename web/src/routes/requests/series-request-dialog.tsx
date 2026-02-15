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
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import type { SeasonInfo } from '@/types'

type SeriesRequestDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  seriesTitle?: string
  monitorFuture: boolean
  onMonitorFutureChange: (value: boolean) => void
  seasons: SeasonInfo[]
  loadingSeasons: boolean
  selectedSeasons: Set<number>
  onToggleSeason: (seasonNumber: number) => void
  onSelectAll: () => void
  onDeselectAll: () => void
  onSubmit: () => void
  isSubmitting: boolean
}

function SeasonsSelector({ seasons, loadingSeasons, selectedSeasons, onToggleSeason, onSelectAll, onDeselectAll, monitorFuture }: {
  seasons: SeasonInfo[]; loadingSeasons: boolean; selectedSeasons: Set<number>
  onToggleSeason: (n: number) => void; onSelectAll: () => void; onDeselectAll: () => void; monitorFuture: boolean
}) {
  const buttonsDisabled = loadingSeasons || seasons.length === 0
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <Label className="text-sm font-medium">Seasons</Label>
        <div className="flex gap-2">
          <Button variant="ghost" size="sm" onClick={onSelectAll} disabled={buttonsDisabled}>All</Button>
          <Button variant="ghost" size="sm" onClick={onDeselectAll} disabled={buttonsDisabled}>None</Button>
        </div>
      </div>
      <SeasonsList seasons={seasons} loading={loadingSeasons} selectedSeasons={selectedSeasons} onToggle={onToggleSeason} />
      {selectedSeasons.size === 0 && monitorFuture ? (
        <p className="text-muted-foreground text-xs">
          No seasons selected. Series will be added to library and only future episodes will be monitored.
        </p>
      ) : null}
    </div>
  )
}

export function SeriesRequestDialog(props: SeriesRequestDialogProps) {
  const submitDisabled = props.isSubmitting || (!props.monitorFuture && props.selectedSeasons.size === 0)

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Request {props.seriesTitle}</DialogTitle>
          <DialogDescription>Select which seasons to request and whether to monitor future episodes</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <Label htmlFor="monitorFuture" className="text-sm font-medium">Monitor future episodes</Label>
            <Switch id="monitorFuture" checked={props.monitorFuture} onCheckedChange={props.onMonitorFutureChange} />
          </div>
          <SeasonsSelector
            seasons={props.seasons} loadingSeasons={props.loadingSeasons} selectedSeasons={props.selectedSeasons}
            onToggleSeason={props.onToggleSeason} onSelectAll={props.onSelectAll} onDeselectAll={props.onDeselectAll} monitorFuture={props.monitorFuture}
          />
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => props.onOpenChange(false)}>Cancel</Button>
          <Button onClick={props.onSubmit} disabled={submitDisabled}>
            {props.isSubmitting ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
            Submit Request
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

type SeasonsListProps = {
  seasons: SeasonInfo[]
  loading: boolean
  selectedSeasons: Set<number>
  onToggle: (seasonNumber: number) => void
}

function SeasonsList({ seasons, loading, selectedSeasons, onToggle }: SeasonsListProps) {
  if (loading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 3 }, (_, i) => (
          <Skeleton key={i} className="h-8 w-full" />
        ))}
      </div>
    )
  }

  if (seasons.length === 0) {
    return (
      <p className="text-muted-foreground py-2 text-sm">No season information available</p>
    )
  }

  const sorted = seasons
    .filter((s) => s.seasonNumber > 0)
    .toSorted((a, b) => a.seasonNumber - b.seasonNumber)

  return (
    <div className="max-h-48 space-y-1 overflow-y-auto rounded-md border p-2">
      {sorted.map((season) => (
        <div key={season.seasonNumber} className="flex items-center space-x-2 py-1">
          <Checkbox
            id={`season-${season.seasonNumber}`}
            checked={selectedSeasons.has(season.seasonNumber)}
            onCheckedChange={() => onToggle(season.seasonNumber)}
          />
          <Label
            htmlFor={`season-${season.seasonNumber}`}
            className="flex-1 cursor-pointer text-sm"
          >
            Season {season.seasonNumber}
            {season.name && season.name !== `Season ${season.seasonNumber}` ? (
              <span className="text-muted-foreground ml-1">({season.name})</span>
            ) : null}
          </Label>
        </div>
      ))}
    </div>
  )
}
