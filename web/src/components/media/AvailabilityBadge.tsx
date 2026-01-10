import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
import type { Series, Season, Movie } from '@/types'

// Types for availability states
type SeriesAvailabilityState = 'available' | 'partial' | 'airing' | 'unreleased'

interface MovieAvailabilityBadgeProps {
  movie: Movie
  className?: string
}

interface SeriesAvailabilityBadgeProps {
  series: Series
  className?: string
}

interface SeasonAvailabilityBadgeProps {
  season: Season
  className?: string
}

interface EpisodeAvailabilityBadgeProps {
  released: boolean
  className?: string
}

// Movie availability badge - uses stored availabilityStatus
export function MovieAvailabilityBadge({ movie, className }: MovieAvailabilityBadgeProps) {
  const status = movie.availabilityStatus || (movie.released ? 'Available' : 'Unreleased')
  const isAvailable = status === 'Available'

  return (
    <Badge
      variant={isAvailable ? 'default' : 'outline'}
      className={cn(
        isAvailable
          ? 'bg-green-600 hover:bg-green-600 text-white'
          : 'border-amber-500 text-amber-500',
        className
      )}
    >
      {status}
    </Badge>
  )
}

// Episode availability badge - simple aired/unaired
export function EpisodeAvailabilityBadge({ released, className }: EpisodeAvailabilityBadgeProps) {
  return (
    <Badge
      variant={released ? 'default' : 'outline'}
      className={cn(
        released
          ? 'bg-green-600 hover:bg-green-600 text-white'
          : 'border-amber-500 text-amber-500',
        className
      )}
    >
      {released ? 'Aired' : 'Unaired'}
    </Badge>
  )
}

// Season availability badge
export function SeasonAvailabilityBadge({ season, className }: SeasonAvailabilityBadgeProps) {
  return (
    <Badge
      variant={season.released ? 'default' : 'outline'}
      className={cn(
        season.released
          ? 'bg-green-600 hover:bg-green-600 text-white'
          : 'border-amber-500 text-amber-500',
        className
      )}
    >
      {season.released ? 'Complete' : 'Airing'}
    </Badge>
  )
}

// Helper function to get badge styling based on availability status text
function getBadgeStyleForStatus(status: string): {
  state: SeriesAvailabilityState
  badgeClassName: string
} {
  if (status === 'Available') {
    return {
      state: 'available',
      badgeClassName: 'bg-green-600 hover:bg-green-600 text-white',
    }
  }
  if (status === 'Unreleased') {
    return {
      state: 'unreleased',
      badgeClassName: 'border-amber-500 text-amber-500',
    }
  }
  if (status.includes('Airing')) {
    return {
      state: 'airing',
      badgeClassName: 'bg-blue-600 hover:bg-blue-600 text-white',
    }
  }
  // Partial availability (e.g., "Season 1 Available", "Seasons 1-2 Available")
  return {
    state: 'partial',
    badgeClassName: 'bg-teal-600 hover:bg-teal-600 text-white',
  }
}

// Helper function to get series availability info - uses stored availabilityStatus
export function getSeriesAvailabilityInfo(series: Series): {
  state: SeriesAvailabilityState
  label: string
  badgeClassName: string
} {
  // Use stored availabilityStatus if available
  if (series.availabilityStatus) {
    const { state, badgeClassName } = getBadgeStyleForStatus(series.availabilityStatus)
    return {
      state,
      label: series.availabilityStatus,
      badgeClassName,
    }
  }

  // Fallback for older data: calculate from released flag
  if (series.released) {
    return {
      state: 'available',
      label: 'Available',
      badgeClassName: 'bg-green-600 hover:bg-green-600 text-white',
    }
  }

  return {
    state: 'unreleased',
    label: 'Unreleased',
    badgeClassName: 'border-amber-500 text-amber-500',
  }
}

// Series availability badge with granular text
export function SeriesAvailabilityBadge({ series, className }: SeriesAvailabilityBadgeProps) {
  const { label, badgeClassName } = getSeriesAvailabilityInfo(series)

  return (
    <Badge
      variant="default"
      className={cn(badgeClassName, className)}
    >
      {label}
    </Badge>
  )
}
