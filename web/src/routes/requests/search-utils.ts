import type {
  MovieSearchResult,
  PortalMovieSearchResult,
  PortalSeriesSearchResult,
  SeriesSearchResult,
} from '@/types'

export function convertToMovieSearchResult(movie: PortalMovieSearchResult): MovieSearchResult {
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

export function convertToSeriesSearchResult(series: PortalSeriesSearchResult): SeriesSearchResult {
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

export const sortByAddedAt = <T extends { availability?: { addedAt?: string | null } }>(
  items: T[],
): T[] =>
  items.toSorted((a, b) => {
    const aDate = a.availability?.addedAt ? new Date(a.availability.addedAt).getTime() : 0
    const bDate = b.availability?.addedAt ? new Date(b.availability.addedAt).getTime() : 0
    return bDate - aDate
  })
