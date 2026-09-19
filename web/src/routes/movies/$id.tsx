import { ErrorState } from '@/components/data/error-state'
import { usePushBack } from '@/components/layout/use-push-back'
import { MediaEditDialog } from '@/components/media/media-edit-dialog'
import { Screen } from '@/components/screen/screen'
import { useUpdateMovie } from '@/hooks'

import { MovieDetailActions } from './movie-detail-actions'
import { MovieDetailContent } from './movie-detail-content'
import { MovieDetailHero } from './movie-detail-hero'
import { MovieDetailSkeleton } from './movie-detail-skeleton'
import { type MovieDetailState, useMovieDetail } from './use-movie-detail'

const HERO_FADE = 200

export function MovieDetailPage() {
  const state = useMovieDetail()
  const back = usePushBack()

  return (
    <Screen title={state.movie?.title ?? 'Movie'} largeTitle={false} transparentUntil={HERO_FADE} back={back}>
      <MovieDetailBody state={state} />
    </Screen>
  )
}

function MovieDetailBody({ state }: { state: MovieDetailState }) {
  const updateMutation = useUpdateMovie()

  if (state.isLoading) {
    return <MovieDetailSkeleton />
  }
  if (state.isError || !state.movie) {
    return <ErrorState message="Movie not found" onRetry={state.refetch} />
  }

  const { movie } = state

  return (
    <>
      <MovieDetailHero
        movie={movie}
        extendedData={state.extendedData}
        isExtendedDataLoading={state.isExtendedDataLoading}
        qualityProfileName={state.qualityProfileName}
        overviewExpanded={state.overviewExpanded}
        onToggleOverview={state.toggleOverviewExpanded}
      />
      <MovieDetailActions
        movie={movie}
        isRefreshing={state.refreshMutation.isPending}
        onToggleMonitored={state.handleToggleMonitored}
        onRefresh={state.handleRefresh}
        onEdit={() => state.setEditDialogOpen(true)}
        onDelete={state.handleDelete}
      />
      <MovieDetailContent state={state} movie={movie} />
      <MediaEditDialog
        open={state.editDialogOpen}
        onOpenChange={state.setEditDialogOpen}
        item={movie}
        updateMutation={updateMutation}
        mediaLabel="Movie"
        moduleType="movie"
        monitoredDescription="Search for releases and upgrade quality"
      />
    </>
  )
}
