import { useCallback, useState } from 'react'

import { useNavigate } from '@tanstack/react-router'
import { toast } from 'sonner'

import {
  useCreateRequest,
  usePortalLibraryMovies,
  usePortalLibrarySeries,
  useSeriesSeasons,
  useWatchRequest,
} from '@/hooks'
import { usePortalAuthStore } from '@/stores'
import type { PortalMovieSearchResult, PortalSeriesSearchResult } from '@/types'

const REQUEST_SUCCESS = { description: 'Your request has been submitted for review.' }

function onRequestError(error: Error) {
  toast.error('Failed to submit request', { description: error.message })
}

function useSeriesDialog() {
  const [requestDialogOpen, setRequestDialogOpen] = useState(false)
  const [selectedSeries, setSelectedSeries] = useState<PortalSeriesSearchResult | null>(null)
  const [monitorFuture, setMonitorFuture] = useState(false)
  const [selectedSeasons, setSelectedSeasons] = useState<Set<number>>(new Set())

  const { data: seasons = [], isLoading: loadingSeasons } = useSeriesSeasons(
    selectedSeries?.tmdbId ?? selectedSeries?.id,
    selectedSeries?.tvdbId ?? undefined,
  )

  const handleSeriesRequestClick = (item: PortalSeriesSearchResult) => {
    setSelectedSeries(item)
    setMonitorFuture(false)
    setSelectedSeasons(new Set())
    setRequestDialogOpen(true)
  }

  const toggleSeasonSelection = (seasonNumber: number) => {
    setSelectedSeasons((prev) => {
      const next = new Set(prev)
      if (next.has(seasonNumber)) {
        next.delete(seasonNumber)
      } else {
        next.add(seasonNumber)
      }
      return next
    })
  }

  return {
    requestDialogOpen, setRequestDialogOpen,
    selectedSeries, setSelectedSeries,
    monitorFuture, setMonitorFuture,
    selectedSeasons, setSelectedSeasons,
    seasons, loadingSeasons,
    handleSeriesRequestClick, toggleSeasonSelection,
    selectAllSeasons: () =>
      setSelectedSeasons(
        new Set(
          seasons
            .filter((s) => s.seasonNumber > 0 && !s.available && !s.existingRequestId)
            .map((s) => s.seasonNumber),
        ),
      ),
    deselectAllSeasons: () => setSelectedSeasons(new Set()),
  }
}

function buildSeriesRequestPayload(dialog: ReturnType<typeof useSeriesDialog>) {
  if (!dialog.selectedSeries) {
    return undefined
  }
  const seasonsArray = [...dialog.selectedSeasons].toSorted((a, b) => a - b)
  const common = {
    tmdbId: dialog.selectedSeries.tmdbId || dialog.selectedSeries.id,
    tvdbId: dialog.selectedSeries.tvdbId ?? undefined,
    title: dialog.selectedSeries.title,
    year: dialog.selectedSeries.year ?? undefined,
    monitorFuture: dialog.monitorFuture,
    posterUrl: dialog.selectedSeries.posterUrl ?? undefined,
  }
  if (seasonsArray.length === 1) {
    return { ...common, mediaType: 'season' as const, seasonNumber: seasonsArray[0] }
  }
  return {
    ...common,
    mediaType: 'series' as const,
    requestedSeasons: seasonsArray.length > 0 ? seasonsArray : undefined,
  }
}

function useLibraryRequests(dialog: ReturnType<typeof useSeriesDialog>) {
  const navigate = useNavigate()
  const createRequest = useCreateRequest()
  const watchRequest = useWatchRequest()
  const [requestedTmdbIds, setRequestedTmdbIds] = useState<Set<number>>(new Set())

  const isRequested = useCallback(
    (tmdbId: number) => requestedTmdbIds.has(tmdbId),
    [requestedTmdbIds],
  )
  const markRequested = (tmdbId: number) =>
    setRequestedTmdbIds((prev) => new Set(prev).add(tmdbId))

  const handleMovieRequest = (movie: PortalMovieSearchResult) => {
    const tmdbId = movie.tmdbId || movie.id
    createRequest.mutate(
      { mediaType: 'movie', tmdbId, title: movie.title, year: movie.year ?? undefined, posterUrl: movie.posterUrl ?? undefined },
      { onSuccess: () => { markRequested(tmdbId); toast.success('Request submitted', REQUEST_SUCCESS) }, onError: onRequestError },
    )
  }

  const handleSubmitSeriesRequest = () => {
    const payload = buildSeriesRequestPayload(dialog)
    if (!payload) { return }
    createRequest.mutate(payload, {
      onSuccess: () => {
        markRequested(payload.tmdbId)
        dialog.setRequestDialogOpen(false)
        dialog.setSelectedSeries(null)
        dialog.setSelectedSeasons(new Set())
        toast.success('Request submitted', REQUEST_SUCCESS)
      },
      onError: onRequestError,
    })
  }

  const handleWatchRequest = (requestId: number) => {
    watchRequest.mutate(requestId, {
      onSuccess: () => toast.success('Now watching request'),
      onError: (error: Error) => toast.error('Failed to watch request', { description: error.message }),
    })
  }

  return {
    isRequested, handleMovieRequest, handleSubmitSeriesRequest, handleWatchRequest,
    goToRequest: (id: number) => void navigate({ to: '/requests/$id', params: { id: String(id) } }),
    isSubmitting: createRequest.isPending,
  }
}

export function usePortalLibrary() {
  const { user } = usePortalAuthStore()
  const [activeTab, setActiveTab] = useState<'movies' | 'series'>('movies')

  const { data: movies = [], isLoading: loadingMovies } = usePortalLibraryMovies()
  const { data: series = [], isLoading: loadingSeries } = usePortalLibrarySeries()

  const dialog = useSeriesDialog()
  const requests = useLibraryRequests(dialog)

  const partialTmdbIds = new Set(
    series.filter((s) => s.availability?.inLibrary && s.availability.canRequest).map((s) => s.tmdbId),
  )

  return {
    user, activeTab, setActiveTab,
    movies, series, loadingMovies, loadingSeries,
    partialTmdbIds,
    ...dialog, ...requests,
  }
}
