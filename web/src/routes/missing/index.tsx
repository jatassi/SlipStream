import { useState, useEffect } from 'react'
import { Film, Tv, Search, Zap, Loader2 } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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

  // Show toast when task completes
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

  // Count items
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

  return (
    <div>
      <PageHeader
        title="Missing"
        description="Media that has been released but not yet downloaded"
        actions={
          searchCount > 0 && (
            <Button disabled={isSearching} onClick={getSearchHandler()}>
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
          <TabsTrigger value="all">
            All
            {totalCount > 0 && (
              <span className="ml-2 text-xs text-muted-foreground">
                ({totalCount})
              </span>
            )}
          </TabsTrigger>
          <TabsTrigger value="movies">
            <Film className="size-4 mr-1" />
            Movies
            {movieCount > 0 && (
              <span className="ml-2 text-xs text-muted-foreground">
                ({movieCount})
              </span>
            )}
          </TabsTrigger>
          <TabsTrigger value="series">
            <Tv className="size-4 mr-1" />
            Series
            {episodeCount > 0 && (
              <span className="ml-2 text-xs text-muted-foreground">
                ({episodeCount})
              </span>
            )}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="all" className="space-y-4">
          {movieCount > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Film className="size-5" />
                  Missing Movies
                </CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <MissingMoviesList movies={movies || []} isSearchingAll={task.isRunning} />
              </CardContent>
            </Card>
          )}

          {(series?.length || 0) > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Tv className="size-5" />
                  Missing Episodes
                </CardTitle>
              </CardHeader>
              <CardContent>
                <MissingSeriesList series={series || []} isSearchingAll={task.isRunning} />
              </CardContent>
            </Card>
          )}

          {movieCount === 0 && episodeCount === 0 && (
            <Card>
              <CardContent className="py-12">
                <div className="flex flex-col items-center justify-center text-center">
                  <Search className="size-12 text-muted-foreground mb-4" />
                  <h3 className="text-lg font-medium">No missing media</h3>
                  <p className="text-muted-foreground mt-1">
                    All monitored media that has been released has been downloaded
                  </p>
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="movies">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Film className="size-5" />
                Missing Movies
              </CardTitle>
            </CardHeader>
            <CardContent className="p-0">
              <MissingMoviesList movies={movies || []} isSearchingAll={task.isRunning} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="series">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Tv className="size-5" />
                Missing Episodes
              </CardTitle>
            </CardHeader>
            <CardContent>
              <MissingSeriesList series={series || []} isSearchingAll={task.isRunning} />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
