import { useCallback, useState } from 'react'

import { useNavigate, useSearch } from '@tanstack/react-router'
import { Loader2, Plus, Search } from 'lucide-react'
import { toast } from 'sonner'

import { EmptyState } from '@/components/data/EmptyState'
import { ExpandableMediaGrid, ExternalMediaCard, SearchResultsSection } from '@/components/search'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import {
  useCreateRequest,
  usePortalMovieSearch,
  usePortalSeriesSearch,
  useSeriesSeasons,
} from '@/hooks'
import { usePortalAuthStore } from '@/stores'
import type {
  MovieSearchResult,
  PortalMovieSearchResult,
  PortalSeriesSearchResult,
  SeriesSearchResult,
} from '@/types'

type PortalSearchPageProps = {
  q: string
}

function convertToMovieSearchResult(movie: PortalMovieSearchResult): MovieSearchResult {
  // Backend returns TMDB ID as 'id', not 'tmdbId' for movies
  const tmdbId = movie.tmdbId || movie.id
  return {
    id: tmdbId,
    tmdbId,
    title: movie.title,
    year: movie.year ?? undefined,
    overview: movie.overview ?? undefined,
    posterUrl: movie.posterUrl ?? undefined,
    backdropUrl: movie.backdropUrl ?? undefined,
  }
}

function convertToSeriesSearchResult(series: PortalSeriesSearchResult): SeriesSearchResult {
  // Backend returns TMDB ID as 'id' or 'tmdbId' for series
  const tmdbId = series.tmdbId || series.id
  return {
    id: tmdbId,
    tmdbId,
    tvdbId: series.tvdbId ?? undefined,
    title: series.title,
    year: series.year ?? undefined,
    overview: series.overview ?? undefined,
    posterUrl: series.posterUrl ?? undefined,
    backdropUrl: series.backdropUrl ?? undefined,
  }
}

export function PortalSearchPage({ q }: PortalSearchPageProps) {
  const navigate = useNavigate()
  const query = q?.trim() || ''
  const { user } = usePortalAuthStore()

  const { data: movies = [], isLoading: loadingMovies } = usePortalMovieSearch(query)
  const { data: series = [], isLoading: loadingSeries } = usePortalSeriesSearch(query)
  const createRequest = useCreateRequest()

  const [requestDialogOpen, setRequestDialogOpen] = useState(false)
  const [selectedSeries, setSelectedSeries] = useState<PortalSeriesSearchResult | null>(null)
  const [monitorFuture, setMonitorFuture] = useState(false)
  const [selectedSeasons, setSelectedSeasons] = useState<Set<number>>(new Set())
  const [requestedTmdbIds, setRequestedTmdbIds] = useState<Set<number>>(new Set())

  const { data: seasons = [], isLoading: loadingSeasons } = useSeriesSeasons(
    selectedSeries?.tmdbId || selectedSeries?.id,
    selectedSeries?.tvdbId || undefined,
  )

  const isRequested = useCallback(
    (tmdbId: number) => requestedTmdbIds.has(tmdbId),
    [requestedTmdbIds],
  )

  const isLoading = loadingMovies || loadingSeries

  // Split results into library items and requestable items
  // Sort library items by addedAt descending (newest first)
  const sortByAddedAt = <T extends { availability?: { addedAt?: string | null } }>(
    items: T[],
  ): T[] =>
    [...items].sort((a, b) => {
      const aDate = a.availability?.addedAt ? new Date(a.availability.addedAt).getTime() : 0
      const bDate = b.availability?.addedAt ? new Date(b.availability.addedAt).getTime() : 0
      return bDate - aDate
    })

  const libraryMovies = sortByAddedAt(movies.filter((m) => m.availability?.inLibrary))
  const librarySeriesItems = sortByAddedAt(series.filter((s) => s.availability?.inLibrary))
  const requestableMovies = movies.filter((m) => !m.availability?.inLibrary)
  const requestableSeries = series.filter((s) => !s.availability?.inLibrary)

  const hasLibraryResults = libraryMovies.length > 0 || librarySeriesItems.length > 0
  const hasRequestableResults = requestableMovies.length > 0 || requestableSeries.length > 0

  const handleMovieRequest = (movie: PortalMovieSearchResult) => {
    const tmdbId = movie.tmdbId || movie.id
    createRequest.mutate(
      {
        mediaType: 'movie',
        tmdbId,
        title: movie.title,
        year: movie.year || undefined,
        posterUrl: movie.posterUrl || undefined,
      },
      {
        onSuccess: () => {
          setRequestedTmdbIds((prev) => new Set(prev).add(tmdbId))
          toast.success('Request submitted', {
            description: 'Your request is being processed',
          })
        },
        onError: (error) => {
          toast.error('Failed to submit request', {
            description: error.message,
          })
        },
      },
    )
  }

  const handleSeriesRequestClick = (series: PortalSeriesSearchResult) => {
    setSelectedSeries(series)
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

  const selectAllSeasons = () => {
    setSelectedSeasons(new Set(seasons.map((s) => s.seasonNumber)))
  }

  const deselectAllSeasons = () => {
    setSelectedSeasons(new Set())
  }

  const handleSubmitSeriesRequest = () => {
    if (!selectedSeries) {
      return
    }

    const tmdbId = selectedSeries.tmdbId || selectedSeries.id
    const seasonsArray = [...selectedSeasons].sort((a, b) => a - b)

    createRequest.mutate(
      {
        mediaType: 'series',
        tmdbId,
        tvdbId: selectedSeries.tvdbId || undefined,
        title: selectedSeries.title,
        year: selectedSeries.year || undefined,
        monitorFuture,
        posterUrl: selectedSeries.posterUrl || undefined,
        requestedSeasons: seasonsArray.length > 0 ? seasonsArray : undefined,
      },
      {
        onSuccess: () => {
          setRequestedTmdbIds((prev) => new Set(prev).add(tmdbId))
          setRequestDialogOpen(false)
          setSelectedSeries(null)
          setSelectedSeasons(new Set())
          toast.success('Request submitted', {
            description: 'Your request is being processed',
          })
        },
        onError: (error) => {
          toast.error('Failed to submit request', {
            description: error.message,
          })
        },
      },
    )
  }

  const goToRequest = (id: number) => {
    navigate({ to: '/requests/$id', params: { id: String(id) } })
  }

  if (!query) {
    return (
      <div className="mx-auto max-w-6xl px-6 pt-6">
        <EmptyState
          icon={<Search className="size-8" />}
          title="Search for content"
          description="Use the search bar above to find movies and series to request"
        />
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-6xl space-y-8 px-6 pt-6">
      {isLoading && !hasLibraryResults && !hasRequestableResults ? (
        <div className="grid grid-cols-3 gap-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-7 xl:grid-cols-8">
          {Array.from({ length: 12 }).map((_, i) => (
            <div key={i} className="bg-muted aspect-[2/3] animate-pulse rounded-lg" />
          ))}
        </div>
      ) : !hasLibraryResults && !hasRequestableResults ? (
        <EmptyState
          icon={<Search className="size-8" />}
          title="No results found"
          description={`No movies or series found for "${query}"`}
        />
      ) : (
        <>
          {/* In Library Section - only show if there are library results */}
          {hasLibraryResults ? (
            <SearchResultsSection
              title="In Library"
              isLoading={isLoading}
              hasResults={hasLibraryResults}
            >
              <div className="space-y-6">
                <ExpandableMediaGrid
                  items={libraryMovies}
                  getKey={(movie) => movie.tmdbId}
                  label="Movies"
                  icon="movie"
                  renderItem={(movie) => (
                    <ExternalMediaCard
                      media={convertToMovieSearchResult(movie)}
                      mediaType="movie"
                      availability={movie.availability}
                      currentUserId={user?.id}
                      onViewRequest={goToRequest}
                      actionLabel="Request"
                      actionIcon={<Plus className="mr-1 size-3 md:mr-2 md:size-4" />}
                      disabledLabel="In Library"
                    />
                  )}
                />
                <ExpandableMediaGrid
                  items={librarySeriesItems}
                  getKey={(s) => s.tmdbId}
                  label="Series"
                  icon="series"
                  renderItem={(item) => (
                    <ExternalMediaCard
                      media={convertToSeriesSearchResult(item)}
                      mediaType="series"
                      availability={item.availability}
                      currentUserId={user?.id}
                      onViewRequest={goToRequest}
                      actionLabel="Request"
                      actionIcon={<Plus className="mr-1 size-3 md:mr-2 md:size-4" />}
                      disabledLabel="In Library"
                    />
                  )}
                />
              </div>
            </SearchResultsSection>
          ) : null}

          {/* Request Section - hide header if no library results */}
          {hasLibraryResults ? (
            <SearchResultsSection
              title="Request"
              icon={<Plus className="size-5" />}
              isLoading={isLoading}
              hasResults={hasRequestableResults}
              emptyMessage={`No new content found for "${query}"`}
            >
              <div className="space-y-6">
                <ExpandableMediaGrid
                  items={requestableMovies}
                  getKey={(movie) => movie.tmdbId}
                  label="Movies"
                  icon="movie"
                  collapsible={false}
                  renderItem={(movie) => (
                    <ExternalMediaCard
                      media={convertToMovieSearchResult(movie)}
                      mediaType="movie"
                      availability={movie.availability}
                      requested={isRequested(movie.tmdbId || movie.id)}
                      currentUserId={user?.id}
                      onAction={() => handleMovieRequest(movie)}
                      onViewRequest={goToRequest}
                      actionLabel="Request"
                      actionIcon={<Plus className="mr-1 size-3 md:mr-2 md:size-4" />}
                      disabledLabel="In Library"
                    />
                  )}
                />
                <ExpandableMediaGrid
                  items={requestableSeries}
                  getKey={(s) => s.tmdbId}
                  label="Series"
                  icon="series"
                  collapsible={false}
                  renderItem={(item) => (
                    <ExternalMediaCard
                      media={convertToSeriesSearchResult(item)}
                      mediaType="series"
                      availability={item.availability}
                      requested={isRequested(item.tmdbId || item.id)}
                      currentUserId={user?.id}
                      onAction={() => handleSeriesRequestClick(item)}
                      onViewRequest={goToRequest}
                      actionLabel="Request"
                      actionIcon={<Plus className="mr-1 size-3 md:mr-2 md:size-4" />}
                      disabledLabel="In Library"
                    />
                  )}
                />
              </div>
            </SearchResultsSection>
          ) : (
            <div className="space-y-6">
              <ExpandableMediaGrid
                items={requestableMovies}
                getKey={(movie) => movie.tmdbId}
                label="Movies"
                icon="movie"
                collapsible={false}
                renderItem={(movie) => (
                  <ExternalMediaCard
                    media={convertToMovieSearchResult(movie)}
                    mediaType="movie"
                    availability={movie.availability}
                    requested={isRequested(movie.tmdbId || movie.id)}
                    currentUserId={user?.id}
                    onAction={() => handleMovieRequest(movie)}
                    onViewRequest={goToRequest}
                    actionLabel="Request"
                    actionIcon={<Plus className="mr-1 size-3 md:mr-2 md:size-4" />}
                    disabledLabel="In Library"
                  />
                )}
              />
              <ExpandableMediaGrid
                items={requestableSeries}
                getKey={(s) => s.tmdbId}
                label="Series"
                icon="series"
                collapsible={false}
                renderItem={(item) => (
                  <ExternalMediaCard
                    media={convertToSeriesSearchResult(item)}
                    mediaType="series"
                    availability={item.availability}
                    requested={isRequested(item.tmdbId || item.id)}
                    currentUserId={user?.id}
                    onAction={() => handleSeriesRequestClick(item)}
                    onViewRequest={goToRequest}
                    actionLabel="Request"
                    actionIcon={<Plus className="mr-1 size-3 md:mr-2 md:size-4" />}
                    disabledLabel="In Library"
                  />
                )}
              />
            </div>
          )}
        </>
      )}

      <Dialog open={requestDialogOpen} onOpenChange={setRequestDialogOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Request {selectedSeries?.title}</DialogTitle>
            <DialogDescription>
              Select which seasons to request and whether to monitor future episodes
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <Label htmlFor="monitorFuture" className="text-sm font-medium">
                Monitor future episodes
              </Label>
              <Switch
                id="monitorFuture"
                checked={monitorFuture}
                onCheckedChange={setMonitorFuture}
              />
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label className="text-sm font-medium">Seasons</Label>
                <div className="flex gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={selectAllSeasons}
                    disabled={loadingSeasons || seasons.length === 0}
                  >
                    All
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={deselectAllSeasons}
                    disabled={loadingSeasons || seasons.length === 0}
                  >
                    None
                  </Button>
                </div>
              </div>

              {loadingSeasons ? (
                <div className="space-y-2">
                  {Array.from({ length: 3 }).map((_, i) => (
                    <Skeleton key={i} className="h-8 w-full" />
                  ))}
                </div>
              ) : seasons.length === 0 ? (
                <p className="text-muted-foreground py-2 text-sm">
                  No season information available
                </p>
              ) : (
                <div className="max-h-48 space-y-1 overflow-y-auto rounded-md border p-2">
                  {seasons
                    .filter((s) => s.seasonNumber > 0)
                    .sort((a, b) => a.seasonNumber - b.seasonNumber)
                    .map((season) => (
                      <div key={season.seasonNumber} className="flex items-center space-x-2 py-1">
                        <Checkbox
                          id={`season-${season.seasonNumber}`}
                          checked={selectedSeasons.has(season.seasonNumber)}
                          onCheckedChange={() => toggleSeasonSelection(season.seasonNumber)}
                        />
                        <Label
                          htmlFor={`season-${season.seasonNumber}`}
                          className="flex-1 cursor-pointer text-sm"
                        >
                          Season {season.seasonNumber}
                          {season.name && season.name !== `Season ${season.seasonNumber}` ? (
                            <span className="text-muted-foreground ml-1">({season.name})</span>
                          ) : null}
                        </Label>
                      </div>
                    ))}
                </div>
              )}

              {selectedSeasons.size === 0 && monitorFuture ? (
                <p className="text-muted-foreground text-xs">
                  No seasons selected. Series will be added to library and only future episodes will
                  be monitored.
                </p>
              ) : null}
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setRequestDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleSubmitSeriesRequest}
              disabled={createRequest.isPending || (!monitorFuture && selectedSeasons.size === 0)}
            >
              {createRequest.isPending ? <Loader2 className="mr-2 size-4 animate-spin" /> : null}
              Submit Request
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

export function PortalSearchPageWrapper() {
  const { q } = useSearch({ strict: false })
  return <PortalSearchPage q={q || ''} />
}
