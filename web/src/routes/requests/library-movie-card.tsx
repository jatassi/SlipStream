import { useState } from 'react'

import { PosterImage } from '@/components/media/poster-image'
import { MediaInfoModal } from '@/components/search/media-info-modal'
import type { PortalMovieSearchResult } from '@/types'

import { convertToMovieSearchResult } from './search-utils'

type LibraryMovieCardProps = {
  movie: PortalMovieSearchResult
  currentUserId?: number
  onAction?: () => void
  onViewRequest?: (id: number) => void
}

export function LibraryMovieCard({
  movie,
  onAction,
}: LibraryMovieCardProps) {
  const [infoOpen, setInfoOpen] = useState(false)
  const media = convertToMovieSearchResult(movie)
  const canRequest = movie.availability?.canRequest ?? false

  return (
    <>
      <button
        type="button"
        className="group bg-card border-border hover:border-movie-500/50 hover:glow-movie block w-full overflow-hidden rounded-lg border text-left transition-all"
        onClick={() => setInfoOpen(true)}
      >
        <div className="relative aspect-[2/3]">
          <PosterImage
            url={movie.posterUrl}
            alt={movie.title}
            type="movie"
            className="absolute inset-0"
          />
          <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black via-black/70 to-transparent p-3 pt-8">
            <h3 className="line-clamp-2 font-semibold text-white drop-shadow-[0_2px_4px_rgba(0,0,0,0.8)]">
              {movie.title}
            </h3>
            <p className="text-sm text-gray-300 drop-shadow-[0_1px_2px_rgba(0,0,0,0.8)]">
              {movie.year ?? 'Unknown year'}
            </p>
          </div>
        </div>
      </button>
      <MediaInfoModal
        open={infoOpen}
        onOpenChange={setInfoOpen}
        media={media}
        mediaType="movie"
        inLibrary
        onAction={canRequest ? onAction : undefined}
        actionLabel="Request"
        disabledLabel="In Library"
      />
    </>
  )
}
