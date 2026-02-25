import type React from 'react'
import { useRef, useState } from 'react'

import { toast } from 'sonner'

import { useApproveRequest, useAutoSearchMovie, useAutoSearchSeason } from '@/hooks'
import type { ApproveRequestInput, Request } from '@/types'

import type { SearchModalState } from './status-config'
import { useAddToLibrary } from './use-add-to-library'

export function useRequestApprove() {
  const [searchModal, setSearchModal] = useState<SearchModalState | null>(null)
  const [processingRequest, setProcessingRequest] = useState<number | null>(null)
  const pendingSeasonsRef = useRef<number[]>([])

  const approveMutation = useApproveRequest()
  const autoSearchMovieMutation = useAutoSearchMovie()
  const autoSearchSeasonMutation = useAutoSearchSeason()
  const addToLibrary = useAddToLibrary()

  const approveAndAdd = async (request: Request, action: ApproveRequestInput['action']) => {
    await approveMutation.mutateAsync({ id: request.id, input: { action } })
    return addToLibrary(request)
  }

  const withProcessing = async (requestId: number, fn: () => Promise<void>) => {
    setProcessingRequest(requestId)
    try {
      await fn()
    } finally {
      setProcessingRequest(null)
    }
  }

  const handleApproveOnly = (request: Request) =>
    withProcessing(request.id, async () => {
      await approveAndAdd(request, 'approve_only')
      toast.success('Request approved and added to library')
    }).catch(showApproveError)

  const handleApproveAndManualSearch = (request: Request) =>
    withProcessing(request.id, async () => {
      const libraryMedia = await approveAndAdd(request, 'manual_search')
      openSearchModal({ request, libraryMedia, setSearchModal, pendingSeasonsRef })
      toast.success('Request approved')
    }).catch(showApproveError)

  const handleApproveAndAutoSearch = (request: Request) =>
    withProcessing(request.id, async () => {
      const { mediaId } = await approveAndAdd(request, 'auto_search')
      await runAutoSearch(request, mediaId, { movie: autoSearchMovieMutation, season: autoSearchSeasonMutation })
    }).catch(showApproveError)

  const handleSearchModalClose = () =>
    advanceOrCloseSearchModal(searchModal, setSearchModal)

  return {
    processingRequest,
    searchModal,
    handleSearchModalClose,
    handleApproveOnly,
    handleApproveAndManualSearch,
    handleApproveAndAutoSearch,
  }
}

function advanceOrCloseSearchModal(
  searchModal: SearchModalState | null,
  setSearchModal: (state: SearchModalState | null) => void,
) {
  if (searchModal?.pendingSeasons && searchModal.pendingSeasons.length > 0) {
    setSearchModal({
      ...searchModal,
      season: searchModal.pendingSeasons[0],
      pendingSeasons: searchModal.pendingSeasons.slice(1),
    })
  } else {
    setSearchModal(null)
  }
}

type LibraryMedia = { mediaId: number; qualityProfileId: number; [key: string]: unknown }

type OpenSearchModalArgs = {
  request: Request
  libraryMedia: LibraryMedia
  setSearchModal: (state: SearchModalState) => void
  pendingSeasonsRef: React.RefObject<number[]>
}

function showApproveError(error: unknown) {
  toast.error('Failed to approve request', {
    description: error instanceof Error ? error.message : 'Unknown error',
  })
}

function openSearchModal({ request, libraryMedia, setSearchModal, pendingSeasonsRef }: OpenSearchModalArgs) {
  if (request.mediaType === 'movie' && 'imdbId' in libraryMedia) {
    setSearchModal({
      open: true,
      mediaType: 'movie',
      mediaId: libraryMedia.mediaId,
      mediaTitle: request.title,
      tmdbId: libraryMedia.tmdbId as number | undefined,
      imdbId: libraryMedia.imdbId as string | undefined,
      qualityProfileId: libraryMedia.qualityProfileId,
      year: libraryMedia.year as number | undefined,
    })
    return
  }

  const seasonsToSearch = getSeasonsToSearch(request, [1])
  pendingSeasonsRef.current = seasonsToSearch.slice(1)
  setSearchModal({
    open: true,
    mediaType: 'series',
    mediaId: libraryMedia.mediaId,
    mediaTitle: request.title,
    tvdbId: 'tvdbId' in libraryMedia ? (libraryMedia.tvdbId as number | undefined) : undefined,
    qualityProfileId: libraryMedia.qualityProfileId,
    season: seasonsToSearch[0],
    pendingSeasons: seasonsToSearch.slice(1),
  })
}

function getSeasonsToSearch(request: Request, fallback: number[]): number[] {
  if (request.requestedSeasons.length > 0) {
    return request.requestedSeasons.toSorted((a, b) => a - b)
  }
  return fallback
}

type SearchMutations = {
  movie: ReturnType<typeof useAutoSearchMovie>
  season: ReturnType<typeof useAutoSearchSeason>
}

async function runAutoSearch(request: Request, mediaId: number, mutations: SearchMutations) {
  if (request.mediaType === 'movie') {
    const result = await mutations.movie.mutateAsync(mediaId)
    showAutoSearchMovieToast(result)
    return
  }

  const seasonsToSearch = getSeasonsToSearch(request, [])
  let totalDownloaded = 0
  let totalFound = 0
  for (const seasonNum of seasonsToSearch) {
    try {
      const result = await mutations.season.mutateAsync({ seriesId: mediaId, seasonNumber: seasonNum })
      totalDownloaded += result.downloaded
      totalFound += result.found
    } catch {
      // Continue searching other seasons
    }
  }
  showAutoSearchSeasonToast(totalDownloaded, totalFound)
}

function showAutoSearchMovieToast(result: { downloaded: boolean; found: boolean }) {
  if (result.downloaded) {
    toast.success('Request approved and download started')
  } else if (result.found) {
    toast.success('Request approved, release found but not grabbed')
  } else {
    toast.success('Request approved, no releases found')
  }
}

function showAutoSearchSeasonToast(totalDownloaded: number, totalFound: number) {
  if (totalDownloaded > 0) {
    toast.success(`Request approved, ${totalDownloaded} download(s) started`)
  } else if (totalFound > 0) {
    toast.success('Request approved, releases found but not grabbed')
  } else {
    toast.success('Request approved, no releases found')
  }
}
