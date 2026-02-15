import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { MovieEditDialog } from '@/components/movies/movie-edit-dialog'

import { MovieDetailActions } from './movie-detail-actions'
import { MovieDetailContent } from './movie-detail-content'
import { MovieDetailHero } from './movie-detail-hero'
import { useMovieDetail } from './use-movie-detail'

export function MovieDetailPage() {
  const state = useMovieDetail()

  if (state.isLoading) {
    return <LoadingState variant="detail" />
  }
  if (state.isError || !state.movie) {
    return <ErrorState message="Movie not found" onRetry={state.refetch} />
  }

  const { movie } = state

  return (
    <div className="-m-6">
      <MovieDetailHero
        movie={movie}
        extendedData={state.extendedData}
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
      <MovieEditDialog
        open={state.editDialogOpen}
        onOpenChange={state.setEditDialogOpen}
        movie={movie}
      />
    </div>
  )
}
