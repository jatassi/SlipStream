import { useState, useMemo } from 'react'
import { Plus, Check, Library, Clock, Download, CheckCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import { NetworkLogo } from '@/components/media/NetworkLogo'
import { PosterImage } from '@/components/media/PosterImage'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { MediaInfoModal } from './MediaInfoModal'
import { DownloadProgressBar } from './DownloadProgressBar'
import { usePortalDownloads } from '@/hooks'
import type { MovieSearchResult, SeriesSearchResult, AvailabilityInfo, Request } from '@/types'

export interface ExternalMediaCardProps {
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
}

export function ExternalMediaCard({
  media,
  mediaType,
  inLibrary,
  availability,
  requested,
  currentUserId,
  className,
  onAction,
  onViewRequest,
  actionLabel = 'Add to Library',
  actionIcon = <Plus className="size-3 md:size-4 mr-1 md:mr-2" />,
  disabledLabel = 'Already Added',
  requestedLabel = 'Requested',
}: ExternalMediaCardProps) {
  const [infoOpen, setInfoOpen] = useState(false)
  const { data: downloads, requests } = usePortalDownloads()

  const tmdbId = media.tmdbId

  // Find ALL requests for this media (by TMDB ID)
  // For series, this includes both 'series' and 'season' type requests
  const matchingRequests = useMemo((): Request[] => {
    if (!requests || !tmdbId) return []
    return requests.filter((r) => {
      if (r.tmdbId !== tmdbId) return false
      // For movies, match 'movie' type
      if (mediaType === 'movie') return r.mediaType === 'movie'
      // For series, match 'series' or 'season' types
      return r.mediaType === 'series' || r.mediaType === 'season'
    })
  }, [requests, tmdbId, mediaType])

  // Also check by availability's existingRequestId if not already found
  const currentRequest = useMemo((): Request | undefined => {
    if (matchingRequests.length > 0) return matchingRequests[0]
    if (!requests || !availability?.existingRequestId) return undefined
    return requests.find((r) => r.id === availability.existingRequestId)
  }, [requests, matchingRequests, availability?.existingRequestId])

  // Determine aggregate status across all requests
  const aggregateStatus = useMemo(() => {
    const allRequests = matchingRequests.length > 0 ? matchingRequests : (currentRequest ? [currentRequest] : [])
    if (allRequests.length === 0) return null

    // If ALL requests are 'available', the item is fully in library
    const allAvailable = allRequests.every((r) => r.status === 'available')
    if (allAvailable) return 'available'

    // If ANY request is still pending, show pending
    if (allRequests.some((r) => r.status === 'pending')) return 'pending'

    // If ANY request is approved (but not available), show approved
    if (allRequests.some((r) => r.status === 'approved')) return 'approved'

    // Fall back to first request's status
    return allRequests[0].status
  }, [matchingRequests, currentRequest])

  const requestStatus = aggregateStatus ?? availability?.existingRequestStatus
  const isApproved = requestStatus === 'approved'
  const isAvailable = requestStatus === 'available'

  // Item is in library if: availability says so, OR all requests are 'available'
  const allRequestsHaveMediaId = matchingRequests.length > 0 && matchingRequests.every((r) => r.mediaId != null)
  const isInLibrary = inLibrary || availability?.inLibrary || (isAvailable && (allRequestsHaveMediaId || currentRequest?.mediaId != null))
  const hasExistingRequest = requested || availability?.existingRequestId != null || matchingRequests.length > 0
  const isOwnRequest = requested || (availability?.existingRequestUserId != null && availability.existingRequestUserId === currentUserId) || matchingRequests.length > 0
  const canRequest = !requested && matchingRequests.length === 0 && !currentRequest && (availability?.canRequest ?? !isInLibrary)

  // Check for actual active download (not just request status)
  // Match by TMDB ID or any of the matching request IDs
  const activeDownload = useMemo(() => {
    if (!downloads) return undefined
    const requestIds = new Set(matchingRequests.map((r) => r.id))
    if (currentRequest) requestIds.add(currentRequest.id)

    return downloads.find((d) => {
      // Match by TMDB ID
      if (tmdbId != null && d.tmdbId != null && d.tmdbId === tmdbId) return true
      // Match by any of our request IDs
      if (requestIds.has(d.requestId)) return true
      // Match by request ID from availability
      if (availability?.existingRequestId != null && d.requestId === availability.existingRequestId) return true
      // Fall back to matching by internal media ID
      if (mediaType === 'movie') return d.movieId != null && availability?.mediaId != null && d.movieId === availability.mediaId
      if (mediaType === 'series') return d.seriesId != null && availability?.mediaId != null && d.seriesId === availability.mediaId
      return false
    })
  }, [downloads, tmdbId, matchingRequests, currentRequest, availability?.existingRequestId, availability?.mediaId, mediaType])

  const hasActiveDownload = !!activeDownload

  const handleAction = (e: React.MouseEvent) => {
    e.stopPropagation()
    onAction?.()
  }

  const handleViewRequest = (e: React.MouseEvent) => {
    e.stopPropagation()
    const requestId = availability?.existingRequestId ?? currentRequest?.id
    if (requestId && onViewRequest) {
      onViewRequest(requestId)
    }
  }

  const title = media.title
  const year = media.year
  const network = mediaType === 'series' ? (media as SeriesSearchResult).network : undefined
  const networkLogoUrl = mediaType === 'series' ? (media as SeriesSearchResult).networkLogoUrl : undefined

  return (
    <div
      className={cn(
        'group rounded-lg overflow-hidden bg-card border border-border transition-all',
        mediaType === 'movie'
          ? 'hover:border-movie-500/50 hover:glow-movie'
          : 'hover:border-tv-500/50 hover:glow-tv',
        className
      )}
    >
      <div
        className="relative aspect-[2/3] cursor-pointer"
        onClick={() => setInfoOpen(true)}
      >
        <PosterImage
          url={media.posterUrl}
          alt={title}
          type={mediaType}
          className="absolute inset-0"
        />

        {/* Status badges - show downloading if active download, otherwise in library */}
        <div className="absolute top-2 left-2 flex flex-col gap-1">
          {hasActiveDownload ? (
            <Badge variant="secondary" className="bg-purple-600 text-white text-[10px] md:text-xs px-1.5 md:px-2 py-0.5">
              <Download className="size-2.5 md:size-3 mr-0.5 md:mr-1" />
              Downloading
            </Badge>
          ) : isInLibrary ? (
            <Badge variant="secondary" className="bg-green-600 text-white text-[10px] md:text-xs px-1.5 md:px-2 py-0.5">
              <Library className="size-2.5 md:size-3 mr-0.5 md:mr-1" />
              In Library
            </Badge>
          ) : hasExistingRequest && (
            isAvailable ? (
              <Badge variant="secondary" className="bg-green-600 text-white text-[10px] md:text-xs px-1.5 md:px-2 py-0.5">
                <CheckCircle className="size-2.5 md:size-3 mr-0.5 md:mr-1" />
                Available
              </Badge>
            ) : isApproved ? (
              <Badge variant="secondary" className="bg-blue-600 text-white text-[10px] md:text-xs px-1.5 md:px-2 py-0.5">
                <Check className="size-2.5 md:size-3 mr-0.5 md:mr-1" />
                Approved
              </Badge>
            ) : (
              <Badge variant="secondary" className="bg-yellow-600 text-white text-[10px] md:text-xs px-1.5 md:px-2 py-0.5">
                <Clock className="size-2.5 md:size-3 mr-0.5 md:mr-1" />
                Requested
              </Badge>
            )
          )}
        </div>

        {/* Network logo */}
        {mediaType === 'series' && (
          <NetworkLogo
            logoUrl={networkLogoUrl}
            network={network}
            className="absolute top-2 right-2"
          />
        )}

        {/* Hover overlay */}
        <div className="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity" />
        <div className="absolute inset-x-0 bottom-0 p-3 flex flex-col justify-end opacity-0 group-hover:opacity-100 transition-opacity">
          <h3 className="font-semibold text-white line-clamp-3">{title}</h3>
          <div className="flex items-center gap-2 text-sm text-gray-300">
            <span>{year || 'Unknown year'}</span>
            {network && !networkLogoUrl && (
              <Badge variant="secondary" className="text-xs">
                {network}
              </Badge>
            )}
          </div>
        </div>
      </div>

      <div className="p-2">
        {/* Show downloading if active download, otherwise in library */}
        {hasActiveDownload && activeDownload ? (
          <DownloadProgressBar
            mediaId={mediaType === 'movie' ? activeDownload.movieId : activeDownload.seriesId}
            mediaType={mediaType}
          />
        ) : isInLibrary ? (
          <Button variant="secondary" size="sm" className="w-full text-xs md:text-sm" disabled>
            <Check className="size-3 md:size-4 mr-1 md:mr-2" />
            In Library
          </Button>
        ) : hasExistingRequest ? (
          isAvailable ? (
            <Button variant="secondary" size="sm" className="w-full text-xs md:text-sm" disabled>
              <CheckCircle className="size-3 md:size-4 mr-1 md:mr-2" />
              Available
            </Button>
          ) : isApproved ? (
            <Button variant="secondary" size="sm" className="w-full text-xs md:text-sm" disabled>
              <Check className="size-3 md:size-4 mr-1 md:mr-2" />
              Approved
            </Button>
          ) : isOwnRequest ? (
            <Button variant="secondary" size="sm" className="w-full text-xs md:text-sm" disabled>
              <Clock className="size-3 md:size-4 mr-1 md:mr-2" />
              {requestedLabel}
            </Button>
          ) : (availability?.existingRequestId ?? currentRequest?.id) && onViewRequest ? (
            <Button variant="secondary" size="sm" className="w-full text-xs md:text-sm" onClick={handleViewRequest}>
              View Request
            </Button>
          ) : (
            <Button variant="secondary" size="sm" className="w-full text-xs md:text-sm" disabled>
              <Clock className="size-3 md:size-4 mr-1 md:mr-2" />
              {requestedLabel}
            </Button>
          )
        ) : canRequest && onAction ? (
          <Button variant="default" size="sm" className="w-full text-xs md:text-sm" onClick={handleAction}>
            {actionIcon}
            {actionLabel}
          </Button>
        ) : (
          <Button variant="default" size="sm" className="w-full text-xs md:text-sm" onClick={handleAction}>
            {actionIcon}
            {actionLabel}
          </Button>
        )}
      </div>

      <MediaInfoModal
        open={infoOpen}
        onOpenChange={setInfoOpen}
        media={media}
        mediaType={mediaType}
        inLibrary={isInLibrary}
        onAction={canRequest && !isInLibrary ? onAction : undefined}
        actionLabel={actionLabel}
        actionIcon={actionIcon}
        disabledLabel={disabledLabel}
      />
    </div>
  )
}
