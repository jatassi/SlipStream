import { useMemo, useState } from 'react'

import { useSearch } from '@tanstack/react-router'

import { movieToCell, type ProfileNames, seriesToCell } from '@/components/media/library-cells'
import type { PosterCellItem } from '@/components/media/poster-cell'
import { useMovies, useMovieSearch, useSeries, useSeriesSearch } from '@/hooks'
import type { Movie, MovieSearchResult, Series, SeriesSearchResult } from '@/types'

const NO_PROFILE_NAMES: ProfileNames = new Map()

export type ExternalResult = {
  tmdbId: number
  title: string
  year: number | null
  kind: 'movie' | 'series'
  posterUrl?: string
  inLibrary: boolean
  addTo: '/movies/add' | '/series/add'
}

function titleMatches(title: string, needle: string): boolean {
  return title.toLowerCase().includes(needle)
}

function libraryResults(movies: Movie[], series: Series[], needle: string): PosterCellItem[] {
  if (needle.length === 0) {
    return []
  }
  const cells = [
    ...movies.filter((movie) => titleMatches(movie.title, needle)).map((movie) => movieToCell(movie, NO_PROFILE_NAMES)),
    ...series.filter((entry) => titleMatches(entry.title, needle)).map((entry) => seriesToCell(entry, NO_PROFILE_NAMES)),
  ]
  return cells.toSorted((a, b) => a.title.localeCompare(b.title))
}

function tmdbIds(items: { tmdbId?: number }[]): Set<number> {
  return new Set(items.map((item) => item.tmdbId).filter((id): id is number => id !== undefined))
}

function toExternalMovie(result: MovieSearchResult, inLibrary: Set<number>): ExternalResult {
  return {
    tmdbId: result.tmdbId,
    title: result.title,
    year: result.year ?? null,
    kind: 'movie',
    posterUrl: result.posterUrl,
    inLibrary: inLibrary.has(result.tmdbId),
    addTo: '/movies/add',
  }
}

function toExternalSeries(result: SeriesSearchResult, inLibrary: Set<number>): ExternalResult {
  return {
    tmdbId: result.tmdbId,
    title: result.title,
    year: result.year ?? null,
    kind: 'series',
    posterUrl: result.posterUrl,
    inLibrary: inLibrary.has(result.tmdbId),
    addTo: '/series/add',
  }
}

export type ExternalSearch = {
  open: boolean
  reveal: () => void
  isLoading: boolean
  results: ExternalResult[]
}

function useExternalSearch(
  query: string,
  autoOpen: boolean,
  library: { movies: Movie[]; series: Series[] },
): ExternalSearch {
  const [revealed, setRevealed] = useState(false)
  const [prevQuery, setPrevQuery] = useState(query)
  if (prevQuery !== query) {
    setPrevQuery(query)
    setRevealed(false)
  }

  const open = revealed || autoOpen
  const externalQuery = open && query.length >= 2 ? query : ''
  const { data: movieResults = [], isFetching: fetchingMovies } = useMovieSearch(externalQuery)
  const { data: seriesResults = [], isFetching: fetchingSeries } = useSeriesSearch(externalQuery)

  const results = useMemo(() => {
    const libraryMovies = tmdbIds(library.movies)
    const librarySeries = tmdbIds(library.series)
    return [
      ...movieResults.map((result) => toExternalMovie(result, libraryMovies)),
      ...seriesResults.map((result) => toExternalSeries(result, librarySeries)),
    ]
  }, [movieResults, seriesResults, library])

  return {
    open,
    reveal: () => setRevealed(true),
    isLoading: fetchingMovies || fetchingSeries,
    results,
  }
}

export function useSearchPage() {
  const search = useSearch({ strict: false })
  const urlQuery = typeof search.q === 'string' ? search.q : ''
  const [text, setText] = useState(urlQuery)
  const [prevUrlQuery, setPrevUrlQuery] = useState(urlQuery)
  if (prevUrlQuery !== urlQuery) {
    setPrevUrlQuery(urlQuery)
    setText(urlQuery)
  }

  const { data: movies = [], isPending: moviesPending } = useMovies()
  const { data: series = [], isPending: seriesPending } = useSeries()
  const libraryLoading = moviesPending || seriesPending

  const query = text.trim()
  const needle = query.toLowerCase()
  const results = useMemo(() => libraryResults(movies, series, needle), [movies, series, needle])
  const library = useMemo(() => ({ movies, series }), [movies, series])
  const external = useExternalSearch(query, !libraryLoading && results.length === 0, library)

  return {
    text,
    setText,
    query,
    libraryLoading,
    searchableCount: movies.length + series.length,
    results,
    external,
  }
}
