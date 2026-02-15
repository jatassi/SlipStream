import { IMDbIcon, MetacriticIcon, RTFreshIcon, RTRottenIcon } from '@/components/media/rating-icons'
import type { ExternalRatings } from '@/types'

type SeriesRatingsBarProps = {
  ratings: ExternalRatings
}

export function SeriesRatingsBar({ ratings }: SeriesRatingsBarProps) {
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
