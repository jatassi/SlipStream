import { useMemo, useState } from 'react'

import type { UseMutationResult } from '@tanstack/react-query'
import { toast } from 'sonner'

import { useGrab, useIndexerMovieSearch, useIndexerTVSearch } from '@/hooks'
import type { GrabRequest, GrabResult, ScoredSearchCriteria, TorrentInfo } from '@/types'

import type { SearchModalProps, SortColumn, SortDirection } from './search-modal-types'
import { buildDialogTitle, buildGrabRequest, compareReleases, resolveMediaType } from './search-modal-utils'

type GrabContext = {
  isMovie: boolean
  seriesId?: number
  season?: number
  episode?: number
  mediaId?: number
  onGrabSuccess?: () => void
}

type GrabParams = {
  release: TorrentInfo
  grabMutation: UseMutationResult<GrabResult, Error, GrabRequest>
  ctx: GrabContext
  setGrabbingGuid: (guid: string | null) => void
}

export function useSearchModal(props: SearchModalProps) {
  const derived = deriveMediaInfo(props)
  const state = useSearchModalState(props.open)
  const criteria = useSearchCriteria({ query: state.query, mediaTitle: derived.mediaTitle, qualityProfileId: props.qualityProfileId, tmdbId: props.tmdbId, imdbId: props.imdbId, tvdbId: props.tvdbId, season: props.season, episode: props.episode, year: props.year })

  const movieSearch = useIndexerMovieSearch(criteria, { enabled: state.searchEnabled && derived.isMovie })
  const tvSearch = useIndexerTVSearch(criteria, { enabled: state.searchEnabled && !derived.isMovie })
  const searchResult = derived.isMovie ? movieSearch : tvSearch

  const grabMutation = useGrab()
  const grabContext: GrabContext = { isMovie: derived.isMovie, seriesId: props.seriesId, season: props.season, episode: props.episode, mediaId: derived.mediaId, onGrabSuccess: props.onGrabSuccess }

  const handleSearch = () => {
    state.setSearchEnabled(true)
    void searchResult.refetch()
  }

  const handleGrab = (release: TorrentInfo) => executeGrab({ release, grabMutation, ctx: grabContext, setGrabbingGuid: state.setGrabbingGuid })

  const rawReleases = useMemo(() => searchResult.data?.releases ?? [], [searchResult.data?.releases])
  const releases = useSortedReleases(rawReleases, state.sortColumn, state.sortDirection)

  return {
    query: state.query,
    setQuery: state.setQuery,
    sortColumn: state.sortColumn,
    sortDirection: state.sortDirection,
    grabbingGuid: state.grabbingGuid,
    isLoading: searchResult.isLoading,
    isError: searchResult.isError,
    error: searchResult.error,
    data: searchResult.data,
    releases,
    errors: searchResult.data?.errors ?? [],
    hasTorrents: rawReleases.length > 0,
    hasSlotInfo: rawReleases.some((r) => r.targetSlotId !== undefined),
    title: buildDialogTitle({ seriesTitle: props.seriesTitle, season: props.season, episode: props.episode, mediaTitle: derived.mediaTitle }),
    handleSearch,
    handleGrab,
    handleSort: state.handleSort,
    refetch: searchResult.refetch,
  }
}

function deriveMediaInfo(props: SearchModalProps) {
  const isMovie = !!props.movieId || !!props.tmdbId
  const mediaTitle = props.movieTitle ?? props.seriesTitle ?? ''
  const mediaId = props.movieId ?? props.seriesId
  return { isMovie, mediaTitle, mediaId }
}

async function executeGrab(params: GrabParams) {
  const { release, grabMutation, ctx, setGrabbingGuid } = params
  setGrabbingGuid(release.guid)
  try {
    const mediaTypeInfo = resolveMediaType({ isMovie: ctx.isMovie, seriesId: ctx.seriesId, season: ctx.season, episode: ctx.episode })
    const request = buildGrabRequest({ release, mediaTypeInfo, mediaId: ctx.mediaId, seriesId: ctx.seriesId, season: ctx.season })
    const result = await grabMutation.mutateAsync(request)
    handleGrabResult(result, release.title, ctx.onGrabSuccess)
  } catch {
    toast.error('Failed to grab release')
  } finally {
    setGrabbingGuid(null)
  }
}

function handleGrabResult(result: GrabResult, title: string, onSuccess?: () => void) {
  if (result.success) {
    toast.success(`Grabbed "${title}"`)
    onSuccess?.()
  } else {
    toast.error(result.error ?? 'Failed to grab release')
  }
}

function useSearchModalState(open: boolean) {
  const [query, setQuery] = useState('')
  const [searchEnabled, setSearchEnabled] = useState(false)
  const [sortColumn, setSortColumn] = useState<SortColumn>('score')
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc')
  const [grabbingGuid, setGrabbingGuid] = useState<string | null>(null)
  const [prevOpen, setPrevOpen] = useState(open)

  if (open !== prevOpen) {
    setPrevOpen(open)
    if (open) {
      setQuery('')
      setSearchEnabled(true)
      setSortColumn('score')
      setSortDirection('desc')
      setGrabbingGuid(null)
    } else {
      setSearchEnabled(false)
    }
  }

  const handleSort = (column: SortColumn) => {
    if (sortColumn === column) {
      setSortDirection((prev) => (prev === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortColumn(column)
      setSortDirection(column === 'title' || column === 'indexer' ? 'asc' : 'desc')
    }
  }

  return { query, setQuery, searchEnabled, setSearchEnabled, sortColumn, sortDirection, grabbingGuid, setGrabbingGuid, handleSort }
}

type SearchCriteriaParams = {
  query: string
  mediaTitle: string
  qualityProfileId: number
  tmdbId?: number
  imdbId?: string
  tvdbId?: number
  season?: number
  episode?: number
  year?: number
}

function useSearchCriteria(params: SearchCriteriaParams): ScoredSearchCriteria {
  const { query, mediaTitle, qualityProfileId, tmdbId, imdbId, tvdbId, season, episode, year } = params
  return useMemo(
    () => ({ query: query || mediaTitle, qualityProfileId, tmdbId, imdbId, tvdbId, season, episode, year, limit: 100 }),
    [query, mediaTitle, qualityProfileId, tmdbId, imdbId, tvdbId, season, episode, year],
  )
}

function useSortedReleases(releases: TorrentInfo[], sortColumn: SortColumn, sortDirection: SortDirection): TorrentInfo[] {
  return useMemo(() => {
    const sorted = [...releases]
    sorted.sort((a, b) => {
      const comparison = compareReleases(a, b, sortColumn)
      return sortDirection === 'asc' ? comparison : -comparison
    })
    return sorted
  }, [releases, sortColumn, sortDirection])
}
