import { Film, Library, Tv } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import { MediaGrid } from '@/components/media/media-grid'
import { Slider } from '@/components/ui/slider'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'
import { useUIStore } from '@/stores'
import type { PortalMovieSearchResult, PortalSeriesSearchResult } from '@/types'

import { LibraryMovieCard } from './library-movie-card'
import { LibrarySeriesCard } from './library-series-card'
import { SeriesRequestDialog } from './series-request-dialog'
import { usePortalLibrary } from './use-portal-library'

function CountBadge({ count, className }: { count: number; className?: string }) {
  if (count <= 0) {
    return null
  }
  return <span className={cn('ml-2 text-xs', className)}>({count})</span>
}

function GridSkeleton({ posterSize = 150 }: { posterSize?: number }) {
  return (
    <div className="grid gap-4" style={{ gridTemplateColumns: `repeat(auto-fill, minmax(${posterSize}px, 1fr))` }}>
      {Array.from({ length: 12 }, (_, i) => (
        <div key={i} className="bg-muted aspect-[2/3] animate-pulse rounded-lg" />
      ))}
    </div>
  )
}

function PosterSizeSlider() {
  const { posterSize, setPosterSize } = useUIStore()
  const isMobile = globalThis.matchMedia('(max-width: 639px)').matches
  return (
    <div className="ml-auto flex items-center gap-2">
      <span className="text-muted-foreground text-xs">Size</span>
      <Slider
        value={[posterSize]}
        onValueChange={(v) => setPosterSize(v[0])}
        min={100}
        max={250}
        step={isMobile ? 25 : 10}
        className="w-16 sm:w-24"
      />
    </div>
  )
}

function LibraryTabs({ s }: { s: ReturnType<typeof usePortalLibrary> }) {
  const { posterSize } = useUIStore()

  return (
    <Tabs value={s.activeTab} onValueChange={(v) => s.setActiveTab(v as 'movies' | 'series')}>
      <div className="flex items-center gap-2 sm:gap-3">
        <TabsList>
          <TabsTrigger
            value="movies"
            className="data-active:glow-movie px-2 data-active:bg-white data-active:text-black sm:px-4"
          >
            <Film className="mr-1.5 size-4" />
            Movies
            {!s.loadingMovies && <CountBadge count={s.movies.length} className="text-muted-foreground" />}
          </TabsTrigger>
          <TabsTrigger
            value="series"
            className="data-active:glow-tv px-2 data-active:bg-white data-active:text-black sm:px-4"
          >
            <Tv className="mr-1.5 size-4" />
            Series
            {!s.loadingSeries && <CountBadge count={s.series.length} className="text-muted-foreground" />}
          </TabsTrigger>
        </TabsList>
        <PosterSizeSlider />
      </div>

      <TabsContent value="movies">
        <MoviesTab
          movies={s.movies}
          loading={s.loadingMovies}
          posterSize={posterSize}
          currentUserId={s.user?.id}
          onAction={s.handleMovieRequest}
          onViewRequest={s.goToRequest}
        />
      </TabsContent>

      <TabsContent value="series">
        <SeriesTab
          series={s.series}
          loading={s.loadingSeries}
          posterSize={posterSize}
          currentUserId={s.user?.id}
          partialTmdbIds={s.partialTmdbIds}
          onAction={s.handleSeriesRequestClick}
          onViewRequest={s.goToRequest}
        />
      </TabsContent>
    </Tabs>
  )
}

export function PortalLibraryPage() {
  const s = usePortalLibrary()

  return (
    <div className="mx-auto max-w-6xl space-y-6 px-3 pt-6 sm:px-6">
      <LibraryTabs s={s} />

      <SeriesRequestDialog
        open={s.requestDialogOpen}
        onOpenChange={s.setRequestDialogOpen}
        seriesTitle={s.selectedSeries?.title}
        monitorFuture={s.monitorFuture}
        onMonitorFutureChange={s.setMonitorFuture}
        seasons={s.seasons}
        loadingSeasons={s.loadingSeasons}
        selectedSeasons={s.selectedSeasons}
        onToggleSeason={s.toggleSeasonSelection}
        onSelectAll={s.selectAllSeasons}
        onDeselectAll={s.deselectAllSeasons}
        onSubmit={s.handleSubmitSeriesRequest}
        isSubmitting={s.isSubmitting}
        onWatchRequest={s.handleWatchRequest}
      />
    </div>
  )
}

function MoviesTab({
  movies,
  loading,
  posterSize,
  currentUserId,
  onAction,
  onViewRequest,
}: {
  movies: PortalMovieSearchResult[]
  loading: boolean
  posterSize: number
  currentUserId?: number
  onAction: (movie: PortalMovieSearchResult) => void
  onViewRequest: (id: number) => void
}) {
  if (loading) {
    return <GridSkeleton posterSize={posterSize} />
  }

  if (movies.length === 0) {
    return (
      <EmptyState
        icon={<Library className="size-8" />}
        title="No movies available"
        description="Movies with files will appear here"
      />
    )
  }

  return (
    <MediaGrid
      items={movies}
      posterSize={posterSize}
      renderCard={(movie) => (
        <LibraryMovieCard
          key={movie.id}
          movie={movie}
          currentUserId={currentUserId}
          onAction={() => onAction(movie)}
          onViewRequest={onViewRequest}
        />
      )}
    />
  )
}

function SeriesTab({
  series,
  loading,
  posterSize,
  currentUserId,
  partialTmdbIds,
  onAction,
  onViewRequest,
}: {
  series: PortalSeriesSearchResult[]
  loading: boolean
  posterSize: number
  currentUserId?: number
  partialTmdbIds: Set<number>
  onAction: (item: PortalSeriesSearchResult) => void
  onViewRequest: (id: number) => void
}) {
  if (loading) {
    return <GridSkeleton posterSize={posterSize} />
  }

  if (series.length === 0) {
    return (
      <EmptyState
        icon={<Library className="size-8" />}
        title="No series available"
        description="Series with files will appear here"
      />
    )
  }

  return (
    <MediaGrid
      items={series}
      posterSize={posterSize}
      renderCard={(item) => (
        <LibrarySeriesCard
          key={item.id}
          series={item}
          currentUserId={currentUserId}
          isPartial={partialTmdbIds.has(item.tmdbId)}
          onAction={() => onAction(item)}
          onViewRequest={onViewRequest}
        />
      )}
    />
  )
}
