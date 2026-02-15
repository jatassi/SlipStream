import { useMemo, useState } from 'react'

import { Search } from 'lucide-react'

import { EmptyState } from '@/components/data/empty-state'
import { MovieCard } from '@/components/movies/movie-card'
import {
  ExpandableMediaGrid,
  ExternalMovieCard,
  ExternalSearchSection,
  ExternalSeriesCard,
  SearchResultsSection,
} from '@/components/search'
import { SeriesCard } from '@/components/series/series-card'
import { useMovies, useMovieSearch, useSeries, useSeriesSearch } from '@/hooks'
import { useAdminRequests } from '@/hooks/admin/use-admin-requests'

type RequestInfo = { id: number; status: 'pending' | 'approved' | 'denied' | 'downloading' | 'available' | 'cancelled' } | undefined

type RequestEntry = { id: number; status: string }

function buildMovieRequestMap(requests: { mediaType: string; tmdbId: number | null; id: number; status: string }[]) {
  const map = new Map<number, RequestEntry>()
  for (const req of requests) {
    if (req.mediaType === 'movie' && req.tmdbId !== null) {
      map.set(req.tmdbId, { id: req.id, status: req.status })
    }
  }
  return map
}

function buildSeriesRequestMap(requests: { mediaType: string; tmdbId: number | null; id: number; status: string }[]) {
  const map = new Map<number, RequestEntry>()
  for (const req of requests) {
    if ((req.mediaType === 'series' || req.mediaType === 'season') && req.tmdbId !== null) {
      const existing = map.get(req.tmdbId)
      if (!existing || (existing.status === 'available' && req.status !== 'available')) {
        map.set(req.tmdbId, { id: req.id, status: req.status })
      }
    }
  }
  return map
}

type SearchPageProps = {
  q: string
}

function useLibrarySearch(query: string) {
  const searchFilter = query ? { search: query } : undefined
  const { data: movies = [], isLoading: loadingMovies } = useMovies(searchFilter)
  const { data: series = [], isLoading: loadingSeries } = useSeries(searchFilter)
  return {
    movies, series,
    isLoading: loadingMovies || loadingSeries,
    hasResults: movies.length > 0 || series.length > 0,
  }
}

function deriveExternalQuery(query: string, shouldSearch: boolean): string {
  return shouldSearch && query.length >= 2 ? query : ''
}

function useExternalSearch(query: string, library: ReturnType<typeof useLibrarySearch>) {
  const [externalEnabled, setExternalEnabled] = useState(false)

  const [prevQuery, setPrevQuery] = useState(query)
  if (prevQuery !== query) {
    setPrevQuery(query)
    setExternalEnabled(false)
  }

  const autoEnable = !library.hasResults && !library.isLoading
  const shouldSearch = autoEnable || externalEnabled
  const externalQuery = deriveExternalQuery(query, shouldSearch)
  const { data: movies = [], isLoading: loadingMovies } = useMovieSearch(externalQuery)
  const { data: series = [], isLoading: loadingSeries } = useSeriesSearch(externalQuery)

  const { data: requests = [] } = useAdminRequests()
  const movieRequests = useMemo(() => buildMovieRequestMap(requests), [requests])
  const seriesRequests = useMemo(() => buildSeriesRequestMap(requests), [requests])

  return {
    movies, series, shouldSearch, setExternalEnabled,
    isLoading: loadingMovies || loadingSeries,
    hasResults: movies.length > 0 || series.length > 0,
    movieRequests, seriesRequests,
  }
}

export function SearchPage({ q }: SearchPageProps) {
  const query = q.trim() || ''
  const library = useLibrarySearch(query)
  const external = useExternalSearch(query, library)

  const libraryMovieTmdbIds = new Set(library.movies.map((m) => m.tmdbId))
  const librarySeriesTmdbIds = new Set(library.series.map((s) => s.tmdbId))

  if (!query) {
    return <EmptyState icon={<Search className="size-8" />} title="Enter a search term" description="Use the search bar above to find movies and series" />
  }

  return (
    <div className="space-y-8">
      <SearchResultsSection title="Library" isLoading={library.isLoading} hasResults={library.hasResults}>
        <div className="space-y-6">
          <ExpandableMediaGrid items={library.movies} getKey={(m) => m.id} label="Movies" icon="movie" renderItem={(m) => <MovieCard movie={m} />} />
          <ExpandableMediaGrid items={library.series} getKey={(s) => s.id} label="Series" icon="series" renderItem={(s) => <SeriesCard series={s} />} />
        </div>
      </SearchResultsSection>

      <ExternalSearchSection
        query={query}
        enabled={external.shouldSearch}
        onEnable={() => external.setExternalEnabled(true)}
        isLoading={external.isLoading}
        hasResults={external.hasResults}
      >
        <div className="space-y-6">
          <ExpandableMediaGrid
            items={external.movies}
            getKey={(m) => m.tmdbId || m.id}
            label="Movies"
            icon="movie"
            collapsible={false}
            renderItem={(m) => (
              <ExternalMovieCard movie={m} inLibrary={libraryMovieTmdbIds.has(m.tmdbId)} requestInfo={external.movieRequests.get(m.tmdbId) as RequestInfo} />
            )}
          />
          <ExpandableMediaGrid
            items={external.series}
            getKey={(s) => s.tmdbId || s.id}
            label="Series"
            icon="series"
            collapsible={false}
            renderItem={(s) => (
              <ExternalSeriesCard series={s} inLibrary={librarySeriesTmdbIds.has(s.tmdbId)} requestInfo={external.seriesRequests.get(s.tmdbId) as RequestInfo} />
            )}
          />
        </div>
      </ExternalSearchSection>
    </div>
  )
}
