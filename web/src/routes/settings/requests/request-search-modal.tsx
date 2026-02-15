import { SearchModal } from '@/components/search/search-modal'

import type { SearchModalState } from './status-config'

type RequestSearchModalProps = {
  searchModal: SearchModalState
  onClose: () => void
}

export function RequestSearchModal({ searchModal, onClose }: RequestSearchModalProps) {
  const isMovie = searchModal.mediaType === 'movie'

  return (
    <SearchModal
      open={searchModal.open}
      onOpenChange={(open) => {
        if (!open) {
          onClose()
        }
      }}
      qualityProfileId={searchModal.qualityProfileId}
      movieId={isMovie ? searchModal.mediaId : undefined}
      movieTitle={isMovie ? searchModal.mediaTitle : undefined}
      tmdbId={searchModal.tmdbId}
      imdbId={searchModal.imdbId}
      year={searchModal.year}
      seriesId={isMovie ? undefined : searchModal.mediaId}
      seriesTitle={isMovie ? undefined : searchModal.mediaTitle}
      tvdbId={searchModal.tvdbId}
      season={searchModal.season}
    />
  )
}
