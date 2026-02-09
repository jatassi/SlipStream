import { Link } from '@tanstack/react-router'
import { UserSearch, Calendar } from 'lucide-react'
import { PosterImage } from '@/components/media/PosterImage'
import { MediaSearchMonitorControls } from '@/components/search'
import { EmptyState } from '@/components/data/EmptyState'
import { formatDate } from '@/lib/formatters'
import { useUpdateMovie } from '@/hooks'
import { toast } from 'sonner'
import type { MissingMovie } from '@/types/missing'

interface MissingMoviesListProps {
  movies: MissingMovie[]
}

export function MissingMoviesList({ movies }: MissingMoviesListProps) {
  const updateMovieMutation = useUpdateMovie()

  const handleToggleMonitored = async (movie: MissingMovie, monitored: boolean) => {
    try {
      await updateMovieMutation.mutateAsync({
        id: movie.id,
        data: { monitored },
      })
      toast.success(monitored ? `"${movie.title}" monitored` : `"${movie.title}" unmonitored`)
    } catch {
      toast.error(`Failed to update "${movie.title}"`)
    }
  }

  if (movies.length === 0) {
    return (
      <EmptyState
        icon={<UserSearch className="size-8 text-movie-400" />}
        title="No missing movies"
        description="All monitored movies with available release dates have been downloaded"
        className="py-8"
      />
    )
  }

  return (
    <div className="space-y-2">
      {movies.map((movie) => (
        <div
          key={movie.id}
          className="group flex items-center gap-4 rounded-lg border border-border bg-card px-4 py-3 transition-colors hover:border-movie-500/50"
        >
          <Link
            to="/movies/$id"
            params={{ id: movie.id.toString() }}
            className="shrink-0"
          >
            <PosterImage
              tmdbId={movie.tmdbId}
              alt={movie.title}
              type="movie"
              className="w-10 h-[60px] rounded-md shadow-sm"
            />
          </Link>

          <div className="min-w-0 flex-1">
            <Link
              to="/movies/$id"
              params={{ id: movie.id.toString() }}
              className="font-medium text-foreground hover:text-movie-400 transition-colors line-clamp-1"
            >
              {movie.title}
            </Link>
            <div className="flex items-center gap-3 text-xs text-muted-foreground mt-0.5">
              {movie.year && <span>{movie.year}</span>}
              {movie.releaseDate && (
                <span className="flex items-center gap-1">
                  <Calendar className="size-3" />
                  {formatDate(movie.releaseDate)}
                </span>
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
              monitored={true}
              onMonitoredChange={(m) => handleToggleMonitored(movie, m)}
              monitorDisabled={updateMovieMutation.isPending}
              qualityProfileId={movie.qualityProfileId}
              tmdbId={movie.tmdbId}
              imdbId={movie.imdbId}
              year={movie.year}
            />
          </div>
        </div>
      ))}
    </div>
  )
}
