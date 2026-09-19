import { Group, Row } from '@/components/grouped-list'
import { StatusDot } from '@/components/media/status-dot'
import { MediaSearchMonitorControls } from '@/components/search'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { formatBytes, formatDate, formatRuntime } from '@/lib/formatters'
import type { ExtendedMovieResult, Movie, MovieFile, Slot, SlotStatus } from '@/types'

import type { MovieDetailState } from './use-movie-detail'

function fileName(path: string): string {
  return path.split(/[/\\]/).pop() ?? path
}

function fileDetail(file: MovieFile): string {
  return [file.quality, file.videoCodec, file.audioCodec].filter(Boolean).join(' · ')
}

export function MovieFileGroup({ state, movie }: { state: MovieDetailState; movie: Movie }) {
  return (
    <Group header="File">
      <Row title="Quality Profile" trailing={state.qualityProfileName ?? 'Unknown'} />
      <Row title="Size on Disk" trailing={formatBytes(movie.sizeOnDisk)} />
      <SlotRows state={state} movie={movie} />
      <FileRows state={state} movie={movie} />
    </Group>
  )
}

function SlotRows({ state, movie }: { state: MovieDetailState; movie: Movie }) {
  if (!state.isMultiVersionEnabled) {
    return null
  }
  return (
    <>
      {(state.slotStatus?.slotStatuses ?? []).map((slot) => (
        <SlotRow key={slot.slotId} slot={slot} state={state} movie={movie} />
      ))}
    </>
  )
}

function FileRows({ state, movie }: { state: MovieDetailState; movie: Movie }) {
  if (movie.movieFiles.length === 0) {
    return <Row title="Files" trailing="None" />
  }
  return (
    <>
      {movie.movieFiles.map((file) => (
        <Row
          key={file.id}
          title={<span className="font-mono text-[13px]">{fileName(file.path)}</span>}
          subtitle={fileDetail(file)}
          trailing={
            state.isMultiVersionEnabled ? (
              <SlotSelect file={file} slots={state.enabledSlots} state={state} />
            ) : (
              formatBytes(file.size)
            )
          }
        />
      ))}
    </>
  )
}

function SlotRow({ slot, state, movie }: { slot: SlotStatus; state: MovieDetailState; movie: Movie }) {
  return (
    <Row
      title={slot.slotName}
      subtitle={slot.currentQuality ?? 'Empty'}
      leading={<StatusDot status={slot.status} />}
      trailing={
        <MediaSearchMonitorControls
          mediaType="movie-slot"
          movieId={movie.id}
          slotId={slot.slotId}
          title={`${movie.title} — ${slot.slotName}`}
          theme="movie"
          monitored={slot.monitored}
          onMonitoredChange={(m) => state.handleToggleSlotMonitored(slot.slotId, m)}
          monitorDisabled={state.setSlotMonitoredMutation.isPending}
          qualityProfileId={state.slotQualityProfiles[slot.slotId] ?? movie.qualityProfileId}
          tmdbId={movie.tmdbId}
          imdbId={movie.imdbId}
          year={movie.year}
        />
      }
    />
  )
}

function SlotSelect({ file, slots, state }: { file: MovieFile; slots: Slot[]; state: MovieDetailState }) {
  return (
    <Select
      value={file.slotId?.toString() ?? 'unassigned'}
      onValueChange={(value) => {
        if (value && value !== 'unassigned') {
          void state.handleAssignFileToSlot(file.id, Number.parseInt(value, 10))
        }
      }}
      disabled={state.assignFileMutation.isPending}
    >
      <SelectTrigger className="h-8 w-32" aria-label={`Slot for ${fileName(file.path)}`}>
        {state.getSlotName(file.slotId) ?? <span className="text-muted-foreground">Unassigned</span>}
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="unassigned" disabled>
          Unassigned
        </SelectItem>
        {slots.map((slot) => (
          <SelectItem key={slot.id} value={slot.id.toString()}>
            {slot.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

type DetailEntry = { label: string; value?: string }

function movieDetailEntries(movie: Movie, extended?: ExtendedMovieResult): DetailEntry[] {
  return [
    { label: 'Year', value: movie.year === undefined ? undefined : String(movie.year) },
    { label: 'Runtime', value: movie.runtime === undefined ? undefined : formatRuntime(movie.runtime) },
    { label: 'Studio', value: movie.studio },
    { label: 'Director', value: extended?.credits?.directors?.[0]?.name },
    { label: 'Content Rating', value: movie.contentRating },
    { label: 'Genres', value: extended?.genres?.join(', ') },
    { label: 'Added', value: movie.addedAt ? formatDate(movie.addedAt) : undefined },
    { label: 'Added By', value: movie.addedByUsername },
  ]
}

export function MovieDetailsGroup({ movie, extended }: { movie: Movie; extended?: ExtendedMovieResult }) {
  const entries = movieDetailEntries(movie, extended).filter((entry) => entry.value !== undefined)

  return (
    <Group header="Details">
      {entries.map((entry) => (
        <Row key={entry.label} title={entry.label} trailing={entry.value} />
      ))}
      {movie.path === undefined ? null : (
        <Row
          title="Path"
          trailing={<span className="max-w-44 truncate font-mono text-[12px]">{movie.path}</span>}
        />
      )}
    </Group>
  )
}
