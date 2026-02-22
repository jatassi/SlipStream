import { useMemo } from 'react'

import { useNavigate } from '@tanstack/react-router'

import { usePortalDownloads, useSeriesSeasons } from '@/hooks'
import { useExtendedMovieMetadata, useExtendedSeriesMetadata } from '@/hooks/use-metadata'
import type {
  ExtendedMovieResult,
  ExtendedSeriesResult,
  PortalDownload,
  Request,
  SeriesSearchResult,
} from '@/types'

import type { MediaInfoModalProps, MediaInfoState } from './media-info-modal-types'

type HookParams = Pick<MediaInfoModalProps, 'open' | 'onOpenChange' | 'media' | 'mediaType' | 'inLibrary' | 'onAction'>

export function useMediaInfoModal(params: HookParams): MediaInfoState {
  const { open, onOpenChange, media, mediaType, inLibrary, onAction } = params
  const navigate = useNavigate()
  const { data: downloads, requests } = usePortalDownloads()

  const { isLoading, extendedData } = useExtendedData(mediaType, open, media.tmdbId)
  const tmdbId = media.tmdbId

  const matchingRequests = useMemo(
    () => findMatchingRequests(requests, tmdbId, mediaType),
    [requests, tmdbId, mediaType],
  )
  const aggregateStatus = useMemo(() => computeAggregateStatus(matchingRequests), [matchingRequests])
  const activeDownload = useMemo(
    () => findActiveDownload(downloads, tmdbId, matchingRequests),
    [downloads, tmdbId, matchingRequests],
  )

  const tvdbId = mediaType === 'series' ? (media as SeriesSearchResult).tvdbId : undefined
  const { data: enrichedSeasons } = useSeriesSeasons(
    mediaType === 'series' && open && inLibrary ? tmdbId : undefined,
    mediaType === 'series' && open && inLibrary ? tvdbId : undefined,
  )

  const handleAdd = makeHandleAdd({ onAction, onOpenChange, mediaType, tmdbId, navigate })
  const derived = deriveExtendedFields(extendedData)
  const libraryStatus = computeLibraryStatus({ aggregateStatus, activeDownload, matchingRequests, inLibrary })

  return {
    extendedData, isLoading, handleAdd, ...derived, ...libraryStatus,
    enrichedSeasons,
  }
}

function useExtendedData(mediaType: 'movie' | 'series', open: boolean, tmdbId: number) {
  const movieQuery = useExtendedMovieMetadata(mediaType === 'movie' && open ? tmdbId : 0)
  const seriesQuery = useExtendedSeriesMetadata(mediaType === 'series' && open ? tmdbId : 0)
  const query = mediaType === 'movie' ? movieQuery : seriesQuery
  return { isLoading: query.isLoading, extendedData: query.data }
}

function deriveExtendedFields(extendedData: ExtendedMovieResult | ExtendedSeriesResult | undefined) {
  return {
    director: extendedData?.credits?.directors?.[0]?.name,
    creators: (extendedData as ExtendedSeriesResult | undefined)?.credits?.creators,
    studio: (extendedData as ExtendedMovieResult | undefined)?.studio,
    seasons: (extendedData as ExtendedSeriesResult | undefined)?.seasons,
    trailerUrl: extendedData?.trailerUrl,
  }
}

type LibraryStatusInput = {
  aggregateStatus: string | null
  activeDownload: PortalDownload | undefined
  matchingRequests: Request[]
  inLibrary: boolean | undefined
}

function computeLibraryStatus(input: LibraryStatusInput) {
  const { aggregateStatus, activeDownload, matchingRequests, inLibrary } = input
  const isAvailable = aggregateStatus === 'available'
  const allHaveMediaId = matchingRequests.length > 0 && matchingRequests.every((r) => r.mediaId !== null)
  const derivedInLibrary = isAvailable && allHaveMediaId
  return {
    isInLibrary: inLibrary === true ? true : derivedInLibrary,
    isPending: aggregateStatus === 'pending',
    isApproved: aggregateStatus === 'approved',
    isAvailable,
    hasActiveDownload: !!activeDownload,
    activeDownload,
  }
}

type HandleAddInput = {
  onAction: (() => void) | undefined
  onOpenChange: (open: boolean) => void
  mediaType: 'movie' | 'series'
  tmdbId: number
  navigate: ReturnType<typeof useNavigate>
}

function makeHandleAdd(input: HandleAddInput) {
  return () => {
    if (input.onAction) {
      input.onAction()
      input.onOpenChange(false)
      return
    }
    const route = input.mediaType === 'movie' ? ('/movies/add' as const) : ('/series/add' as const)
    void input.navigate({ to: route, search: { tmdbId: input.tmdbId } })
    input.onOpenChange(false)
  }
}

function findMatchingRequests(
  requests: Request[] | undefined,
  tmdbId: number,
  mediaType: 'movie' | 'series',
): Request[] {
  if (!requests || !tmdbId) {
    return []
  }
  return requests.filter((r) => {
    if (r.tmdbId !== tmdbId) {
      return false
    }
    if (mediaType === 'movie') {
      return r.mediaType === 'movie'
    }
    return r.mediaType === 'series' || r.mediaType === 'season'
  })
}

function computeAggregateStatus(matchingRequests: Request[]) {
  if (matchingRequests.length === 0) {
    return null
  }
  if (matchingRequests.every((r) => r.status === 'available')) {
    return 'available'
  }
  if (matchingRequests.some((r) => r.status === 'pending')) {
    return 'pending'
  }
  if (matchingRequests.some((r) => r.status === 'approved')) {
    return 'approved'
  }
  return matchingRequests[0].status
}

function findActiveDownload(
  downloads: PortalDownload[] | undefined,
  tmdbId: number,
  matchingRequests: Request[],
) {
  if (!downloads || !tmdbId) {
    return undefined
  }
  const requestIds = new Set(matchingRequests.map((r) => r.id))
  return downloads.find((d) => d.tmdbId === tmdbId || requestIds.has(d.requestId))
}
