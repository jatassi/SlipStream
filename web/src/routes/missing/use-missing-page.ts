import { useEffect, useState } from 'react'

import { toast } from 'sonner'

import {
  useGlobalLoading,
  useMissingMovies,
  useMissingSeries,
  useQualityProfiles,
  useSearchAllMissing,
  useSearchAllMissingMovies,
  useSearchAllMissingSeries,
  useSearchAllUpgradable,
  useSearchAllUpgradableMovies,
  useSearchAllUpgradableSeries,
  useUpgradableMovies,
  useUpgradableSeries,
} from '@/hooks'
import { useAutoSearchStore } from '@/stores'
import type { QualityProfile } from '@/types/quality-profile'

export type ViewMode = 'missing' | 'upgradable'
export type MediaFilter = 'all' | 'movies' | 'series'

async function executeSearch(searchFn: () => Promise<unknown>) {
  try {
    await searchFn()
  } catch (error) {
    if (error instanceof Error && error.message.includes('409')) {
      toast.warning('A search task is already running')
    } else {
      toast.error('Failed to start search')
    }
  }
}

function showTaskResultToast(result: {
  downloaded: number
  found: number
  failed: number
  totalSearched: number
}) {
  const { downloaded, found, failed, totalSearched } = result
  if (downloaded > 0) {
    toast.success(`Downloaded ${downloaded} release${downloaded === 1 ? '' : 's'}`, {
      description: `Searched ${totalSearched} items, found ${found}`,
    })
  } else if (found > 0) {
    toast.info(`Found ${found} releases but none downloaded`, {
      description: `Searched ${totalSearched} items`,
    })
  } else if (failed > 0) {
    toast.error(`Search failed for ${failed} items`, {
      description: `Searched ${totalSearched} items`,
    })
  } else {
    toast.warning('No releases found', {
      description: `Searched ${totalSearched} items`,
    })
  }
}

function getSearchCount(filter: MediaFilter, movieCount: number, episodeCount: number) {
  if (filter === 'movies') {
    return movieCount
  }
  if (filter === 'series') {
    return episodeCount
  }
  return movieCount + episodeCount
}

function getSearchButtonStyle(filter: MediaFilter, movieCount: number, episodeCount: number) {
  const hasMovies = filter === 'movies' || (filter === 'all' && movieCount > 0)
  const hasSeries = filter === 'series' || (filter === 'all' && episodeCount > 0)
  if (hasMovies && hasSeries) {
    return 'glow-media-sm'
  }
  if (hasMovies) {
    return 'bg-movie-500 hover:bg-movie-600 glow-movie-sm'
  }
  if (hasSeries) {
    return 'bg-tv-500 hover:bg-tv-600 glow-tv-sm'
  }
  return ''
}

function useMissingQueries() {
  const missingMovies = useMissingMovies()
  const missingSeries = useMissingSeries()
  const upgradableMovies = useUpgradableMovies()
  const upgradableSeries = useUpgradableSeries()
  return { missingMovies, missingSeries, upgradableMovies, upgradableSeries }
}

function useSearchMutations() {
  return {
    missing: {
      movies: useSearchAllMissingMovies(),
      series: useSearchAllMissingSeries(),
      all: useSearchAllMissing(),
    },
    upgradable: {
      movies: useSearchAllUpgradableMovies(),
      series: useSearchAllUpgradableSeries(),
      all: useSearchAllUpgradable(),
    },
  }
}

function useTaskResultNotifier() {
  const { task, clearResult } = useAutoSearchStore()
  useEffect(() => {
    if (task.result) {
      showTaskResultToast(task.result)
      clearResult()
    }
  }, [task.result, clearResult])
  return task
}

type Queries = ReturnType<typeof useMissingQueries>

function deriveCounts(queries: Queries) {
  return {
    missingMovieCount: queries.missingMovies.data?.length ?? 0,
    missingEpisodeCount:
      queries.missingSeries.data?.reduce((acc, s) => acc + s.missingCount, 0) ?? 0,
    upgradableMovieCount: queries.upgradableMovies.data?.length ?? 0,
    upgradableEpisodeCount:
      queries.upgradableSeries.data?.reduce((acc, s) => acc + s.upgradableCount, 0) ?? 0,
  }
}

function deriveViewState(queries: Queries, isMissingView: boolean, globalLoading: boolean) {
  const viewLoading = isMissingView
    ? queries.missingMovies.isLoading || queries.missingSeries.isLoading
    : queries.upgradableMovies.isLoading || queries.upgradableSeries.isLoading
  const isError = isMissingView
    ? queries.missingMovies.isError || queries.missingSeries.isError
    : queries.upgradableMovies.isError || queries.upgradableSeries.isError
  return { isLoading: globalLoading || viewLoading, isError }
}

type Mutations = ReturnType<typeof useSearchMutations>

function deriveIsSearching(task: { isRunning: boolean }, mutations: Mutations) {
  return (
    task.isRunning ||
    mutations.missing.all.isPending ||
    mutations.missing.movies.isPending ||
    mutations.missing.series.isPending ||
    mutations.upgradable.all.isPending ||
    mutations.upgradable.movies.isPending ||
    mutations.upgradable.series.isPending
  )
}

function buildQualityMaps(profiles: QualityProfile[] | undefined) {
  const names = new Map(profiles?.map((p) => [p.id, p.name]))
  const map = new Map<number, QualityProfile>(profiles?.map((p) => [p.id, p]))
  return { qualityProfileNames: names, qualityProfileMap: map }
}

function refetchQueries(queries: Queries, isMissingView: boolean) {
  if (isMissingView) {
    void queries.missingMovies.refetch()
    void queries.missingSeries.refetch()
  } else {
    void queries.upgradableMovies.refetch()
    void queries.upgradableSeries.refetch()
  }
}

export function useMissingPage() {
  const [view, setView] = useState<ViewMode>('missing')
  const [filter, setFilter] = useState<MediaFilter>('all')
  const queries = useMissingQueries()
  const { data: qualityProfiles } = useQualityProfiles()
  const mutations = useSearchMutations()
  const task = useTaskResultNotifier()
  const globalLoading = useGlobalLoading()

  const isMissingView = view === 'missing'
  const counts = deriveCounts(queries)
  const movieCount = isMissingView ? counts.missingMovieCount : counts.upgradableMovieCount
  const episodeCount = isMissingView ? counts.missingEpisodeCount : counts.upgradableEpisodeCount
  const { isLoading, isError } = deriveViewState(queries, isMissingView, globalLoading)

  return {
    view,
    setView,
    filter,
    setFilter,
    isMissingView,
    isLoading,
    isError,
    isSearching: deriveIsSearching(task, mutations),
    movieCount,
    episodeCount,
    totalCount: movieCount + episodeCount,
    searchCount: getSearchCount(filter, movieCount, episodeCount),
    searchButtonStyle: getSearchButtonStyle(filter, movieCount, episodeCount),
    handleRefetch: () => refetchQueries(queries, isMissingView),
    handleSearch: () => void executeSearch(() => mutations[view][filter].mutateAsync()),
    ...buildQualityMaps(qualityProfiles),
    missingMovies: queries.missingMovies.data ?? [],
    missingSeries: queries.missingSeries.data ?? [],
    upgradableMovies: queries.upgradableMovies.data ?? [],
    upgradableSeries: queries.upgradableSeries.data ?? [],
    ...counts,
  }
}
