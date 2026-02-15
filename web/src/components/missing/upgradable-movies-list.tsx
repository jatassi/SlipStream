import { Link } from '@tanstack/react-router'
import { ArrowRight, SlidersVertical, TrendingUp } from 'lucide-react'
import { toast } from 'sonner'

import { EmptyState } from '@/components/data/empty-state'
import { PosterImage } from '@/components/media/poster-image'
import { MediaSearchMonitorControls } from '@/components/search'
import { Badge } from '@/components/ui/badge'
import { useUpdateMovie } from '@/hooks'
import type { UpgradableMovie } from '@/types/missing'
import type { QualityProfile } from '@/types/quality-profile'
import { PREDEFINED_QUALITIES } from '@/types/quality-profile'

const qualityById = new Map(PREDEFINED_QUALITIES.map((q) => [q.id, q.name]))

type UpgradableMoviesListProps = {
  movies: UpgradableMovie[]
  qualityProfiles: Map<number, QualityProfile>
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

function QualityBadges({
  currentName,
  cutoffName,
  profile,
}: {
  currentName: string
  cutoffName: string
  profile?: QualityProfile
}) {
  return (
    <div className="text-muted-foreground mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
      <Badge variant="secondary" className="gap-1 px-1.5 py-0 text-[10px]">
        <span className="text-yellow-500">{currentName}</span>
        <ArrowRight className="text-muted-foreground size-2.5" />
        <span className="text-green-500">{cutoffName}</span>
      </Badge>
      {profile ? (
        <Badge variant="secondary" className="gap-1 px-1.5 py-0 text-[10px]">
          <SlidersVertical className="size-2.5" />
          {profile.name}
        </Badge>
      ) : null}
    </div>
  )
}

function UpgradableMovieRow({
  movie,
  qualityProfiles,
  onToggleMonitored,
  isUpdating,
}: {
  movie: UpgradableMovie
  qualityProfiles: Map<number, QualityProfile>
  onToggleMonitored: (movie: UpgradableMovie, monitored: boolean) => void
  isUpdating: boolean
}) {
  const profile = qualityProfiles.get(movie.qualityProfileId)
  const currentName = qualityById.get(movie.currentQualityId) ?? 'Unknown'
  const cutoffName = profile ? (qualityById.get(profile.cutoff) ?? 'Unknown') : 'Unknown'

  return (
    <div className="group border-border bg-card hover:border-movie-500/50 flex items-center gap-4 rounded-lg border px-4 py-3 transition-colors">
      <MovieRowPoster movieId={movie.id} tmdbId={movie.tmdbId} title={movie.title} />
      <div className="min-w-0 flex-1">
        <MovieRowTitle movieId={movie.id} title={movie.title} year={movie.year} />
        <QualityBadges currentName={currentName} cutoffName={cutoffName} profile={profile} />
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
  movie: UpgradableMovie,
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

export function UpgradableMoviesList({ movies, qualityProfiles }: UpgradableMoviesListProps) {
  const updateMovieMutation = useUpdateMovie()

  if (movies.length === 0) {
    return (
      <EmptyState
        icon={<TrendingUp className="text-movie-400 size-8" />}
        title="No upgradable movies"
        description="All monitored movies meet their quality cutoff"
        className="py-8"
      />
    )
  }

  return (
    <div className="space-y-2">
      {movies.map((movie) => (
        <UpgradableMovieRow
          key={movie.id}
          movie={movie}
          qualityProfiles={qualityProfiles}
          onToggleMonitored={(m, monitored) =>
            handleToggleMonitored(m, monitored, updateMovieMutation.mutateAsync)
          }
          isUpdating={updateMovieMutation.isPending}
        />
      ))}
    </div>
  )
}
