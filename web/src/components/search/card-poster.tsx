import { NetworkLogo } from '@/components/media/network-logo'
import { PosterImage } from '@/components/media/poster-image'
import { Badge } from '@/components/ui/badge'

import { StatusBadge } from './status-badge'

type CardPosterProps = {
  posterUrl?: string
  title: string
  year?: number
  mediaType: 'movie' | 'series'
  network?: string
  networkLogoUrl?: string
  hasActiveDownload: boolean
  isInLibrary: boolean
  hasExistingRequest: boolean
  isAvailable: boolean
  isApproved: boolean
  onClick: () => void
}

export function CardPoster({
  posterUrl,
  title,
  year,
  mediaType,
  network,
  networkLogoUrl,
  hasActiveDownload,
  isInLibrary,
  hasExistingRequest,
  isAvailable,
  isApproved,
  onClick,
}: CardPosterProps) {
  return (
    <button type="button" className="relative aspect-[2/3] w-full cursor-pointer text-left" onClick={onClick}>
      <PosterImage url={posterUrl} alt={title} type={mediaType} className="absolute inset-0" />

      <div className="absolute top-2 left-2 flex flex-col gap-1">
        <StatusBadge
          hasActiveDownload={hasActiveDownload}
          isInLibrary={isInLibrary}
          hasExistingRequest={hasExistingRequest}
          isAvailable={isAvailable}
          isApproved={isApproved}
        />
      </div>

      {mediaType === 'series' && (
        <NetworkLogo
          logoUrl={networkLogoUrl}
          network={network}
          className="absolute top-2 right-2"
        />
      )}

      <div className="absolute inset-0 bg-black/40 opacity-0 transition-opacity group-hover:opacity-100" />
      <HoverOverlay title={title} year={year} network={network} networkLogoUrl={networkLogoUrl} />
    </button>
  )
}

type HoverOverlayProps = {
  title: string
  year?: number
  network?: string
  networkLogoUrl?: string
}

function HoverOverlay({ title, year, network, networkLogoUrl }: HoverOverlayProps) {
  return (
    <div className="absolute inset-x-0 bottom-0 flex flex-col justify-end p-3 opacity-0 transition-opacity group-hover:opacity-100">
      <h3 className="line-clamp-3 font-semibold text-white">{title}</h3>
      <div className="flex items-center gap-2 text-sm text-gray-300">
        <span>{year ?? 'Unknown year'}</span>
        {network && !networkLogoUrl ? (
          <Badge variant="secondary" className="text-xs">
            {network}
          </Badge>
        ) : null}
      </div>
    </div>
  )
}
