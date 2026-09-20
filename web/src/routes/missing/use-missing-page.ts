import { useEffect, useMemo, useState } from 'react'

import { toast } from 'sonner'

import {
  useMissingMovies,
  useMissingSeries,
  useQualityProfiles,
  useSearchAllMissingMovies,
  useSearchAllMissingSeries,
  useSearchAllUpgradableMovies,
  useSearchAllUpgradableSeries,
  useUpgradableMovies,
  useUpgradableSeries,
} from '@/hooks'
import { getEnabledModules } from '@/modules'
import { useAutoSearchStore, useUIStore } from '@/stores'
import type { QualityProfile } from '@/types/quality-profile'

export type ViewMode = 'missing' | 'upgradable'

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

function useMissingQueries() {
  return {
    missingMovies: useMissingMovies(),
    missingSeries: useMissingSeries(),
    upgradableMovies: useUpgradableMovies(),
    upgradableSeries: useUpgradableSeries(),
  }
}

function useSearchMutations() {
  return {
    missing: {
      movie: useSearchAllMissingMovies(),
      tv: useSearchAllMissingSeries(),
    },
    upgradable: {
      movie: useSearchAllUpgradableMovies(),
      tv: useSearchAllUpgradableSeries(),
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
type Mutations = ReturnType<typeof useSearchMutations>

function buildQualityMaps(profiles: QualityProfile[] | undefined) {
  return {
    qualityProfileNames: new Map(profiles?.map((p) => [p.id, p.name])),
    qualityProfileMap: new Map<number, QualityProfile>(profiles?.map((p) => [p.id, p])),
  }
}

function deriveIsSearching(task: { isRunning: boolean }, mutations: Mutations): boolean {
  return (
    task.isRunning ||
    mutations.missing.movie.isPending ||
    mutations.missing.tv.isPending ||
    mutations.upgradable.movie.isPending ||
    mutations.upgradable.tv.isPending
  )
}

function activeQueries(queries: Queries, moduleId: string, view: ViewMode) {
  if (view === 'missing') {
    return moduleId === 'movie' ? queries.missingMovies : queries.missingSeries
  }
  return moduleId === 'movie' ? queries.upgradableMovies : queries.upgradableSeries
}

function countFor(queries: Queries, moduleId: string, view: ViewMode): number {
  if (moduleId === 'movie') {
    const list = view === 'missing' ? queries.missingMovies.data : queries.upgradableMovies.data
    return list?.length ?? 0
  }
  if (view === 'missing') {
    return queries.missingSeries.data?.reduce((acc, s) => acc + s.missingCount, 0) ?? 0
  }
  return queries.upgradableSeries.data?.reduce((acc, s) => acc + s.upgradableCount, 0) ?? 0
}

export function useMissingPage() {
  const modules = getEnabledModules()
  const [moduleId, setModuleId] = useState(modules[0]?.id ?? 'movie')
  const [view, setView] = useState<ViewMode>('missing')

  const queries = useMissingQueries()
  const { data: qualityProfiles } = useQualityProfiles()
  const mutations = useSearchMutations()
  const task = useTaskResultNotifier()
  const globalLoading = useUIStore((s) => s.globalLoading)

  const active = activeQueries(queries, moduleId, view)
  const qualityMaps = useMemo(() => buildQualityMaps(qualityProfiles), [qualityProfiles])
  const searchMutation = view === 'missing' ? mutations.missing : mutations.upgradable

  return {
    modules,
    moduleId,
    setModuleId,
    view,
    setView,
    isLoading: globalLoading || active.isLoading,
    isError: active.isError,
    isSearching: deriveIsSearching(task, mutations),
    count: countFor(queries, moduleId, view),
    handleRefetch: () => {
      void active.refetch()
    },
    handleSearchAll: () => {
      const mutation = moduleId === 'movie' ? searchMutation.movie : searchMutation.tv
      void executeSearch(() => mutation.mutateAsync())
    },
    ...qualityMaps,
    missingMovies: queries.missingMovies.data ?? [],
    missingSeries: queries.missingSeries.data ?? [],
    upgradableMovies: queries.upgradableMovies.data ?? [],
    upgradableSeries: queries.upgradableSeries.data ?? [],
  }
}
