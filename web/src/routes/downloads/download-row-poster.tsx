import { Download, Film, Tv } from 'lucide-react'

import { PosterImage } from '@/components/media/poster-image'
import { cn } from '@/lib/utils'

const ICON_MAP: Record<string, React.ElementType> = {
  movie: Film,
  series: Tv,
}

const THEME_POSTER_CLASSES: Record<string, string> = {
  movie: 'bg-movie-500/20 text-movie-500',
  series: 'bg-tv-500/20 text-tv-500',
}

type DownloadRowPosterProps = {
  mediaType: string
  tmdbId: number | undefined
  tvdbId: number | undefined
  alt: string
}

export function DownloadRowPoster({ mediaType, tmdbId, tvdbId, alt }: DownloadRowPosterProps) {
  if (tmdbId || tvdbId) {
    return (
      <div className="size-10 overflow-hidden rounded">
        <PosterImage
          tmdbId={tmdbId}
          tvdbId={tvdbId}
          alt={alt}
          type={mediaType === 'movie' ? 'movie' : 'series'}
          className="size-full object-cover"
        />
      </div>
    )
  }

  const Icon = ICON_MAP[mediaType] ?? Download
  const themeClass = THEME_POSTER_CLASSES[mediaType] ?? 'bg-muted text-muted-foreground'
  return (
    <div className={cn('flex size-10 items-center justify-center rounded', themeClass)}>
      <Icon className="size-5" />
    </div>
  )
}
