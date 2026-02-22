import { useSearch } from '@tanstack/react-router'
import { Search } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'

import { SearchLoadingSkeleton } from './search-loading-skeleton'
import { SearchResultsContent } from './search-results-content'
import { SeriesRequestDialog } from './series-request-dialog'
import { useRequestSearch } from './use-request-search'

type PortalSearchPageProps = {
  q: string
}

export function PortalSearchPage({ q }: PortalSearchPageProps) {
  const query = q.trim() || ''
  const s = useRequestSearch(query)

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
      <SearchBody query={query} state={s} />

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

type SearchBodyProps = {
  query: string
  state: ReturnType<typeof useRequestSearch>
}

function SearchBody({ query, state: s }: SearchBodyProps) {
  if (s.isLoading && !s.hasLibraryResults && !s.hasRequestableResults) {
    return <SearchLoadingSkeleton />
  }

  if (!s.hasLibraryResults && !s.hasRequestableResults) {
    return (
      <EmptyState
        icon={<Search className="size-8" />}
        title="No results found"
        description={`No movies or series found for "${query}"`}
      />
    )
  }

  return (
    <SearchResultsContent
      query={query}
      isLoading={s.isLoading}
      hasLibraryResults={s.hasLibraryResults}
      hasRequestableResults={s.hasRequestableResults}
      libraryMovies={s.libraryMovies}
      librarySeriesItems={s.librarySeriesItems}
      partialSeries={s.partialSeries}
      requestableMovies={s.requestableMovies}
      requestableSeries={s.requestableSeries}
      currentUserId={s.user?.id}
      isRequested={s.isRequested}
      onMovieRequest={s.handleMovieRequest}
      onSeriesRequestClick={s.handleSeriesRequestClick}
      onViewRequest={s.goToRequest}
    />
  )
}

export function PortalSearchPageWrapper() {
  const searchParams: { q?: string } = useSearch({ strict: false })
  return <PortalSearchPage q={searchParams.q ?? ''} />
}
