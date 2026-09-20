import { Star } from 'lucide-react'

import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'

import { BackdropImage } from './backdrop-image'
import type { MediaStatus } from './media-status'
import { PosterImage } from './poster-image'
import { StatusPill } from './status-pill'

export type MediaDetailHeroProps = {
  kind: 'movie' | 'series'
  title: string
  status: MediaStatus
  facts: string[]
  genres?: string[]
  rating?: number
  isMetadataLoading?: boolean
  tmdbId?: number
  tvdbId?: number
  version?: string | null
}

const KIND_LABEL: Record<'movie' | 'series', string> = {
  movie: 'Movie',
  series: 'Series',
}

export function MediaDetailHero({
  kind,
  title,
  status,
  facts,
  genres,
  rating,
  isMetadataLoading,
  tmdbId,
  tvdbId,
  version,
}: MediaDetailHeroProps) {
  const artwork = { tmdbId, tvdbId, type: kind, alt: title, version } as const
  const line = [...facts, ...(genres && genres.length > 0 ? [genres.join(', ')] : [])]

  return (
    <div className="relative">
      <BackdropImage {...artwork} className="h-56 w-full md:h-72" />
      <div className="relative -mt-24 flex items-end gap-4 px-screen">
        <PosterImage
          {...artwork}
          className="rounded-card w-28 shrink-0 shadow-[0_10px_30px_rgba(0,0,0,0.5)] aspect-[2/3]"
        />
        <div className="min-w-0 flex-1 pb-1">
          <div
            className={cn(
              'text-caption font-semibold tracking-wide uppercase',
              kind === 'movie' ? 'text-movie-400' : 'text-tv-400',
            )}
          >
            {KIND_LABEL[kind]}
          </div>
          <h1 className="text-heading mt-0.5 font-bold text-balance">{title}</h1>
          <HeroFacts line={line} isMetadataLoading={isMetadataLoading} />
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <StatusPill status={status} />
            <HeroRating rating={rating} isMetadataLoading={isMetadataLoading} />
          </div>
        </div>
      </div>
    </div>
  )
}

function HeroFacts({ line, isMetadataLoading }: { line: string[]; isMetadataLoading?: boolean }) {
  if (line.length === 0 && isMetadataLoading === true) {
    return <Skeleton className="mt-1 h-4 w-40 bg-white/10" />
  }
  return <div className="nums text-footnote text-muted-foreground mt-1">{line.join(' · ')}</div>
}

function HeroRating({ rating, isMetadataLoading }: { rating?: number; isMetadataLoading?: boolean }) {
  if (rating === undefined) {
    return isMetadataLoading === true && <Skeleton className="h-4 w-10 bg-white/10" />
  }
  return (
    <span className="nums text-caption inline-flex items-center gap-1 font-semibold text-amber-400">
      <Star className="size-3 fill-current" />
      {rating.toFixed(1)}
    </span>
  )
}
