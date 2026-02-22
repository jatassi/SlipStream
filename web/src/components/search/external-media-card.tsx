import { useState } from 'react'

import { Plus } from 'lucide-react'

import { cn } from '@/lib/utils'
import type { AvailabilityInfo, MovieSearchResult, SeriesSearchResult } from '@/types'

import { CardActionButton } from './card-action-button'
import { CardPoster } from './card-poster'
import { MediaInfoModal } from './media-info-modal'
import { useExternalMediaCard } from './use-external-media-card'

type ExternalMediaCardProps = {
  media: MovieSearchResult | SeriesSearchResult
  mediaType: 'movie' | 'series'
  inLibrary?: boolean
  availability?: AvailabilityInfo
  requested?: boolean
  currentUserId?: number
  className?: string
  onAction?: () => void
  onViewRequest?: (id: number) => void
  actionLabel?: string
  actionIcon?: React.ReactNode
  disabledLabel?: string
  requestedLabel?: string
  canRequest?: boolean
}

const HOVER_CLASSES = {
  movie: 'hover:border-movie-500/50 hover:glow-movie',
  series: 'hover:border-tv-500/50 hover:glow-tv',
} as const

function getSeriesField<K extends keyof SeriesSearchResult>(
  media: MovieSearchResult | SeriesSearchResult,
  mediaType: 'movie' | 'series',
  field: K,
): SeriesSearchResult[K] | undefined {
  if (mediaType !== 'series') {
    return undefined
  }
  return (media as SeriesSearchResult)[field]
}

export function ExternalMediaCard(props: ExternalMediaCardProps) {
  const { media, mediaType } = props
  const actionLabel = props.actionLabel ?? 'Add to Library'
  const actionIcon = props.actionIcon ?? <Plus className="mr-1 size-3 md:mr-2 md:size-4" />
  const disabledLabel = props.disabledLabel ?? 'Already Added'
  const requestedLabel = props.requestedLabel ?? 'Requested'
  const [infoOpen, setInfoOpen] = useState(false)

  const s = useExternalMediaCard({
    tmdbId: media.tmdbId,
    mediaType,
    inLibrary: props.inLibrary,
    availability: props.availability,
    requested: props.requested,
    currentUserId: props.currentUserId,
  })

  return (
    <div className={cn('group bg-card border-border overflow-hidden rounded-lg border transition-all', HOVER_CLASSES[mediaType], props.className)}>
      <CardPoster
        posterUrl={media.posterUrl} title={media.title} year={media.year} mediaType={mediaType}
        network={getSeriesField(media, mediaType, 'network')}
        networkLogoUrl={getSeriesField(media, mediaType, 'networkLogoUrl')}
        hasActiveDownload={s.hasActiveDownload} isInLibrary={s.isInLibrary}
        canRequest={props.canRequest ?? s.canRequest}
        seasonAvailability={props.availability?.seasonAvailability}
        hasExistingRequest={s.hasExistingRequest} isAvailable={s.isAvailable} isApproved={s.isApproved}
        onClick={() => setInfoOpen(true)}
      />
      <div className="p-2">
        <CardActionButton
          mediaType={mediaType} hasActiveDownload={s.hasActiveDownload}
          activeDownloadMediaId={s.activeDownloadMediaId} isInLibrary={s.isInLibrary}
          canRequest={props.canRequest ?? s.canRequest}
          hasExistingRequest={s.hasExistingRequest} isAvailable={s.isAvailable}
          isApproved={s.isApproved} isOwnRequest={s.isOwnRequest} viewRequestId={s.viewRequestId}
          onAction={props.onAction} onViewRequest={props.onViewRequest}
          actionLabel={actionLabel} actionIcon={actionIcon} requestedLabel={requestedLabel}
        />
      </div>
      <MediaInfoModal
        open={infoOpen} onOpenChange={setInfoOpen} media={media} mediaType={mediaType}
        inLibrary={s.isInLibrary} onAction={(s.canRequest || props.canRequest) ? props.onAction : undefined}
        actionLabel={actionLabel} actionIcon={actionIcon} disabledLabel={disabledLabel}
      />
    </div>
  )
}
