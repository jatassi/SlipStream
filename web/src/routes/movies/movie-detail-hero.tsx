import { SlidersVertical } from 'lucide-react'

import { BackdropImage } from '@/components/media/backdrop-image'
import { MediaStatusBadge } from '@/components/media/media-status-badge'
import { PosterImage } from '@/components/media/poster-image'
import { StudioLogo } from '@/components/media/studio-logo'
import { TitleTreatment } from '@/components/media/title-treatment'
import { Badge } from '@/components/ui/badge'
import type { ExtendedMovieResult, Movie } from '@/types'

import { MetadataRow } from './movie-detail-metadata'
import { MovieRatings } from './movie-detail-ratings'

type MovieDetailHeroProps = {
  movie: Movie
  extendedData?: ExtendedMovieResult
  isExtendedDataLoading: boolean
  qualityProfileName?: string
  overviewExpanded: boolean
  onToggleOverview: () => void
}

export function MovieDetailHero({
  movie,
  extendedData,
  isExtendedDataLoading,
  qualityProfileName,
  overviewExpanded,
  onToggleOverview,
}: MovieDetailHeroProps) {
  return (
    <div className="relative h-64 md:h-80">
      <BackdropImage
        tmdbId={movie.tmdbId}
        type="movie"
        alt={movie.title}
        version={movie.updatedAt}
        className="absolute inset-0"
      />
      <StudioSection movie={movie} />
      <div className="absolute inset-0 flex items-end p-6">
        <div className="flex max-w-4xl items-end gap-6">
          <div className="hidden shrink-0 md:block">
            <PosterImage
              tmdbId={movie.tmdbId}
              alt={movie.title}
              type="movie"
              version={movie.updatedAt}
              className="h-60 w-40 rounded-lg shadow-lg"
            />
          </div>
          <HeroInfo
            movie={movie}
            extendedData={extendedData}
            isExtendedDataLoading={isExtendedDataLoading}
            qualityProfileName={qualityProfileName}
            overviewExpanded={overviewExpanded}
            onToggleOverview={onToggleOverview}
          />
        </div>
      </div>
    </div>
  )
}

function StudioSection({ movie }: { movie: Movie }) {
  if (!movie.studio) {
    return null
  }
  return (
    <StudioLogo
      tmdbId={movie.tmdbId}
      type="movie"
      alt={movie.studio}
      version={movie.updatedAt}
      className="absolute top-4 right-4 z-10"
      fallback={
        <span className="rounded bg-black/50 px-2.5 py-1 text-xs font-medium text-white/80 backdrop-blur-sm">
          {movie.studio}
        </span>
      }
    />
  )
}

function HeroInfo({
  movie,
  extendedData,
  isExtendedDataLoading,
  qualityProfileName,
  overviewExpanded,
  onToggleOverview,
}: MovieDetailHeroProps) {
  return (
    <div className="flex-1 space-y-2">
      <div className="flex flex-wrap items-center gap-2">
        <MediaStatusBadge status={movie.status} />
        {qualityProfileName ? (
          <Badge variant="secondary" className="gap-1">
            <SlidersVertical className="size-3" />
            {qualityProfileName}
          </Badge>
        ) : null}
      </div>
      <TitleTreatment
        tmdbId={movie.tmdbId}
        type="movie"
        alt={movie.title}
        version={movie.updatedAt}
        fallback={<h1 className="text-3xl font-bold text-white">{movie.title}</h1>}
      />
      <MetadataRow movie={movie} extendedData={extendedData} isExtendedDataLoading={isExtendedDataLoading} />
      <MovieRatings ratings={extendedData?.ratings} isLoading={isExtendedDataLoading} />
      {movie.overview ? (
        <button
          type="button"
          className={`max-w-2xl cursor-pointer text-sm text-gray-300 text-left ${overviewExpanded ? '' : 'line-clamp-2'}`}
          onClick={onToggleOverview}
        >
          {movie.overview}
        </button>
      ) : null}
    </div>
  )
}
