import { useState, useEffect } from 'react'
import { Film, Tv, Binoculars, Zap, Loader2 } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { LoadingState } from '@/components/data/LoadingState'
import { ErrorState } from '@/components/data/ErrorState'
import { MissingMoviesList } from '@/components/missing/MissingMoviesList'
import { MissingSeriesList } from '@/components/missing/MissingSeriesList'
import {
  useMissingMovies,
  useMissingSeries,
  useSearchAllMissing,
  useSearchAllMissingMovies,
  useSearchAllMissingSeries,
} from '@/hooks'
import { useAutoSearchStore } from '@/stores'
import { toast } from 'sonner'

type MediaFilter = 'all' | 'movies' | 'series'

export function MissingPage() {
  const [filter, setFilter] = useState<MediaFilter>('all')

  const {
    data: movies,
    isLoading: moviesLoading,
    isError: moviesError,
    refetch: refetchMovies,
  } = useMissingMovies()

  const {
    data: series,
    isLoading: seriesLoading,
    isError: seriesError,
    refetch: refetchSeries,
  } = useMissingSeries()

  const searchAllMutation = useSearchAllMissing()
  const searchMoviesMutation = useSearchAllMissingMovies()
  const searchSeriesMutation = useSearchAllMissingSeries()

  const { task, clearResult } = useAutoSearchStore()
  const isSearching = task.isRunning || searchAllMutation.isPending || searchMoviesMutation.isPending || searchSeriesMutation.isPending

  useEffect(() => {
    if (task.result) {
      const { downloaded, found, failed, totalSearched } = task.result
      if (downloaded > 0) {
        toast.success(`Downloaded ${downloaded} release${downloaded !== 1 ? 's' : ''}`, {
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

  const isLoading = moviesLoading || seriesLoading
  const isError = moviesError || seriesError

  const movieCount = movies?.length || 0
  const episodeCount = series?.reduce((acc, s) => acc + s.missingCount, 0) || 0
  const totalCount = movieCount + episodeCount

  const handleRefetch = () => {
    refetchMovies()
    refetchSeries()
  }

  const handleSearchAll = async () => {
    try {
      await searchAllMutation.mutateAsync()
    } catch (error) {
      if (error instanceof Error && error.message.includes('409')) {
        toast.warning('A search task is already running')
      } else {
        toast.error('Failed to start search')
      }
    }
  }

  const handleSearchMovies = async () => {
    try {
      await searchMoviesMutation.mutateAsync()
    } catch (error) {
      if (error instanceof Error && error.message.includes('409')) {
        toast.warning('A search task is already running')
      } else {
        toast.error('Failed to start search')
      }
    }
  }

  const handleSearchSeries = async () => {
    try {
      await searchSeriesMutation.mutateAsync()
    } catch (error) {
      if (error instanceof Error && error.message.includes('409')) {
        toast.warning('A search task is already running')
      } else {
        toast.error('Failed to start search')
      }
    }
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Missing" />
        <LoadingState variant="list" />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Missing" />
        <ErrorState onRetry={handleRefetch} />
      </div>
    )
  }

  const getSearchHandler = () => {
    switch (filter) {
      case 'movies':
        return handleSearchMovies
      case 'series':
        return handleSearchSeries
      default:
        return handleSearchAll
    }
  }

  const getSearchCount = () => {
    switch (filter) {
      case 'movies':
        return movieCount
      case 'series':
        return episodeCount
      default:
        return totalCount
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

  return (
    <div>
      <PageHeader
        title="Missing"
        description="Media that has been released but not yet downloaded"
        actions={
          searchCount > 0 && (
            <Button
              disabled={isSearching}
              onClick={getSearchHandler()}
              className={cn(getSearchButtonStyle())}
            >
              {isSearching ? (
                <Loader2 className="size-4 mr-2 animate-spin" />
              ) : (
                <Zap className="size-4 mr-2" />
              )}
              Search All ({searchCount})
            </Button>
          )
        }
      />

      <Tabs
        value={filter}
        onValueChange={(v) => setFilter(v as MediaFilter)}
        className="space-y-4"
      >
        <TabsList>
          <TabsTrigger
            value="all"
            className="px-4 data-active:bg-white data-active:text-black data-active:glow-media-sm"
          >
            All
            {totalCount > 0 && (
              <span className="ml-2 text-xs data-active:text-black/60">
                ({totalCount})
              </span>
            )}
          </TabsTrigger>
          <TabsTrigger
            value="movies"
            className="data-active:bg-white data-active:text-black data-active:glow-movie"
          >
            <Film className="size-4 mr-1.5" />
            Movies
            {movieCount > 0 && (
              <span className="ml-2 text-xs text-muted-foreground">
                ({movieCount})
              </span>
            )}
          </TabsTrigger>
          <TabsTrigger
            value="series"
            className="data-active:bg-white data-active:text-black data-active:glow-tv"
          >
            <Tv className="size-4 mr-1.5" />
            Series
            {episodeCount > 0 && (
              <span className="ml-2 text-xs text-muted-foreground">
                ({episodeCount})
              </span>
            )}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="all" className="space-y-6">
          {movieCount > 0 && (
            <div className="space-y-3">
              <h2 className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                <Film className="size-4 text-movie-400" />
                Movies
                <span className="text-movie-400">({movieCount})</span>
              </h2>
              <MissingMoviesList movies={movies || []} />
            </div>
          )}

          {(series?.length || 0) > 0 && (
            <div className="space-y-3">
              <h2 className="flex items-center gap-2 text-sm font-medium text-muted-foreground">
                <Tv className="size-4 text-tv-400" />
                Episodes
                <span className="text-tv-400">({episodeCount})</span>
              </h2>
              <MissingSeriesList series={series || []} />
            </div>
          )}

          {movieCount === 0 && episodeCount === 0 && (
            <div className="flex flex-col items-center justify-center text-center py-16">
              <Binoculars className="size-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-medium">No missing media</h3>
              <p className="text-muted-foreground mt-1">
                All monitored media that has been released has been downloaded
              </p>
            </div>
          )}
        </TabsContent>

        <TabsContent value="movies">
          <MissingMoviesList movies={movies || []} />
        </TabsContent>

        <TabsContent value="series">
          <MissingSeriesList series={series || []} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
