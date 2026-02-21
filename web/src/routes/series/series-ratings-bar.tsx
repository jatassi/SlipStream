import { IMDbIcon, MetacriticIcon, RTFreshIcon, RTRottenIcon } from '@/components/media/rating-icons'
import { Skeleton } from '@/components/ui/skeleton'
import type { ExternalRatings } from '@/types'

type SeriesRatingsBarProps = {
  ratings?: ExternalRatings
  isLoading?: boolean
}

export function SeriesRatingsBar({ ratings, isLoading }: SeriesRatingsBarProps) {
  if (isLoading) {
    return (
      <div className="flex items-center gap-4">
        <Skeleton className="h-5 w-14 rounded bg-white/10" />
        <Skeleton className="h-5 w-12 rounded bg-white/10" />
      </div>
    )
  }

  if (!ratings) {
    return null
  }

  const hasAny =
    ratings.rottenTomatoes !== undefined ||
    ratings.imdbRating !== undefined ||
    ratings.metacritic !== undefined

  if (!hasAny) {return null}

  return (
    <div className="flex items-center gap-4 text-sm text-gray-300">
      {ratings.rottenTomatoes !== undefined && (
        <span className="flex items-center gap-1.5">
          {ratings.rottenTomatoes >= 60 ? (
            <RTFreshIcon className="h-5" />
          ) : (
            <RTRottenIcon className="h-5" />
          )}
          <span className="font-medium">{ratings.rottenTomatoes}%</span>
        </span>
      )}
      {ratings.imdbRating !== undefined && (
        <span className="flex items-center gap-1.5">
          <IMDbIcon className="h-4" />
          <span className="font-medium">{ratings.imdbRating.toFixed(1)}</span>
        </span>
      )}
      {ratings.metacritic !== undefined && (
        <span className="flex items-center gap-1.5">
          <MetacriticIcon className="h-5" />
          <span className="font-medium">{ratings.metacritic}</span>
        </span>
      )}
    </div>
  )
}
