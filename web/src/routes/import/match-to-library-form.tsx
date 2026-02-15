import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { useEpisodes, useMovies, useSeries } from '@/hooks'
import { useMultiVersionSettings, useSlots } from '@/hooks/use-slots'
import type { Slot } from '@/types'

function MovieSelect({
  selectedMovieId,
  onMovieChange,
}: {
  selectedMovieId: string
  onMovieChange: (v: string) => void
}) {
  const { data: movies } = useMovies()
  if (!movies) {
    return null
  }

  const selected = movies.find((m) => m.id.toString() === selectedMovieId)
  const label = selected ? `${selected.title} (${selected.year})` : 'Select a movie'

  return (
    <div className="space-y-2">
      <Label htmlFor="movie-select">Movie</Label>
      <Select value={selectedMovieId} onValueChange={(v) => v && onMovieChange(v)}>
        <SelectTrigger id="movie-select">{label}</SelectTrigger>
        <SelectContent>
          {movies.map((movie) => (
            <SelectItem key={movie.id} value={movie.id.toString()}>
              {movie.title} ({movie.year})
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

function formatEpLabel(ep: { seasonNumber: number; episodeNumber: number; title: string }) {
  const s = String(ep.seasonNumber).padStart(2, '0')
  const e = String(ep.episodeNumber).padStart(2, '0')
  return `S${s}E${e} - ${ep.title}`
}

function SeriesSelect({
  selectedSeriesId,
  onSeriesChange,
  onEpisodeChange,
}: {
  selectedSeriesId: string
  onSeriesChange: (v: string) => void
  onEpisodeChange: (v: string) => void
}) {
  const { data: allSeries } = useSeries()
  if (!allSeries) {
    return null
  }

  const label = allSeries.find((s) => s.id.toString() === selectedSeriesId)?.title ?? 'Select a series'

  return (
    <div className="space-y-2">
      <Label htmlFor="series-select">Series</Label>
      <Select
        value={selectedSeriesId}
        onValueChange={(v) => {
          if (v) {
            onSeriesChange(v)
            onEpisodeChange('')
          }
        }}
      >
        <SelectTrigger id="series-select">{label}</SelectTrigger>
        <SelectContent>
          {allSeries.map((s) => (
            <SelectItem key={s.id} value={s.id.toString()}>
              {s.title}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

function EpisodePicker({
  selectedSeriesId,
  selectedEpisodeId,
  onEpisodeChange,
}: {
  selectedSeriesId: string
  selectedEpisodeId: string
  onEpisodeChange: (v: string) => void
}) {
  const seriesIdNum = selectedSeriesId ? Number.parseInt(selectedSeriesId) : 0
  const { data: episodes } = useEpisodes(seriesIdNum)

  const sorted = episodes?.toSorted((a, b) =>
    a.seasonNumber === b.seasonNumber
      ? a.episodeNumber - b.episodeNumber
      : a.seasonNumber - b.seasonNumber,
  )

  if (!selectedSeriesId || !sorted || sorted.length === 0) {
    return null
  }

  const selectedEp = sorted.find((e) => e.id.toString() === selectedEpisodeId)
  const label = selectedEp ? formatEpLabel(selectedEp) : 'Select an episode'

  return (
    <div className="space-y-2">
      <Label htmlFor="episode-select">Episode</Label>
      <Select value={selectedEpisodeId} onValueChange={(v) => v && onEpisodeChange(v)}>
        <SelectTrigger id="episode-select">{label}</SelectTrigger>
        <SelectContent>
          {sorted.map((ep) => (
            <SelectItem key={ep.id} value={ep.id.toString()}>
              {formatEpLabel(ep)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

function SlotSelect({
  selectedSlotId,
  onSlotChange,
}: {
  selectedSlotId: string
  onSlotChange: (v: string) => void
}) {
  const { data: multiVersionSettings } = useMultiVersionSettings()
  const { data: slots } = useSlots()

  const isEnabled = multiVersionSettings?.enabled ?? false
  const enabledSlots = slots?.filter((s: Slot) => s.enabled) ?? []

  if (!isEnabled || enabledSlots.length === 0) {
    return null
  }

  const selected = enabledSlots.find((s: Slot) => s.id.toString() === selectedSlotId)
  const label = selected?.name ?? 'Auto-assign (recommended)'

  return (
    <div className="mt-4 space-y-2 border-t pt-4">
      <h4 className="text-sm font-medium">Version Slot (Multi-Version)</h4>
      <p className="text-muted-foreground text-xs">
        Optionally assign this file to a specific version slot. Leave blank for automatic
        assignment.
      </p>
      <Select value={selectedSlotId} onValueChange={(v) => onSlotChange(v ?? '')}>
        <SelectTrigger>{label}</SelectTrigger>
        <SelectContent>
          <SelectItem value="">Auto-assign (recommended)</SelectItem>
          {enabledSlots.map((slot: Slot) => (
            <SelectItem key={slot.id} value={slot.id.toString()}>
              {slot.name} {slot.qualityProfile ? `(${slot.qualityProfile.name})` : ''}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}

type FormProps = {
  selectedType: 'movie' | 'episode'
  onTypeChange: (v: 'movie' | 'episode') => void
  selectedMovieId: string
  onMovieChange: (v: string) => void
  selectedSeriesId: string
  onSeriesChange: (v: string) => void
  selectedEpisodeId: string
  onEpisodeChange: (v: string) => void
  selectedSlotId: string
  onSlotChange: (v: string) => void
}

function MediaTypeSelector({ value, onChange }: { value: 'movie' | 'episode'; onChange: (v: 'movie' | 'episode') => void }) {
  return (
    <div className="space-y-2">
      <Label htmlFor="media-type">Media Type</Label>
      <Select value={value} onValueChange={(v) => { if (v === 'movie' || v === 'episode') { onChange(v) } }}>
        <SelectTrigger id="media-type">{value === 'movie' ? 'Movie' : 'TV Episode'}</SelectTrigger>
        <SelectContent>
          <SelectItem value="movie">Movie</SelectItem>
          <SelectItem value="episode">TV Episode</SelectItem>
        </SelectContent>
      </Select>
    </div>
  )
}

export function MatchToLibraryForm(props: FormProps) {
  return (
    <div className="space-y-4 border-t pt-4">
      <h4 className="text-sm font-medium">Match to Library</h4>
      <MediaTypeSelector value={props.selectedType} onChange={props.onTypeChange} />
      {props.selectedType === 'movie' ? (
        <MovieSelect selectedMovieId={props.selectedMovieId} onMovieChange={props.onMovieChange} />
      ) : (
        <>
          <SeriesSelect selectedSeriesId={props.selectedSeriesId} onSeriesChange={props.onSeriesChange} onEpisodeChange={props.onEpisodeChange} />
          <EpisodePicker selectedSeriesId={props.selectedSeriesId} selectedEpisodeId={props.selectedEpisodeId} onEpisodeChange={props.onEpisodeChange} />
        </>
      )}
      <SlotSelect selectedSlotId={props.selectedSlotId} onSlotChange={props.onSlotChange} />
    </div>
  )
}
