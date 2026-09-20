import { ErrorState } from '@/components/data/error-state'
import { usePushBack } from '@/components/layout/use-push-back'
import { MediaDetailHero } from '@/components/media/media-detail-hero'
import { MediaDetailMenu } from '@/components/media/media-detail-menu'
import { MediaDetailSkeleton } from '@/components/media/media-detail-skeleton'
import { MediaEditDialog } from '@/components/media/media-edit-dialog'
import { MediaOverview } from '@/components/media/media-overview'
import { PillRow } from '@/components/media/pill-action'
import { Screen } from '@/components/screen/screen'
import { MediaSearchMonitorControls } from '@/components/search'
import { useUpdateMovie } from '@/hooks'
import { formatRuntime } from '@/lib/formatters'
import type { Movie } from '@/types'

import { MovieDetailCredits } from './movie-detail-credits'
import { MovieDetailsGroup, MovieFileGroup } from './movie-detail-groups'
import { type MovieDetailState, useMovieDetail } from './use-movie-detail'

const HERO_FADE = 200

function heroFacts(movie: Movie): string[] {
  const facts: string[] = []
  if (movie.year !== undefined) {
    facts.push(String(movie.year))
  }
  if (movie.runtime !== undefined && movie.runtime > 0) {
    facts.push(formatRuntime(movie.runtime))
  }
  return facts
}

export function MovieDetailPage() {
  const state = useMovieDetail()
  const back = usePushBack()
  const title = state.movie?.title ?? 'Movie'

  return (
    <Screen
      title={title}
      largeTitle={false}
      transparentUntil={HERO_FADE}
      back={back}
      trailing={
        state.movie === undefined ? undefined : (
          <MediaDetailMenu
            mediaLabel="Movie"
            title={title}
            onEdit={() => state.setEditDialogOpen(true)}
            onRefresh={state.handleRefresh}
            onDelete={state.handleDelete}
          />
        )
      }
    >
      <MovieDetailBody state={state} />
    </Screen>
  )
}

function MovieDetailBody({ state }: { state: MovieDetailState }) {
  if (state.isLoading) {
    return <MediaDetailSkeleton />
  }
  if (state.isError || !state.movie) {
    return <ErrorState message="Movie not found" onRetry={state.refetch} />
  }

  const { movie } = state

  return (
    <div className="mx-auto w-full max-w-3xl">
      <MovieHeader state={state} movie={movie} />
      <MediaOverview text={movie.overview} />
      <MovieFileGroup state={state} movie={movie} />
      <MovieDetailsGroup movie={movie} extended={state.extendedData} />
      <div className="px-screen">
        <MovieDetailCredits credits={state.extendedData?.credits} isLoading={state.isExtendedDataLoading} />
      </div>
      <MovieEditSheet state={state} movie={movie} />
    </div>
  )
}

function MovieHeader({ state, movie }: { state: MovieDetailState; movie: Movie }) {
  return (
    <>
      <MediaDetailHero
        kind="movie"
        title={movie.title}
        status={movie.status}
        facts={heroFacts(movie)}
        genres={state.extendedData?.genres}
        rating={state.extendedData?.ratings?.imdbRating}
        isMetadataLoading={state.isExtendedDataLoading}
        tmdbId={movie.tmdbId}
        version={movie.updatedAt}
      />
      <PillRow className="px-screen pt-5 pb-7">
        <MediaSearchMonitorControls
          mediaType="movie"
          movieId={movie.id}
          title={movie.title}
          theme="movie"
          variant="pill"
          monitored={movie.monitored}
          onMonitoredChange={state.handleToggleMonitored}
          qualityProfileId={movie.qualityProfileId}
          tmdbId={movie.tmdbId}
          imdbId={movie.imdbId}
          year={movie.year}
        />
      </PillRow>
    </>
  )
}

function MovieEditSheet({ state, movie }: { state: MovieDetailState; movie: Movie }) {
  const updateMutation = useUpdateMovie()
  return (
    <MediaEditDialog
      open={state.editDialogOpen}
      onOpenChange={state.setEditDialogOpen}
      item={movie}
      updateMutation={updateMutation}
      mediaLabel="Movie"
      moduleType="movie"
      monitoredDescription="Search for releases and upgrade quality"
    />
  )
}
