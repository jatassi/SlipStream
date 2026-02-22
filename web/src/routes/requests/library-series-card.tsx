import { useState } from 'react'

import { NetworkLogo } from '@/components/media/network-logo'
import { PosterImage } from '@/components/media/poster-image'
import { MediaInfoModal } from '@/components/search/media-info-modal'
import { SeasonAvailabilityBadge } from '@/components/search/season-availability-badge'
import { Badge } from '@/components/ui/badge'
import type { PortalSeriesSearchResult, SeasonAvailabilityInfo } from '@/types'

import { convertToSeriesSearchResult } from './search-utils'

type LibrarySeriesCardProps = {
  series: PortalSeriesSearchResult
  currentUserId?: number
  isPartial?: boolean
  onAction?: () => void
  onViewRequest?: (id: number) => void
}

function AvailabilityBadge({
  isPartial,
  seasonAvailability,
}: {
  isPartial?: boolean
  seasonAvailability?: SeasonAvailabilityInfo[]
}) {
  if (isPartial && seasonAvailability) {
    return <SeasonAvailabilityBadge seasonAvailability={seasonAvailability} />
  }
  return (
    <Badge variant="secondary" className="bg-green-600 px-1.5 py-0.5 text-[10px] text-white md:px-2 md:text-xs">
      Available
    </Badge>
  )
}

export function LibrarySeriesCard({ series, isPartial, onAction }: LibrarySeriesCardProps) {
  const [infoOpen, setInfoOpen] = useState(false)
  const media = convertToSeriesSearchResult(series)

  return (
    <>
      <button
        type="button"
        className="group bg-card border-border hover:border-tv-500/50 hover:glow-tv block w-full overflow-hidden rounded-lg border text-left transition-all"
        onClick={() => setInfoOpen(true)}
      >
        <div className="relative aspect-[2/3]">
          <PosterImage url={series.posterUrl} alt={series.title} type="series" className="absolute inset-0" />
          <div className="absolute top-2 left-2 z-10">
            <AvailabilityBadge isPartial={isPartial} seasonAvailability={series.availability?.seasonAvailability} />
          </div>
          <NetworkLogo logoUrl={series.networkLogoUrl} network={series.network} className="absolute top-2 right-2 z-10 max-w-[40%]" />
          <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black via-black/70 to-transparent p-3 pt-8">
            <h3 className="line-clamp-2 font-semibold text-white drop-shadow-[0_2px_4px_rgba(0,0,0,0.8)]">{series.title}</h3>
            <p className="text-sm text-gray-300 drop-shadow-[0_1px_2px_rgba(0,0,0,0.8)]">{series.year ?? 'Unknown year'}</p>
          </div>
        </div>
      </button>
      <MediaInfoModal
        open={infoOpen}
        onOpenChange={setInfoOpen}
        media={media}
        mediaType="series"
        inLibrary
        onAction={isPartial ? onAction : undefined}
        actionLabel="Request"
        disabledLabel="In Library"
      />
    </>
  )
}
