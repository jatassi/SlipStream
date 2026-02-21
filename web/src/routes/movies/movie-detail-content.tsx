import { SlotStatusCard } from '@/components/slots'
import type { Movie } from '@/types'

import { MovieDetailCredits } from './movie-detail-credits'
import { MovieDetailFiles } from './movie-detail-files'
import type { MovieDetailState } from './use-movie-detail'

type MovieDetailContentProps = {
  state: MovieDetailState
  movie: Movie
}

export function MovieDetailContent({ state, movie }: MovieDetailContentProps) {
  return (
    <div className="space-y-6 p-6">
      {state.isMultiVersionEnabled ? (
        <SlotStatusCard
          status={state.slotStatus}
          isLoading={state.isLoadingSlotStatus}
          movieId={state.movieId}
          movieTitle={movie.title}
          qualityProfileId={movie.qualityProfileId}
          tmdbId={movie.tmdbId}
          imdbId={movie.imdbId}
          year={movie.year}
          slotQualityProfiles={state.slotQualityProfiles}
          onToggleMonitored={state.handleToggleSlotMonitored}
          isUpdating={state.setSlotMonitoredMutation.isPending}
        />
      ) : null}
      <MovieDetailFiles
        files={movie.movieFiles}
        isMultiVersionEnabled={state.isMultiVersionEnabled}
        expandedFileId={state.expandedFileId}
        enabledSlots={state.enabledSlots}
        isAssigning={state.assignFileMutation.isPending}
        onToggleExpandFile={state.toggleExpandedFile}
        onAssignFileToSlot={state.handleAssignFileToSlot}
        getSlotName={state.getSlotName}
      />
      <MovieDetailCredits credits={state.extendedData?.credits} isLoading={state.isExtendedDataLoading} />
    </div>
  )
}
