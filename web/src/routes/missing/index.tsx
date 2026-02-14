import { useEffect, useState } from 'react'

import { Binoculars, Film, Loader2, TrendingUp, Tv, Zap } from 'lucide-react'
import { toast } from 'sonner'

import { ErrorState } from '@/components/data/ErrorState'
import { PageHeader } from '@/components/layout/PageHeader'
import { MissingMoviesList } from '@/components/missing/MissingMoviesList'
import { MissingSeriesList } from '@/components/missing/MissingSeriesList'
import { UpgradableMoviesList } from '@/components/missing/UpgradableMoviesList'
import { UpgradableSeriesList } from '@/components/missing/UpgradableSeriesList'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, tabsListVariants, TabsTrigger } from '@/components/ui/tabs'
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
import { cn } from '@/lib/utils'
import { useAutoSearchStore } from '@/stores'
import type { QualityProfile } from '@/types/qualityProfile'

type ViewMode = 'missing' | 'upgradable'
type MediaFilter = 'all' | 'movies' | 'series'

const handleSearch = async (searchFn: () => Promise<unknown>) => {
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

export function MissingPage() {
  const [view, setView] = useState<ViewMode>('missing')
  const [filter, setFilter] = useState<MediaFilter>('all')

  // Missing data
  const {
    data: missingMovies,
    isLoading: missingMoviesLoading,
    isError: missingMoviesError,
    refetch: refetchMissingMovies,
  } = useMissingMovies()

  const {
    data: missingSeries,
    isLoading: missingSeriesLoading,
    isError: missingSeriesError,
    refetch: refetchMissingSeries,
  } = useMissingSeries()

  // Upgradable data
  const {
    data: upgradableMovies,
    isLoading: upgradableMoviesLoading,
    isError: upgradableMoviesError,
    refetch: refetchUpgradableMovies,
  } = useUpgradableMovies()

  const {
    data: upgradableSeries,
    isLoading: upgradableSeriesLoading,
    isError: upgradableSeriesError,
    refetch: refetchUpgradableSeries,
  } = useUpgradableSeries()

  const { data: qualityProfiles } = useQualityProfiles()
  const qualityProfileNames = new Map(qualityProfiles?.map((p) => [p.id, p.name]))
  const qualityProfileMap = new Map<number, QualityProfile>(qualityProfiles?.map((p) => [p.id, p]))

  // Missing search mutations
  const searchAllMissingMutation = useSearchAllMissing()
  const searchMissingMoviesMutation = useSearchAllMissingMovies()
  const searchMissingSeriesMutation = useSearchAllMissingSeries()

  // Upgradable search mutations
  const searchAllUpgradableMutation = useSearchAllUpgradable()
  const searchUpgradableMoviesMutation = useSearchAllUpgradableMovies()
  const searchUpgradableSeriesMutation = useSearchAllUpgradableSeries()

  const { task, clearResult } = useAutoSearchStore()
  const isSearching =
    task.isRunning ||
    searchAllMissingMutation.isPending ||
    searchMissingMoviesMutation.isPending ||
    searchMissingSeriesMutation.isPending ||
    searchAllUpgradableMutation.isPending ||
    searchUpgradableMoviesMutation.isPending ||
    searchUpgradableSeriesMutation.isPending

  useEffect(() => {
    if (task.result) {
      const { downloaded, found, failed, totalSearched } = task.result
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
      clearResult()
    }
  }, [task.result, clearResult])

  // Counts
  const missingMovieCount = missingMovies?.length || 0
  const missingEpisodeCount = missingSeries?.reduce((acc, s) => acc + s.missingCount, 0) || 0
  const missingTotalCount = missingMovieCount + missingEpisodeCount

  const upgradableMovieCount = upgradableMovies?.length || 0
  const upgradableEpisodeCount =
    upgradableSeries?.reduce((acc, s) => acc + s.upgradableCount, 0) || 0
  const upgradableTotalCount = upgradableMovieCount + upgradableEpisodeCount

  const isMissingView = view === 'missing'
  const movieCount = isMissingView ? missingMovieCount : upgradableMovieCount
  const episodeCount = isMissingView ? missingEpisodeCount : upgradableEpisodeCount
  const totalCount = isMissingView ? missingTotalCount : upgradableTotalCount

  const globalLoading = useGlobalLoading()
  const isLoading =
    globalLoading ||
    (isMissingView
      ? missingMoviesLoading || missingSeriesLoading
      : upgradableMoviesLoading || upgradableSeriesLoading)

  const isError = isMissingView
    ? missingMoviesError || missingSeriesError
    : upgradableMoviesError || upgradableSeriesError

  const handleRefetch = () => {
    if (isMissingView) {
      refetchMissingMovies()
      refetchMissingSeries()
    } else {
      refetchUpgradableMovies()
      refetchUpgradableSeries()
    }
  }

  const getSearchHandler = () => {
    if (isMissingView) {
      switch (filter) {
        case 'movies': {
          return () => handleSearch(() => searchMissingMoviesMutation.mutateAsync())
        }
        case 'series': {
          return () => handleSearch(() => searchMissingSeriesMutation.mutateAsync())
        }
        default: {
          return () => handleSearch(() => searchAllMissingMutation.mutateAsync())
        }
      }
    } else {
      switch (filter) {
        case 'movies': {
          return () => handleSearch(() => searchUpgradableMoviesMutation.mutateAsync())
        }
        case 'series': {
          return () => handleSearch(() => searchUpgradableSeriesMutation.mutateAsync())
        }
        default: {
          return () => handleSearch(() => searchAllUpgradableMutation.mutateAsync())
        }
      }
    }
  }

  const getSearchCount = () => {
    switch (filter) {
      case 'movies': {
        return movieCount
      }
      case 'series': {
        return episodeCount
      }
      default: {
        return totalCount
      }
    }
  }

  const searchCount = getSearchCount()

  const getSearchButtonStyle = () => {
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

  if (isError) {
    return (
      <div>
        <PageHeader title={isMissingView ? 'Missing' : 'Upgradable'} />
        <ErrorState onRetry={handleRefetch} />
      </div>
    )
  }

  return (
    <div>
      <PageHeader
        title={isMissingView ? 'Missing' : 'Upgradable'}
        description={
          isLoading ? (
            <Skeleton className="h-4 w-64" />
          ) : isMissingView ? (
            'Media that has been released but not yet downloaded'
          ) : (
            'Media with files below the quality cutoff'
          )
        }
        actions={
          <div className="flex items-center gap-2">
            {isLoading ? (
              <Button disabled>
                <Zap className="mr-2 size-4" />
                Search All
              </Button>
            ) : searchCount > 0 ? (
              <Button
                disabled={isSearching}
                onClick={getSearchHandler()}
                className={cn(getSearchButtonStyle())}
              >
                {isSearching ? (
                  <Loader2 className="mr-2 size-4 animate-spin" />
                ) : (
                  <Zap className="mr-2 size-4" />
                )}
                Search All ({searchCount})
              </Button>
            ) : null}
          </div>
        }
      />

      <Tabs value={filter} onValueChange={(v) => setFilter(v as MediaFilter)} className="space-y-4">
        <div
          className={cn(
            'flex flex-wrap items-center justify-between gap-3',
            isLoading && 'pointer-events-none opacity-50',
          )}
        >
          <TabsList>
            <TabsTrigger
              value="all"
              className="data-active:glow-media-sm px-4 data-active:bg-white data-active:text-black"
            >
              All
              {!isLoading && totalCount > 0 && (
                <span className="ml-2 text-xs data-active:text-black/60">({totalCount})</span>
              )}
            </TabsTrigger>
            <TabsTrigger
              value="movies"
              className="data-active:glow-movie data-active:bg-white data-active:text-black"
            >
              <Film className="mr-1.5 size-4" />
              Movies
              {!isLoading && movieCount > 0 && (
                <span className="text-muted-foreground ml-2 text-xs">({movieCount})</span>
              )}
            </TabsTrigger>
            <TabsTrigger
              value="series"
              className="data-active:glow-tv data-active:bg-white data-active:text-black"
            >
              <Tv className="mr-1.5 size-4" />
              Series
              {!isLoading && episodeCount > 0 && (
                <span className="text-muted-foreground ml-2 text-xs">({episodeCount})</span>
              )}
            </TabsTrigger>
          </TabsList>

          <div className={tabsListVariants()}>
            <button
              onClick={() => setView('missing')}
              className={cn(
                'inline-flex h-[calc(100%-1px)] items-center justify-center gap-1.5 rounded-md border border-transparent px-1.5 py-0.5 text-sm font-medium whitespace-nowrap transition-all',
                isMissingView
                  ? 'bg-background text-foreground dark:border-input dark:bg-input/30 shadow-sm'
                  : 'text-foreground/60 hover:text-foreground dark:text-muted-foreground dark:hover:text-foreground',
              )}
            >
              <Binoculars className="size-4" />
              Missing
            </button>
            <button
              onClick={() => setView('upgradable')}
              className={cn(
                'inline-flex h-[calc(100%-1px)] items-center justify-center gap-1.5 rounded-md border border-transparent px-1.5 py-0.5 text-sm font-medium whitespace-nowrap transition-all',
                isMissingView
                  ? 'text-foreground/60 hover:text-foreground dark:text-muted-foreground dark:hover:text-foreground'
                  : 'bg-background text-foreground dark:border-input dark:bg-input/30 shadow-sm',
              )}
            >
              <TrendingUp className="size-4" />
              Upgradable
            </button>
          </div>
        </div>

        {isLoading ? (
          <div className="mt-4 space-y-6">
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <Skeleton className="size-4 rounded-full" />
                <Skeleton className="h-4 w-16" />
                <Skeleton className="h-4 w-8" />
              </div>
              <MissingSkeletonRows count={5} />
            </div>
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <Skeleton className="size-4 rounded-full" />
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-4 w-8" />
              </div>
              <MissingSkeletonRows count={4} />
            </div>
          </div>
        ) : isMissingView ? (
          <>
            <TabsContent value="all" className="space-y-6">
              {missingMovieCount > 0 && (
                <div className="space-y-3">
                  <h2 className="text-muted-foreground flex items-center gap-2 text-sm font-medium">
                    <Film className="text-movie-400 size-4" />
                    Movies
                    <span className="text-movie-400">({missingMovieCount})</span>
                  </h2>
                  <MissingMoviesList
                    movies={missingMovies || []}
                    qualityProfileNames={qualityProfileNames}
                  />
                </div>
              )}

              {(missingSeries?.length || 0) > 0 && (
                <div className="space-y-3">
                  <h2 className="text-muted-foreground flex items-center gap-2 text-sm font-medium">
                    <Tv className="text-tv-400 size-4" />
                    Episodes
                    <span className="text-tv-400">({missingEpisodeCount})</span>
                  </h2>
                  <MissingSeriesList
                    series={missingSeries || []}
                    qualityProfileNames={qualityProfileNames}
                  />
                </div>
              )}

              {missingMovieCount === 0 && missingEpisodeCount === 0 && (
                <div className="flex flex-col items-center justify-center py-16 text-center">
                  <Binoculars className="text-muted-foreground mb-4 size-12" />
                  <h3 className="text-lg font-medium">No missing media</h3>
                  <p className="text-muted-foreground mt-1">
                    All monitored media that has been released has been downloaded
                  </p>
                </div>
              )}
            </TabsContent>

            <TabsContent value="movies">
              <MissingMoviesList
                movies={missingMovies || []}
                qualityProfileNames={qualityProfileNames}
              />
            </TabsContent>

            <TabsContent value="series">
              <MissingSeriesList
                series={missingSeries || []}
                qualityProfileNames={qualityProfileNames}
              />
            </TabsContent>
          </>
        ) : (
          <>
            <TabsContent value="all" className="space-y-6">
              {upgradableMovieCount > 0 && (
                <div className="space-y-3">
                  <h2 className="text-muted-foreground flex items-center gap-2 text-sm font-medium">
                    <Film className="text-movie-400 size-4" />
                    Movies
                    <span className="text-movie-400">({upgradableMovieCount})</span>
                  </h2>
                  <UpgradableMoviesList
                    movies={upgradableMovies || []}
                    qualityProfiles={qualityProfileMap}
                  />
                </div>
              )}

              {(upgradableSeries?.length || 0) > 0 && (
                <div className="space-y-3">
                  <h2 className="text-muted-foreground flex items-center gap-2 text-sm font-medium">
                    <Tv className="text-tv-400 size-4" />
                    Episodes
                    <span className="text-tv-400">({upgradableEpisodeCount})</span>
                  </h2>
                  <UpgradableSeriesList
                    series={upgradableSeries || []}
                    qualityProfiles={qualityProfileMap}
                  />
                </div>
              )}

              {upgradableMovieCount === 0 && upgradableEpisodeCount === 0 && (
                <div className="flex flex-col items-center justify-center py-16 text-center">
                  <TrendingUp className="text-muted-foreground mb-4 size-12" />
                  <h3 className="text-lg font-medium">No upgradable media</h3>
                  <p className="text-muted-foreground mt-1">
                    All monitored media meets the quality cutoff
                  </p>
                </div>
              )}
            </TabsContent>

            <TabsContent value="movies">
              <UpgradableMoviesList
                movies={upgradableMovies || []}
                qualityProfiles={qualityProfileMap}
              />
            </TabsContent>

            <TabsContent value="series">
              <UpgradableSeriesList
                series={upgradableSeries || []}
                qualityProfiles={qualityProfileMap}
              />
            </TabsContent>
          </>
        )}
      </Tabs>
    </div>
  )
}

function MissingSkeletonRows({ count }: { count: number }) {
  return (
    <div className="space-y-2">
      {Array.from({ length: count }, (_, i) => (
        <div
          key={i}
          className="border-border bg-card flex items-center gap-4 rounded-lg border px-4 py-3"
        >
          <Skeleton className="hidden h-[60px] w-10 shrink-0 rounded-md sm:block" />
          <div className="min-w-0 flex-1 space-y-1.5">
            <div className="flex items-baseline gap-2">
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-3 w-10" />
            </div>
            <Skeleton className="h-4 w-20 rounded-full" />
          </div>
          <div className="ml-auto flex shrink-0 items-center gap-1.5">
            <Skeleton className="size-8 rounded-md" />
            <Skeleton className="size-8 rounded-md" />
          </div>
        </div>
      ))}
    </div>
  )
}
