import { Link } from '@tanstack/react-router'
import { SlidersVertical, UserSearch } from 'lucide-react'
import { toast } from 'sonner'

import { EmptyState } from '@/components/data/empty-state'
import { PosterImage } from '@/components/media/poster-image'
import { MediaSearchMonitorControls } from '@/components/search'
import { Badge } from '@/components/ui/badge'
import { useUpdateMovie } from '@/hooks'
import type { MissingMovie } from '@/types/missing'

type MissingMoviesListProps = {
  movies: MissingMovie[]
  qualityProfileNames: Map<number, string>
}

function MovieRowPoster({ movieId, tmdbId, title }: { movieId: number; tmdbId?: number; title: string }) {
  return (
    <Link to="/movies/$id" params={{ id: movieId.toString() }} className="hidden shrink-0 sm:block">
      <PosterImage
        tmdbId={tmdbId}
        alt={title}
        type="movie"
        className="h-[60px] w-10 rounded-md shadow-sm"
      />
    </Link>
  )
}

function MovieRowTitle({ movieId, title, year }: { movieId: number; title: string; year?: number }) {
  return (
    <div className="flex items-baseline gap-2">
      <Link
        to="/movies/$id"
        params={{ id: movieId.toString() }}
        className="text-foreground hover:text-movie-400 font-medium transition-colors sm:line-clamp-1"
      >
        {title}
      </Link>
      {year ? <span className="text-muted-foreground shrink-0 text-xs">({year})</span> : null}
    </div>
  )
}

function MissingMovieRow({
  movie,
  qualityProfileNames,
  onToggleMonitored,
  isUpdating,
}: {
  movie: MissingMovie
  qualityProfileNames: Map<number, string>
  onToggleMonitored: (movie: MissingMovie, monitored: boolean) => void
  isUpdating: boolean
}) {
  return (
    <div className="group border-border bg-card hover:border-movie-500/50 flex items-center gap-4 rounded-lg border px-4 py-3 transition-colors">
      <MovieRowPoster movieId={movie.id} tmdbId={movie.tmdbId} title={movie.title} />
      <div className="min-w-0 flex-1">
        <MovieRowTitle movieId={movie.id} title={movie.title} year={movie.year} />
        <div className="text-muted-foreground mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
          {qualityProfileNames.get(movie.qualityProfileId) && (
            <Badge variant="secondary" className="gap-1 px-1.5 py-0 text-[10px]">
              <SlidersVertical className="size-2.5" />
              {qualityProfileNames.get(movie.qualityProfileId)}
            </Badge>
          )}
        </div>
      </div>
      <div className="ml-auto shrink-0">
        <MediaSearchMonitorControls
          mediaType="movie"
          movieId={movie.id}
          title={movie.title}
          theme="movie"
          size="sm"
          monitored
          onMonitoredChange={(m) => onToggleMonitored(movie, m)}
          monitorDisabled={isUpdating}
          qualityProfileId={movie.qualityProfileId}
          tmdbId={movie.tmdbId}
          imdbId={movie.imdbId}
          year={movie.year}
        />
      </div>
    </div>
  )
}

async function handleToggleMonitored(
  movie: MissingMovie,
  monitored: boolean,
  updateFn: ReturnType<typeof useUpdateMovie>['mutateAsync'],
) {
  try {
    await updateFn({ id: movie.id, data: { monitored } })
    toast.success(monitored ? `"${movie.title}" monitored` : `"${movie.title}" unmonitored`)
  } catch {
    toast.error(`Failed to update "${movie.title}"`)
  }
}

export function MissingMoviesList({ movies, qualityProfileNames }: MissingMoviesListProps) {
  const updateMovieMutation = useUpdateMovie()

  if (movies.length === 0) {
    return (
      <EmptyState
        icon={<UserSearch className="text-movie-400 size-8" />}
        title="No missing movies"
        description="All monitored movies with available release dates have been downloaded"
        className="py-8"
      />
    )
  }

  return (
    <div className="space-y-2">
      {movies.map((movie) => (
        <MissingMovieRow
          key={movie.id}
          movie={movie}
          qualityProfileNames={qualityProfileNames}
          onToggleMonitored={(m, monitored) =>
            handleToggleMonitored(m, monitored, updateMovieMutation.mutateAsync)
          }
          isUpdating={updateMovieMutation.isPending}
        />
      ))}
    </div>
  )
}
