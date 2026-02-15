import { IMDbIcon, MetacriticIcon, RTFreshIcon, RTRottenIcon } from '@/components/media/rating-icons'
import type { ExternalRatings } from '@/types'

function rtIcon(score: number) {
  return score >= 60 ? <RTFreshIcon className="h-5" /> : <RTRottenIcon className="h-5" />
}

export function RatingsDisplay({ ratings }: { ratings: ExternalRatings }) {
  const hasRatings =
    ratings.rottenTomatoes ?? ratings.rottenAudience ?? ratings.imdbRating ?? ratings.metacritic

  if (!hasRatings && !ratings.awards) {
    return null
  }

  return (
    <div className="space-y-2">
      {hasRatings ? <RatingsBadges ratings={ratings} /> : null}
      {ratings.awards ? (
        <p className="text-muted-foreground text-sm">
          <span className="text-foreground font-medium">Awards:</span> {ratings.awards}
        </p>
      ) : null}
    </div>
  )
}

function RatingsBadges({ ratings }: { ratings: ExternalRatings }) {
  return (
    <div className="flex flex-wrap items-center gap-4">
      {ratings.rottenTomatoes !== undefined && (
        <RatingItem icon={rtIcon(ratings.rottenTomatoes)} value={`${ratings.rottenTomatoes}%`} label="Critics" />
      )}
      {ratings.rottenAudience !== undefined && (
        <RatingItem icon={rtIcon(ratings.rottenAudience)} value={`${ratings.rottenAudience}%`} label="Audience" />
      )}
      {ratings.imdbRating !== undefined && (
        <ImdbRating rating={ratings.imdbRating} votes={ratings.imdbVotes} />
      )}
      {ratings.metacritic !== undefined && (
        <RatingItem icon={<MetacriticIcon className="h-5" />} value={String(ratings.metacritic)} />
      )}
    </div>
  )
}

function ImdbRating({ rating, votes }: { rating: number; votes?: number }) {
  return (
    <div className="flex items-center gap-1.5">
      <IMDbIcon className="h-4" />
      <span className="text-sm font-medium">{rating.toFixed(1)}</span>
      {votes !== undefined && (
        <span className="text-muted-foreground text-xs">({votes.toLocaleString()} votes)</span>
      )}
    </div>
  )
}

function RatingItem({ icon, value, label }: { icon: React.ReactNode; value: string; label?: string }) {
  return (
    <div className="flex items-center gap-1.5">
      {icon}
      <span className="text-sm font-medium">{value}</span>
      {label ? <span className="text-muted-foreground text-xs">{label}</span> : null}
    </div>
  )
}
