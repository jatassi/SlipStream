import {
  addMonths,
  addWeeks,
  endOfWeek,
  format,
  getDay,
  isPast,
  isSameMonth,
  isThisMonth,
  isThisWeek,
  isToday,
  isTomorrow,
  isYesterday,
  startOfWeek,
} from 'date-fns'

export type MediaGroup<T> = {
  key: string
  label: string
  items: T[]
}

export type GroupingContext = {
  qualityProfileNames: Map<number, string>
  rootFolderNames: Map<number, string>
}

type Groupable = {
  monitored: boolean
  qualityProfileId: number
  rootFolderId?: number
  nextAiring?: string
  releaseDate?: string
  addedAt: string
  sizeOnDisk?: number
}

type SortField =
  | 'title'
  | 'monitored'
  | 'qualityProfile'
  | 'nextAirDate'
  | 'releaseDate'
  | 'dateAdded'
  | 'rootFolder'
  | 'sizeOnDisk'

const WEEKDAY_NAMES = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']

function getRelativeFutureDateGroup(dateStr: string | undefined): { key: string; label: string } {
  if (!dateStr) {
    return { key: '_tba', label: 'TBA' }
  }

  const date = new Date(dateStr)
  const now = new Date()

  if (isPast(date) && !isToday(date)) {
    return { key: '_aired', label: 'Aired' }
  }
  if (isToday(date)) {
    return { key: '_today', label: 'Today' }
  }
  if (isTomorrow(date)) {
    return { key: '_tomorrow', label: 'Tomorrow' }
  }

  const endOfThisWeek = endOfWeek(now, { weekStartsOn: getDay(startOfWeek(now)) as 0 })
  if (date <= endOfThisWeek) {
    const dayName = WEEKDAY_NAMES[getDay(date)]
    return { key: `_day_${getDay(date)}`, label: dayName }
  }

  const nextWeekEnd = endOfWeek(addWeeks(now, 1))
  if (date <= nextWeekEnd) {
    return { key: '_next_week', label: 'Next Week' }
  }

  if (isThisMonth(date)) {
    return { key: '_this_month', label: 'Later This Month' }
  }

  const nextMonth = addMonths(now, 1)
  if (isSameMonth(date, nextMonth)) {
    return { key: '_next_month', label: 'Next Month' }
  }

  const label = format(date, 'MMMM yyyy')
  return { key: label, label }
}

function getRelativePastDateGroup(dateStr: string): { key: string; label: string } {
  const date = new Date(dateStr)
  const now = new Date()

  if (isToday(date)) {
    return { key: '_today', label: 'Today' }
  }
  if (isYesterday(date)) {
    return { key: '_yesterday', label: 'Yesterday' }
  }

  if (isThisWeek(date)) {
    const dayName = WEEKDAY_NAMES[getDay(date)]
    return { key: `_day_${getDay(date)}`, label: dayName }
  }

  const lastWeekStart = startOfWeek(addWeeks(now, -1))
  const lastWeekEnd = endOfWeek(addWeeks(now, -1))
  if (date >= lastWeekStart && date <= lastWeekEnd) {
    return { key: '_last_week', label: 'Last Week' }
  }

  if (isThisMonth(date)) {
    return { key: '_this_month', label: 'Earlier This Month' }
  }

  const lastMonth = addMonths(now, -1)
  if (isSameMonth(date, lastMonth)) {
    return { key: '_last_month', label: 'Last Month' }
  }

  const label = format(date, 'MMMM yyyy')
  return { key: label, label }
}

function getSizeGroup(bytes: number | undefined): { key: string; label: string } {
  if (!bytes || bytes === 0) {
    return { key: '0', label: 'No Files' }
  }
  const gb = bytes / (1024 * 1024 * 1024)
  if (gb < 1) {
    return { key: '1', label: 'Under 1 GB' }
  }
  if (gb < 5) {
    return { key: '2', label: '1 \u2013 5 GB' }
  }
  if (gb < 10) {
    return { key: '3', label: '5 \u2013 10 GB' }
  }
  if (gb < 25) {
    return { key: '4', label: '10 \u2013 25 GB' }
  }
  if (gb < 50) {
    return { key: '5', label: '25 \u2013 50 GB' }
  }
  if (gb < 100) {
    return { key: '6', label: '50 \u2013 100 GB' }
  }
  return { key: '7', label: 'Over 100 GB' }
}

function getGroupKey<T extends Groupable>(
  item: T,
  sortField: SortField,
  context: GroupingContext,
): { key: string; label: string } {
  switch (sortField) {
    case 'monitored': {
      return item.monitored
        ? { key: 'monitored', label: 'Monitored' }
        : { key: 'unmonitored', label: 'Not Monitored' }
    }
    case 'qualityProfile': {
      const name = context.qualityProfileNames.get(item.qualityProfileId) || 'Unknown'
      return { key: String(item.qualityProfileId), label: name }
    }
    case 'rootFolder': {
      const name = context.rootFolderNames.get(item.rootFolderId || 0) || 'Unknown'
      return { key: String(item.rootFolderId || 0), label: name }
    }
    case 'nextAirDate': {
      return getRelativeFutureDateGroup(item.nextAiring)
    }
    case 'releaseDate': {
      return getRelativePastDateGroup(item.releaseDate || item.addedAt)
    }
    case 'dateAdded': {
      return getRelativePastDateGroup(item.addedAt)
    }
    case 'sizeOnDisk': {
      return getSizeGroup(item.sizeOnDisk)
    }
    default: {
      return { key: '', label: '' }
    }
  }
}

export function groupMedia<T extends Groupable>(
  items: T[],
  sortField: SortField,
  context: GroupingContext,
): MediaGroup<T>[] | null {
  if (sortField === 'title' || items.length === 0) {
    return null
  }

  const groups: MediaGroup<T>[] = []
  let currentGroup: MediaGroup<T> | null = null

  for (const item of items) {
    const { key, label } = getGroupKey(item, sortField, context)
    if (currentGroup?.key !== key) {
      currentGroup = { key, label, items: [] }
      groups.push(currentGroup)
    }
    currentGroup.items.push(item)
  }

  return groups
}
