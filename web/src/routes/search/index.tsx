import { useMemo, useRef, useState } from 'react'

import { Search } from 'lucide-react'

import { EmptyState } from '@/components/data/EmptyState'
import { MovieCard } from '@/components/movies/MovieCard'
import {
  ExpandableMediaGrid,
  ExternalMovieCard,
  ExternalSearchSection,
  ExternalSeriesCard,
  SearchResultsSection,
} from '@/components/search'
import { SeriesCard } from '@/components/series/SeriesCard'
import { useMovies, useMovieSearch, useSeries, useSeriesSearch } from '@/hooks'
import { useAdminRequests } from '@/hooks/admin/useAdminRequests'

type SearchPageProps = {
  q: string
}

export function SearchPage({ q }: SearchPageProps) {
  const query = q.trim() || ''
  const [externalEnabled, setExternalEnabled] = useState(false)

  // Reset external search when query changes
  const prevQueryRef = useRef(query)
  if (prevQueryRef.current !== query) {
    prevQueryRef.current = query
    if (externalEnabled) {
      setExternalEnabled(false)
    }
  }

  // Fetch library results
  const { data: libraryMovies = [], isLoading: loadingLibraryMovies } = useMovies(
    query ? { search: query } : undefined,
  )
  const { data: librarySeries = [], isLoading: loadingLibrarySeries } = useSeries(
    query ? { search: query } : undefined,
  )

  const isLibraryLoading = loadingLibraryMovies || loadingLibrarySeries
  const hasLibraryResults = libraryMovies.length > 0 || librarySeries.length > 0

  // Auto-enable external search when no library results (after loading completes)
  const shouldSearchExternal = (!hasLibraryResults && !isLibraryLoading) || externalEnabled

  // Fetch external results (conditional)
  const { data: externalMovies = [], isLoading: loadingExternalMovies } = useMovieSearch(
    shouldSearchExternal && query.length >= 2 ? query : '',
  )
  const { data: externalSeries = [], isLoading: loadingExternalSeries } = useSeriesSearch(
    shouldSearchExternal && query.length >= 2 ? query : '',
  )

  const isExternalLoading = loadingExternalMovies || loadingExternalSeries
  const hasExternalResults = externalMovies.length > 0 || externalSeries.length > 0

  // Get library TMDB IDs to detect "in library" state for external results
  const libraryMovieTmdbIds = new Set(libraryMovies.map((m) => m.tmdbId))
  const librarySeriesTmdbIds = new Set(librarySeries.map((s) => s.tmdbId))

  // Fetch portal requests to show request status on external search results
  const { data: requests = [] } = useAdminRequests()

  // Build lookup maps for requests by TMDB ID
  const movieRequestsByTmdbId = useMemo(() => {
    const map = new Map<number, { id: number; status: string }>()
    for (const req of requests) {
      if (req.mediaType === 'movie' && req.tmdbId !== null) {
        map.set(req.tmdbId, { id: req.id, status: req.status })
      }
    }
    return map
  }, [requests])

  const seriesRequestsByTmdbId = useMemo(() => {
    const map = new Map<number, { id: number; status: string }>()
    for (const req of requests) {
      if ((req.mediaType === 'series' || req.mediaType === 'season') && req.tmdbId !== null) {
        // Keep the most relevant request (prefer non-available over available for badge display)
        const existing = map.get(req.tmdbId)
        if (!existing || (existing.status === 'available' && req.status !== 'available')) {
          map.set(req.tmdbId, { id: req.id, status: req.status })
        }
      }
    }
    return map
  }, [requests])

  if (!query) {
    return (
      <EmptyState
        icon={<Search className="size-8" />}
        title="Enter a search term"
        description="Use the search bar above to find movies and series"
      />
    )
  }

  return (
    <div className="space-y-8">
      {/* Library Results Section */}
      <SearchResultsSection
        title="Library"
        isLoading={isLibraryLoading}
        hasResults={hasLibraryResults}
      >
        <div className="space-y-6">
          <ExpandableMediaGrid
            items={libraryMovies}
            getKey={(movie) => movie.id}
            label="Movies"
            icon="movie"
            renderItem={(movie) => <MovieCard movie={movie} />}
          />
          <ExpandableMediaGrid
            items={librarySeries}
            getKey={(series) => series.id}
            label="Series"
            icon="series"
            renderItem={(series) => <SeriesCard series={series} />}
          />
        </div>
      </SearchResultsSection>

      {/* External Results Section */}
      <ExternalSearchSection
        query={query}
        enabled={shouldSearchExternal}
        onEnable={() => setExternalEnabled(true)}
        isLoading={isExternalLoading}
        hasResults={hasExternalResults}
      >
        <div className="space-y-6">
          <ExpandableMediaGrid
            items={externalMovies}
            getKey={(movie) => movie.tmdbId || movie.id}
            label="Movies"
            icon="movie"
            collapsible={false}
            renderItem={(movie) => (
              <ExternalMovieCard
                movie={movie}
                inLibrary={libraryMovieTmdbIds.has(movie.tmdbId)}
                requestInfo={
                  movieRequestsByTmdbId.get(movie.tmdbId) as
                    | {
                        id: number
                        status:
                          | 'pending'
                          | 'approved'
                          | 'denied'
                          | 'downloading'
                          | 'available'
                          | 'cancelled'
                      }
                    | undefined
                }
              />
            )}
          />
          <ExpandableMediaGrid
            items={externalSeries}
            getKey={(series) => series.tmdbId || series.id}
            label="Series"
            icon="series"
            collapsible={false}
            renderItem={(series) => (
              <ExternalSeriesCard
                series={series}
                inLibrary={librarySeriesTmdbIds.has(series.tmdbId)}
                requestInfo={
                  seriesRequestsByTmdbId.get(series.tmdbId) as
                    | {
                        id: number
                        status:
                          | 'pending'
                          | 'approved'
                          | 'denied'
                          | 'downloading'
                          | 'available'
                          | 'cancelled'
                      }
                    | undefined
                }
              />
            )}
          />
        </div>
      </ExternalSearchSection>
    </div>
  )
}
