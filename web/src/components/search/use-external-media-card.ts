import { useMemo } from 'react'

import { usePortalDownloads } from '@/hooks'
import type { AvailabilityInfo, Request } from '@/types'

type UseExternalMediaCardParams = {
  tmdbId: number
  mediaType: 'movie' | 'series'
  inLibrary?: boolean
  availability?: AvailabilityInfo
  requested?: boolean
  currentUserId?: number
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

function resolveCurrentRequest(
  requests: Request[] | undefined,
  matched: Request[],
  availability?: AvailabilityInfo,
): Request | undefined {
  if (matched.length > 0) {
    return matched[0]
  }
  if (!requests || !availability?.existingRequestId) {
    return undefined
  }
  return requests.find((r) => r.id === availability.existingRequestId)
}

function collectAllRequests(matched: Request[], current: Request | undefined): Request[] {
  if (matched.length > 0) {
    return matched
  }
  if (current) {
    return [current]
  }
  return []
}

function computeAggregateStatus(matched: Request[], current: Request | undefined) {
  const all = collectAllRequests(matched, current)
  if (all.length === 0) {
    return null
  }
  if (all.every((r) => r.status === 'available')) {
    return 'available'
  }
  if (all.some((r) => r.status === 'pending')) {
    return 'pending'
  }
  if (all.some((r) => r.status === 'approved')) {
    return 'approved'
  }
  return all[0].status
}

type DeriveContext = {
  matched: Request[]
  current: Request | undefined
  aggregateStatus: string | null
}

function resolveRequestStatus(ctx: DeriveContext, availability?: AvailabilityInfo) {
  return ctx.aggregateStatus ?? availability?.existingRequestStatus
}

function checkExplicitlyInLibrary(params: UseExternalMediaCardParams): boolean {
  if (params.inLibrary === true) {
    return true
  }
  return params.availability?.inLibrary ?? false
}

function checkInLibraryViaRequests(ctx: DeriveContext, availability?: AvailabilityInfo): boolean {
  const isAvailable = resolveRequestStatus(ctx, availability) === 'available'
  if (!isAvailable) {
    return false
  }
  const allHaveMediaId = ctx.matched.length > 0 && ctx.matched.every((r) => r.mediaId !== null)
  return allHaveMediaId || ctx.current?.mediaId !== null
}

function checkInLibrary(ctx: DeriveContext, params: UseExternalMediaCardParams): boolean {
  return checkExplicitlyInLibrary(params) || checkInLibraryViaRequests(ctx, params.availability)
}

function checkHasExistingRequest(requested: boolean, matched: Request[], availability?: AvailabilityInfo): boolean {
  if (requested || matched.length > 0) {
    return true
  }
  return availability?.existingRequestId !== undefined && availability.existingRequestId !== null
}

function checkIsOwnRequest(requested: boolean, matched: Request[], params: UseExternalMediaCardParams): boolean {
  if (requested || matched.length > 0) {
    return true
  }
  const { availability, currentUserId } = params
  return availability?.existingRequestUserId !== undefined && availability.existingRequestUserId !== null && availability.existingRequestUserId === currentUserId
}

function deriveFlags(ctx: DeriveContext, params: UseExternalMediaCardParams) {
  const requested = params.requested === true
  const requestStatus = resolveRequestStatus(ctx, params.availability)
  const isInLibrary = checkInLibrary(ctx, params)

  return {
    isInLibrary,
    isApproved: requestStatus === 'approved',
    isAvailable: requestStatus === 'available',
    hasExistingRequest: checkHasExistingRequest(requested, ctx.matched, params.availability),
    isOwnRequest: checkIsOwnRequest(requested, ctx.matched, params),
    canRequest: !requested && ctx.matched.length === 0 && !ctx.current && (params.availability?.canRequest ?? !isInLibrary),
  }
}

export function useExternalMediaCard(params: UseExternalMediaCardParams) {
  const { tmdbId, mediaType, availability } = params
  const { data: downloads, requests } = usePortalDownloads()

  const matchingRequests = useMemo(
    () => findMatchingRequests(requests, tmdbId, mediaType),
    [requests, tmdbId, mediaType],
  )

  const currentRequest = useMemo(
    () => resolveCurrentRequest(requests, matchingRequests, availability),
    [requests, matchingRequests, availability],
  )

  const aggregateStatus = useMemo(
    () => computeAggregateStatus(matchingRequests, currentRequest),
    [matchingRequests, currentRequest],
  )

  const ctx: DeriveContext = { matched: matchingRequests, current: currentRequest, aggregateStatus }
  const flags = deriveFlags(ctx, params)

  const activeDownload = useMemo(
    () => findActiveDownload({ downloads, tmdbId, matched: matchingRequests, current: currentRequest, availability, mediaType }),
    [downloads, tmdbId, matchingRequests, currentRequest, availability, mediaType],
  )

  const activeDownloadMediaId =
    mediaType === 'movie' ? activeDownload?.movieId : activeDownload?.seriesId

  return {
    ...flags,
    activeDownload,
    activeDownloadMediaId,
    hasActiveDownload: !!activeDownload,
    viewRequestId: availability?.existingRequestId ?? currentRequest?.id,
  }
}

type FindDownloadParams = {
  downloads: ReturnType<typeof usePortalDownloads>['data']
  tmdbId: number
  matched: Request[]
  current: Request | undefined
  availability: AvailabilityInfo | undefined
  mediaType: 'movie' | 'series'
}

function findActiveDownload({ downloads, tmdbId, matched, current, availability, mediaType }: FindDownloadParams) {
  if (!downloads) {
    return undefined
  }
  const requestIds = new Set(matched.map((r) => r.id))
  if (current) {
    requestIds.add(current.id)
  }

  return downloads.find((d) => {
    if (d.tmdbId !== undefined && d.tmdbId === tmdbId) {
      return true
    }
    if (requestIds.has(d.requestId)) {
      return true
    }
    if (availability?.existingRequestId !== undefined && d.requestId === availability.existingRequestId) {
      return true
    }
    return matchesByMediaId(d, availability, mediaType)
  })
}

function matchesByMediaId(
  d: { movieId?: number; seriesId?: number },
  availability: AvailabilityInfo | undefined,
  mediaType: 'movie' | 'series',
): boolean {
  if (availability?.mediaId === undefined) {
    return false
  }
  if (mediaType === 'movie') {
    return d.movieId !== undefined && d.movieId === availability.mediaId
  }
  return d.seriesId !== undefined && d.seriesId === availability.mediaId
}
