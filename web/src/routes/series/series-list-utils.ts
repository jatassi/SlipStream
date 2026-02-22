import { DEFAULT_SORT_DIRECTIONS } from '@/lib/table-columns'
import type { Series } from '@/types'

import type { FilterStatus, SortField } from './use-series-list'

export function filterSeries(
  series: Series[],
  statusFilters: FilterStatus[],
  allFiltersSelected: boolean,
): Series[] {
  if (allFiltersSelected) {
    return series
  }
  return series.filter((s) => {
    if (statusFilters.includes('monitored') && s.monitored) {
      return true
    }
    if (statusFilters.includes(s.productionStatus as FilterStatus)) {
      return true
    }
    const statusKeys = [
      'unreleased',
      'missing',
      'downloading',
      'failed',
      'upgradable',
      'available',
    ] as const
    return statusKeys.some((k) => statusFilters.includes(k) && s.statusCounts[k] > 0)
  })
}

type SortConfig = {
  sortField: SortField
  sortDirection: 'asc' | 'desc'
  profileNameMap: Map<number, string>
}

export function sortSeries(series: Series[], config: SortConfig): Series[] {
  const defaultDir = DEFAULT_SORT_DIRECTIONS[config.sortField]
  const dirMultiplier = config.sortDirection === defaultDir ? 1 : -1

  return series.toSorted((a, b) => compareSeries(a, b, config) * dirMultiplier)
}

function compareByTitle(a: Series, b: Series): number {
  return a.sortTitle.localeCompare(b.sortTitle)
}

function compareByMonitored(a: Series, b: Series): number {
  return (b.monitored ? 1 : 0) - (a.monitored ? 1 : 0) || compareByTitle(a, b)
}

function compareByQualityProfile(a: Series, b: Series, profileNameMap: Map<number, string>): number {
  const nameA = profileNameMap.get(a.qualityProfileId) ?? ''
  const nameB = profileNameMap.get(b.qualityProfileId) ?? ''
  return nameA.localeCompare(nameB) || compareByTitle(a, b)
}

function compareByNextAirDate(a: Series, b: Series): number {
  if (!a.nextAiring && !b.nextAiring) {
    return compareByTitle(a, b)
  }
  if (!a.nextAiring) {
    return 1
  }
  if (!b.nextAiring) {
    return -1
  }
  return new Date(a.nextAiring).getTime() - new Date(b.nextAiring).getTime()
}

function compareByDateAdded(a: Series, b: Series): number {
  return new Date(b.addedAt).getTime() - new Date(a.addedAt).getTime()
}

function compareByRootFolder(a: Series, b: Series): number {
  return a.rootFolderId - b.rootFolderId || compareByTitle(a, b)
}

function compareBySizeOnDisk(a: Series, b: Series): number {
  return b.sizeOnDisk - a.sizeOnDisk
}

type Comparator = (a: Series, b: Series, config: SortConfig) => number

const COMPARATORS: Record<SortField, Comparator> = {
  title: (a, b) => compareByTitle(a, b),
  monitored: (a, b) => compareByMonitored(a, b),
  qualityProfile: (a, b, c) => compareByQualityProfile(a, b, c.profileNameMap),
  nextAirDate: (a, b) => compareByNextAirDate(a, b),
  dateAdded: (a, b) => compareByDateAdded(a, b),
  rootFolder: (a, b) => compareByRootFolder(a, b),
  sizeOnDisk: (a, b) => compareBySizeOnDisk(a, b),
}

function compareSeries(a: Series, b: Series, config: SortConfig): number {
  return COMPARATORS[config.sortField](a, b, config)
}
