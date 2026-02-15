import { Download, Film, Tv } from 'lucide-react'

import { PosterImage } from '@/components/media/poster-image'
import { cn } from '@/lib/utils'

const ICON_MAP = {
  movie: Film,
  series: Tv,
  unknown: Download,
} as const

type DownloadRowPosterProps = {
  mediaType: 'movie' | 'series' | 'unknown'
  tmdbId: number | undefined
  tvdbId: number | undefined
  alt: string
}

export function DownloadRowPoster({ mediaType, tmdbId, tvdbId, alt }: DownloadRowPosterProps) {
  const isMovie = mediaType === 'movie'
  const isSeries = mediaType === 'series'

  if (tmdbId || tvdbId) {
    return (
      <div className="size-10 overflow-hidden rounded">
        <PosterImage
          tmdbId={tmdbId}
          tvdbId={tvdbId}
          alt={alt}
          type={isMovie ? 'movie' : 'series'}
          className="size-full object-cover"
        />
      </div>
    )
  }

  const Icon = ICON_MAP[mediaType]
  return (
    <div
      className={cn(
        'flex size-10 items-center justify-center rounded',
        isMovie && 'bg-movie-500/20 text-movie-500',
        isSeries && 'bg-tv-500/20 text-tv-500',
        !isMovie && !isSeries && 'bg-muted text-muted-foreground',
      )}
    >
      <Icon className="size-5" />
    </div>
  )
}
