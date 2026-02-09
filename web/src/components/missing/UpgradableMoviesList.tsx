import { Link } from '@tanstack/react-router'
import { ArrowRight, SlidersVertical, TrendingUp } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { PosterImage } from '@/components/media/PosterImage'
import { MediaSearchMonitorControls } from '@/components/search'
import { EmptyState } from '@/components/data/EmptyState'
import { useUpdateMovie } from '@/hooks'
import { toast } from 'sonner'
import { PREDEFINED_QUALITIES } from '@/types/qualityProfile'
import type { UpgradableMovie } from '@/types/missing'
import type { QualityProfile } from '@/types/qualityProfile'

const qualityById = new Map(PREDEFINED_QUALITIES.map((q) => [q.id, q.name]))

interface UpgradableMoviesListProps {
  movies: UpgradableMovie[]
  qualityProfiles: Map<number, QualityProfile>
}

export function UpgradableMoviesList({ movies, qualityProfiles }: UpgradableMoviesListProps) {
  const updateMovieMutation = useUpdateMovie()

  const handleToggleMonitored = async (movie: UpgradableMovie, monitored: boolean) => {
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
        icon={<TrendingUp className="size-8 text-movie-400" />}
        title="No upgradable movies"
        description="All monitored movies meet their quality cutoff"
        className="py-8"
      />
    )
  }

  return (
    <div className="space-y-2">
      {movies.map((movie) => {
        const profile = qualityProfiles.get(movie.qualityProfileId)
        const currentName = qualityById.get(movie.currentQualityId) ?? 'Unknown'
        const cutoffName = profile ? (qualityById.get(profile.cutoff) ?? 'Unknown') : 'Unknown'

        return (
          <div
            key={movie.id}
            className="group flex items-center gap-4 rounded-lg border border-border bg-card px-4 py-3 transition-colors hover:border-movie-500/50"
          >
            <Link
              to="/movies/$id"
              params={{ id: movie.id.toString() }}
              className="hidden sm:block shrink-0"
            >
              <PosterImage
                tmdbId={movie.tmdbId}
                alt={movie.title}
                type="movie"
                className="w-10 h-[60px] rounded-md shadow-sm"
              />
            </Link>

            <div className="min-w-0 flex-1">
              <div className="flex items-baseline gap-2">
                <Link
                  to="/movies/$id"
                  params={{ id: movie.id.toString() }}
                  className="font-medium text-foreground hover:text-movie-400 transition-colors sm:line-clamp-1"
                >
                  {movie.title}
                </Link>
                {movie.year && (
                  <span className="shrink-0 text-xs text-muted-foreground">({movie.year})</span>
                )}
              </div>
              <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground mt-0.5">
                <Badge variant="secondary" className="gap-1 text-[10px] px-1.5 py-0">
                  <span className="text-yellow-500">{currentName}</span>
                  <ArrowRight className="size-2.5 text-muted-foreground" />
                  <span className="text-green-500">{cutoffName}</span>
                </Badge>
                {profile && (
                  <Badge variant="secondary" className="gap-1 text-[10px] px-1.5 py-0">
                    <SlidersVertical className="size-2.5" />
                    {profile.name}
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
        )
      })}
    </div>
  )
}
