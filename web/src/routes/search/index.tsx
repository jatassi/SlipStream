import { useState, useEffect } from 'react'
import { useNavigate } from '@tanstack/react-router'
import { Search, Film, Tv, Loader2, ChevronRight, ChevronDown } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/data/EmptyState'
import { MovieCard } from '@/components/movies/MovieCard'
import { SeriesCard } from '@/components/series/SeriesCard'
import { ExternalMovieCard } from '@/components/search/ExternalMovieCard'
import { ExternalSeriesCard } from '@/components/search/ExternalSeriesCard'
import { 
  useMovies, 
  useSeries, 
  useMovieSearch, 
  useSeriesSearch, 
  useTMDBSearchOrdering, 
  useUpdateTMDBSearchOrdering, 
  useDeveloperMode 
} from '@/hooks'

interface SearchPageProps {
  q: string
}

export function SearchPage({ q }: SearchPageProps) {
  const navigate = useNavigate()
  const query = q?.trim() || ''
  const [externalEnabled, setExternalEnabled] = useState(false)
  const [expandedMovies, setExpandedMovies] = useState(false)
  const [expandedSeries, setExpandedSeries] = useState(false)
  const [expandedLibraryMovies, setExpandedLibraryMovies] = useState(false)
  const [expandedLibrarySeries, setExpandedLibrarySeries] = useState(false)

  // Hooks
  const developerMode = useDeveloperMode()
  const disableSearchOrdering = useTMDBSearchOrdering()
  const updateTMDBSearchOrdering = useUpdateTMDBSearchOrdering()

  // Reset expansion state when query changes
  useEffect(() => {
    setExpandedMovies(false)
    setExpandedSeries(false)
    setExpandedLibraryMovies(false)
    setExpandedLibrarySeries(false)
    setExternalEnabled(false)
  }, [query])

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
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-2 mr-4">
              <span className="text-sm text-muted-foreground">Search Ordering:</span>
              <Button
                variant={disableSearchOrdering ? "destructive" : "default"}
                size="sm"
                onClick={() => updateTMDBSearchOrdering.mutate(!disableSearchOrdering)}
                disabled={updateTMDBSearchOrdering.isPending}
              >
                {disableSearchOrdering ? "Disabled" : "Enabled"}
              </Button>
              <span className="text-xs text-gray-500 ml-2">
                {developerMode ? 'DEV MODE' : 'API'}
              </span>
            </div>
            <Button variant="ghost" onClick={() => navigate({ to: '/' })}>
              Back to Dashboard
            </Button>
          </div>
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
                <div 
                  className="flex items-center gap-2 text-sm text-muted-foreground cursor-pointer hover:text-foreground transition-all duration-200"
                  onClick={() => setExpandedLibraryMovies(!expandedLibraryMovies)}
                >
                  <ChevronRight 
                    className={`size-4 transition-transform duration-200 ${expandedLibraryMovies ? 'rotate-90' : ''}`}
                  />
                  <span>Movies ({libraryMovies.length})</span>
                </div>
                <div className={`grid gap-4 grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 transition-all duration-300 ease-in-out ${expandedLibraryMovies ? 'grid-rows-[1fr]' : 'grid-rows-1'}`}>
                  {(expandedLibraryMovies ? libraryMovies : libraryMovies.slice(0, 3)).map((movie) => (
                    <MovieCard key={movie.id} movie={movie} />
                  ))}
                </div>
                {libraryMovies.length > 3 && (
                  <div 
                    className="flex justify-center pt-2 cursor-pointer hover:text-foreground transition-all duration-200"
                    onClick={() => setExpandedLibraryMovies(!expandedLibraryMovies)}
                  >
                    <div className="flex items-center gap-1 text-sm text-muted-foreground">
                      {expandedLibraryMovies ? (
                        <>
                          <ChevronDown className="size-4" />
                          <span>Show less</span>
                        </>
                      ) : (
                        <>
                          <ChevronDown className="size-4" />
                          <span>Show {libraryMovies.length - 3} more</span>
                        </>
                      )}
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* Library Series */}
            {librarySeries.length > 0 && (
              <div className="space-y-3">
                <div 
                  className="flex items-center gap-2 text-sm text-muted-foreground cursor-pointer hover:text-foreground transition-all duration-200"
                  onClick={() => setExpandedLibrarySeries(!expandedLibrarySeries)}
                >
                  <ChevronRight 
                    className={`size-4 transition-transform duration-200 ${expandedLibrarySeries ? 'rotate-90' : ''}`}
                  />
                  <span>Series ({librarySeries.length})</span>
                </div>
                <div className={`grid gap-4 grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 transition-all duration-300 ease-in-out ${expandedLibrarySeries ? 'grid-rows-[1fr]' : 'grid-rows-1'}`}>
                  {(expandedLibrarySeries ? librarySeries : librarySeries.slice(0, 3)).map((series) => (
                    <SeriesCard key={series.id} series={series} />
                  ))}
                </div>
                {librarySeries.length > 3 && (
                  <div 
                    className="flex justify-center pt-2 cursor-pointer hover:text-foreground transition-all duration-200"
                    onClick={() => setExpandedLibrarySeries(!expandedLibrarySeries)}
                  >
                    <div className="flex items-center gap-1 text-sm text-muted-foreground">
                      {expandedLibrarySeries ? (
                        <>
                          <ChevronDown className="size-4" />
                          <span>Show less</span>
                        </>
                      ) : (
                        <>
                          <ChevronDown className="size-4" />
                          <span>Show {librarySeries.length - 3} more</span>
                        </>
                      )}
                    </div>
                  </div>
                )}
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
                  {(expandedMovies ? externalMovies : externalMovies.slice(0, 5)).map((movie) => (
                    <ExternalMovieCard
                      key={movie.tmdbId || movie.id}
                      movie={movie}
                      inLibrary={libraryMovieTmdbIds.has(movie.tmdbId)}
                    />
                  ))}
                  {!expandedMovies && externalMovies.length > 5 && (
                    <div 
                      className="aspect-[2/3] rounded-lg border-2 border-dashed border-border bg-card/50 flex items-center justify-center cursor-pointer hover:bg-card/80 transition-colors"
                      onClick={() => setExpandedMovies(true)}
                    >
                      <div className="text-center p-4">
                        <ChevronDown className="size-6 mx-auto mb-2 text-muted-foreground" />
                        <span className="text-sm text-muted-foreground">
                          Show {externalMovies.length - 5} more...
                        </span>
                      </div>
                    </div>
                  )}
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
                  {(expandedSeries ? externalSeries : externalSeries.slice(0, 5)).map((series) => (
                    <ExternalSeriesCard
                      key={series.tmdbId || series.id}
                      series={series}
                      inLibrary={librarySeriesTmdbIds.has(series.tmdbId)}
                    />
                  ))}
                  {!expandedSeries && externalSeries.length > 5 && (
                    <div 
                      className="aspect-[2/3] rounded-lg border-2 border-dashed border-border bg-card/50 flex items-center justify-center cursor-pointer hover:bg-card/80 transition-colors"
                      onClick={() => setExpandedSeries(true)}
                    >
                      <div className="text-center p-4">
                        <ChevronDown className="size-6 mx-auto mb-2 text-muted-foreground" />
                        <span className="text-sm text-muted-foreground">
                          Show {externalSeries.length - 5} more...
                        </span>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}
      </section>
    </div>
  )
}