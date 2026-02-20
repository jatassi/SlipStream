import { ErrorState } from '@/components/data/error-state'
import { MediaEditDialog } from '@/components/media/media-edit-dialog'
import { useUpdateMovie } from '@/hooks'

import { MovieDetailActions } from './movie-detail-actions'
import { MovieDetailContent } from './movie-detail-content'
import { MovieDetailHero } from './movie-detail-hero'
import { MovieDetailSkeleton } from './movie-detail-skeleton'
import { useMovieDetail } from './use-movie-detail'

export function MovieDetailPage() {
  const state = useMovieDetail()
  const updateMutation = useUpdateMovie()

  if (state.isLoading) {
    return <MovieDetailSkeleton />
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
      <MediaEditDialog
        open={state.editDialogOpen}
        onOpenChange={state.setEditDialogOpen}
        item={movie}
        updateMutation={updateMutation}
        mediaLabel="Movie"
        monitoredDescription="Search for releases and upgrade quality"
      />
    </div>
  )
}
