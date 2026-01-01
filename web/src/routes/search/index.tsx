import { useState, useEffect } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Search, Film, Tv, Loader2 } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/data/EmptyState'
import { MovieCard } from '@/components/movies/MovieCard'
import { SeriesCard } from '@/components/series/SeriesCard'
import { ExternalMovieCard } from '@/components/search/ExternalMovieCard'
import { ExternalSeriesCard } from '@/components/search/ExternalSeriesCard'
import { useMovies, useSeries, useMovieSearch, useSeriesSearch } from '@/hooks'

interface SearchPageProps {
  q: string
}

export function SearchPage({ q }: SearchPageProps) {
  const navigate = useNavigate()
  const query = q?.trim() || ''
  const [externalEnabled, setExternalEnabled] = useState(false)

  // Fetch library results
  const { data: libraryMovies = [], isLoading: loadingLibraryMovies } = useMovies(
    query ? { search: query } : undefined
  )
  const { data: librarySeries = [], isLoading: loadingLibrarySeries } = useSeries(
    query ? { search: query } : undefined
  )

  const isLibraryLoading = loadingLibraryMovies || loadingLibrarySeries
  const hasLibraryResults = libraryMovies.length > 0 || librarySeries.length > 0

  // Auto-enable external search when no library results (after loading completes)
  const shouldSearchExternal = (!hasLibraryResults && !isLibraryLoading) || externalEnabled

  // Fetch external results (conditional)
  const { data: externalMovies = [], isLoading: loadingExternalMovies } = useMovieSearch(
    shouldSearchExternal && query.length >= 2 ? query : ''
  )
  const { data: externalSeries = [], isLoading: loadingExternalSeries } = useSeriesSearch(
    shouldSearchExternal && query.length >= 2 ? query : ''
  )

  const isExternalLoading = loadingExternalMovies || loadingExternalSeries
  const hasExternalResults = externalMovies.length > 0 || externalSeries.length > 0

  // Get library TMDB IDs to detect "in library" state for external results
  const libraryMovieTmdbIds = new Set(libraryMovies.map((m) => m.tmdbId))
  const librarySeriesTmdbIds = new Set(librarySeries.map((s) => s.tmdbId))

  // Reset external state when query changes
  useEffect(() => {
    setExternalEnabled(false)
  }, [query])

  if (!query) {
    return (
      <div>
        <PageHeader title="Search" />
        <EmptyState
          icon={<Search className="size-8" />}
          title="Enter a search term"
          description="Use the search bar above to find movies and series"
        />
      </div>
    )
  }

  return (
    <div className="space-y-8">
      <PageHeader
        title={`Search results for "${query}"`}
        actions={
          <Button variant="ghost" onClick={() => navigate({ to: '/' })}>
            Back to Dashboard
          </Button>
        }
      />

      {/* Library Results Section */}
      <section className="space-y-4">
        <div className="flex items-center gap-2">
          <h2 className="text-lg font-semibold">Your Library</h2>
          {isLibraryLoading && <Loader2 className="size-4 animate-spin text-muted-foreground" />}
        </div>

        {isLibraryLoading ? (
          <div className="grid gap-4 grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
            {Array.from({ length: 6 }).map((_, i) => (
              <div
                key={i}
                className="aspect-[2/3] rounded-lg bg-muted animate-pulse"
              />
            ))}
          </div>
        ) : !hasLibraryResults ? (
          <div className="rounded-lg border border-border bg-card p-6 text-center text-muted-foreground">
            No movies or series matching "{query}" in your library
          </div>
        ) : (
          <div className="space-y-6">
            {/* Library Movies */}
            {libraryMovies.length > 0 && (
              <div className="space-y-3">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Film className="size-4" />
                  <span>Movies ({libraryMovies.length})</span>
                </div>
                <div className="grid gap-4 grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
                  {libraryMovies.map((movie) => (
                    <MovieCard key={movie.id} movie={movie} />
                  ))}
                </div>
              </div>
            )}

            {/* Library Series */}
            {librarySeries.length > 0 && (
              <div className="space-y-3">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Tv className="size-4" />
                  <span>Series ({librarySeries.length})</span>
                </div>
                <div className="grid gap-4 grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
                  {librarySeries.map((series) => (
                    <SeriesCard key={series.id} series={series} />
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </section>

      {/* External Results Section */}
      <section className="space-y-4">
        <div className="flex items-center gap-2">
          <h2 className="text-lg font-semibold">Add New</h2>
          {isExternalLoading && <Loader2 className="size-4 animate-spin text-muted-foreground" />}
        </div>

        {!shouldSearchExternal ? (
          <div className="rounded-lg border border-dashed border-border bg-card/50 p-6 text-center">
            <p className="text-muted-foreground mb-4">
              Want to add something new to your library?
            </p>
            <Button onClick={() => setExternalEnabled(true)}>
              <Search className="size-4 mr-2" />
              Search TMDB for "{query}"
            </Button>
          </div>
        ) : isExternalLoading ? (
          <div className="grid gap-4 grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
            {Array.from({ length: 6 }).map((_, i) => (
              <div
                key={i}
                className="aspect-[2/3] rounded-lg bg-muted animate-pulse"
              />
            ))}
          </div>
        ) : !hasExternalResults ? (
          <div className="rounded-lg border border-border bg-card p-6 text-center text-muted-foreground">
            No external results found for "{query}"
          </div>
        ) : (
          <div className="space-y-6">
            {/* External Movies */}
            {externalMovies.length > 0 && (
              <div className="space-y-3">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Film className="size-4" />
                  <span>Movies from TMDB ({externalMovies.length})</span>
                </div>
                <div className="grid gap-4 grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
                  {externalMovies.map((movie) => (
                    <ExternalMovieCard
                      key={movie.tmdbId || movie.id}
                      movie={movie}
                      inLibrary={libraryMovieTmdbIds.has(movie.tmdbId)}
                    />
                  ))}
                </div>
              </div>
            )}

            {/* External Series */}
            {externalSeries.length > 0 && (
              <div className="space-y-3">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Tv className="size-4" />
                  <span>Series from TMDB ({externalSeries.length})</span>
                </div>
                <div className="grid gap-4 grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
                  {externalSeries.map((series) => (
                    <ExternalSeriesCard
                      key={series.tmdbId || series.id}
                      series={series}
                      inLibrary={librarySeriesTmdbIds.has(series.tmdbId)}
                    />
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </section>
    </div>
  )
}
