import { Library } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import type { SeasonAvailabilityInfo } from '@/types'

import { formatSeasonRange } from './format-season-range'

const BADGE_CLASS = 'px-1.5 py-0.5 text-[10px] text-white md:px-2 md:text-xs'
const ICON_CLASS = 'mr-0.5 size-2.5 md:mr-1 md:size-3'

type SeasonAvailabilityBadgeProps = {
  seasonAvailability: SeasonAvailabilityInfo[]
}

export function SeasonAvailabilityBadge({ seasonAvailability }: SeasonAvailabilityBadgeProps) {
  const availableSeasons = seasonAvailability
    .filter((s) => s.available && s.seasonNumber > 0)
    .map((s) => s.seasonNumber)

  const totalSeasons = seasonAvailability.filter((s) => s.seasonNumber > 0).length

  if (availableSeasons.length === 0) { return null }

  const text = formatSeasonRange(availableSeasons, totalSeasons)

  return (
    <Badge variant="secondary" className={`bg-green-600 ${BADGE_CLASS}`}>
      <Library className={ICON_CLASS} />
      {text}
    </Badge>
  )
}
