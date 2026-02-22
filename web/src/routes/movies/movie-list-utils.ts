import { DEFAULT_SORT_DIRECTIONS } from '@/lib/table-columns'
import type { Movie } from '@/types'

import type { FilterStatus, SortField } from './use-movie-list'

export function filterMovies(
  movies: Movie[],
  statusFilters: FilterStatus[],
  allFiltersSelected: boolean,
): Movie[] {
  if (allFiltersSelected) {
    return movies
  }
  return movies.filter((movie) => {
    if (statusFilters.includes('monitored') && movie.monitored) {
      return true
    }
    return statusFilters.includes(movie.status as FilterStatus)
  })
}

type SortConfig = {
  sortField: SortField
  sortDirection: 'asc' | 'desc'
  profileNameMap: Map<number, string>
}

export function sortMovies(movies: Movie[], config: SortConfig): Movie[] {
  const defaultDir = DEFAULT_SORT_DIRECTIONS[config.sortField]
  const dirMultiplier = config.sortDirection === defaultDir ? 1 : -1

  return movies.toSorted((a, b) => compareMovies(a, b, config) * dirMultiplier)
}

function compareByTitle(a: Movie, b: Movie): number {
  return a.sortTitle.localeCompare(b.sortTitle)
}

function compareByMonitored(a: Movie, b: Movie): number {
  return (b.monitored ? 1 : 0) - (a.monitored ? 1 : 0) || compareByTitle(a, b)
}

function compareByQualityProfile(a: Movie, b: Movie, profileNameMap: Map<number, string>): number {
  const nameA = profileNameMap.get(a.qualityProfileId) ?? ''
  const nameB = profileNameMap.get(b.qualityProfileId) ?? ''
  return nameA.localeCompare(nameB) || compareByTitle(a, b)
}

function compareByReleaseDate(a: Movie, b: Movie): number {
  const dateA = a.releaseDate ?? a.physicalReleaseDate ?? a.theatricalReleaseDate
  const dateB = b.releaseDate ?? b.physicalReleaseDate ?? b.theatricalReleaseDate
  if (!dateA && !dateB) {
    return compareByTitle(a, b)
  }
  if (!dateA) {
    return 1
  }
  if (!dateB) {
    return -1
  }
  return new Date(dateB).getTime() - new Date(dateA).getTime()
}

function compareByDateAdded(a: Movie, b: Movie): number {
  return new Date(b.addedAt).getTime() - new Date(a.addedAt).getTime()
}

function compareByRootFolder(a: Movie, b: Movie): number {
  return a.rootFolderId - b.rootFolderId || compareByTitle(a, b)
}

function compareBySizeOnDisk(a: Movie, b: Movie): number {
  return b.sizeOnDisk - a.sizeOnDisk
}

type Comparator = (a: Movie, b: Movie, config: SortConfig) => number

const COMPARATORS: Record<SortField, Comparator> = {
  title: (a, b) => compareByTitle(a, b),
  monitored: (a, b) => compareByMonitored(a, b),
  qualityProfile: (a, b, c) => compareByQualityProfile(a, b, c.profileNameMap),
  releaseDate: (a, b) => compareByReleaseDate(a, b),
  dateAdded: (a, b) => compareByDateAdded(a, b),
  rootFolder: (a, b) => compareByRootFolder(a, b),
  sizeOnDisk: (a, b) => compareBySizeOnDisk(a, b),
}

function compareMovies(a: Movie, b: Movie, config: SortConfig): number {
  return COMPARATORS[config.sortField](a, b, config)
}
