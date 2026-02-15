import { Plus } from 'lucide-react'

import { SearchResultsSection } from '@/components/search'
import type { PortalMovieSearchResult, PortalSeriesSearchResult } from '@/types'

import { MovieGrid, SeriesGrid } from './search-media-grids'

type SearchResultsContentProps = {
  query: string
  isLoading: boolean
  hasLibraryResults: boolean
  hasRequestableResults: boolean
  libraryMovies: PortalMovieSearchResult[]
  librarySeriesItems: PortalSeriesSearchResult[]
  requestableMovies: PortalMovieSearchResult[]
  requestableSeries: PortalSeriesSearchResult[]
  currentUserId?: number
  isRequested: (tmdbId: number) => boolean
  onMovieRequest: (movie: PortalMovieSearchResult) => void
  onSeriesRequestClick: (item: PortalSeriesSearchResult) => void
  onViewRequest: (id: number) => void
}

export function SearchResultsContent({
  query,
  isLoading,
  hasLibraryResults,
  hasRequestableResults,
  libraryMovies,
  librarySeriesItems,
  requestableMovies,
  requestableSeries,
  currentUserId,
  isRequested,
  onMovieRequest,
  onSeriesRequestClick,
  onViewRequest,
}: SearchResultsContentProps) {
  return (
    <>
      {hasLibraryResults ? (
        <LibrarySection
          isLoading={isLoading}
          libraryMovies={libraryMovies}
          librarySeriesItems={librarySeriesItems}
          currentUserId={currentUserId}
          onViewRequest={onViewRequest}
        />
      ) : null}

      <RequestableSection
        query={query}
        isLoading={isLoading}
        hasLibraryResults={hasLibraryResults}
        hasRequestableResults={hasRequestableResults}
        requestableMovies={requestableMovies}
        requestableSeries={requestableSeries}
        currentUserId={currentUserId}
        isRequested={isRequested}
        onMovieRequest={onMovieRequest}
        onSeriesRequestClick={onSeriesRequestClick}
        onViewRequest={onViewRequest}
      />
    </>
  )
}

type LibrarySectionProps = {
  isLoading: boolean
  libraryMovies: PortalMovieSearchResult[]
  librarySeriesItems: PortalSeriesSearchResult[]
  currentUserId?: number
  onViewRequest: (id: number) => void
}

function LibrarySection({
  isLoading,
  libraryMovies,
  librarySeriesItems,
  currentUserId,
  onViewRequest,
}: LibrarySectionProps) {
  return (
    <SearchResultsSection title="In Library" isLoading={isLoading} hasResults>
      <div className="space-y-6">
        <MovieGrid
          items={libraryMovies}
          currentUserId={currentUserId}
          onViewRequest={onViewRequest}
        />
        <SeriesGrid
          items={librarySeriesItems}
          currentUserId={currentUserId}
          onViewRequest={onViewRequest}
        />
      </div>
    </SearchResultsSection>
  )
}

type RequestableSectionProps = {
  query: string
  isLoading: boolean
  hasLibraryResults: boolean
  hasRequestableResults: boolean
  requestableMovies: PortalMovieSearchResult[]
  requestableSeries: PortalSeriesSearchResult[]
  currentUserId?: number
  isRequested: (tmdbId: number) => boolean
  onMovieRequest: (movie: PortalMovieSearchResult) => void
  onSeriesRequestClick: (item: PortalSeriesSearchResult) => void
  onViewRequest: (id: number) => void
}

function RequestableSectionContent({
  requestableMovies,
  requestableSeries,
  currentUserId,
  isRequested,
  onMovieRequest,
  onSeriesRequestClick,
  onViewRequest,
}: Omit<RequestableSectionProps, 'query' | 'isLoading' | 'hasLibraryResults' | 'hasRequestableResults'>) {
  return (
    <div className="space-y-6">
      <MovieGrid
        items={requestableMovies}
        currentUserId={currentUserId}
        collapsible={false}
        isRequested={isRequested}
        onAction={onMovieRequest}
        onViewRequest={onViewRequest}
      />
      <SeriesGrid
        items={requestableSeries}
        currentUserId={currentUserId}
        collapsible={false}
        isRequested={isRequested}
        onAction={onSeriesRequestClick}
        onViewRequest={onViewRequest}
      />
    </div>
  )
}

function RequestableSection({
  query,
  isLoading,
  hasLibraryResults,
  hasRequestableResults,
  requestableMovies,
  requestableSeries,
  currentUserId,
  isRequested,
  onMovieRequest,
  onSeriesRequestClick,
  onViewRequest,
}: RequestableSectionProps) {
  const gridProps = {
    requestableMovies,
    requestableSeries,
    currentUserId,
    isRequested,
    onMovieRequest,
    onSeriesRequestClick,
    onViewRequest,
  }

  if (!hasLibraryResults) {
    return <RequestableSectionContent {...gridProps} />
  }

  return (
    <SearchResultsSection
      title="Request"
      icon={<Plus className="size-5" />}
      isLoading={isLoading}
      hasResults={hasRequestableResults}
      emptyMessage={`No new content found for "${query}"`}
    >
      <RequestableSectionContent {...gridProps} />
    </SearchResultsSection>
  )
}
