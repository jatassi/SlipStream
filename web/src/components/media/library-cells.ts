import type { Movie, Series } from '@/types'

import { aggregateMediaStatus } from './media-status'
import type { PosterCellItem } from './poster-cell'

export type ProfileNames = Map<number, string>

export function movieToCell(movie: Movie, profileNames: ProfileNames): PosterCellItem {
  return {
    id: movie.id,
    title: movie.title,
    href: `/movies/${movie.id}`,
    status: movie.status,
    year: movie.year,
    quality: movie.movieFiles[0]?.quality ?? profileNames.get(movie.qualityProfileId),
    posterType: 'movie',
    tmdbId: movie.tmdbId,
    version: movie.updatedAt,
  }
}

function firstYear(series: Series): number | null {
  if (series.year !== undefined) {
    return series.year
  }
  if (series.firstAired === undefined) {
    return null
  }
  const date = new Date(series.firstAired)
  return Number.isNaN(date.getTime()) ? null : date.getFullYear()
}

export function seriesToCell(series: Series, profileNames: ProfileNames): PosterCellItem {
  return {
    id: series.id,
    title: series.title,
    href: `/series/${series.id}`,
    status: aggregateMediaStatus(series.statusCounts),
    year: firstYear(series),
    quality: profileNames.get(series.qualityProfileId),
    posterType: 'series',
    tmdbId: series.tmdbId,
    tvdbId: series.tvdbId,
    version: series.updatedAt,
  }
}
