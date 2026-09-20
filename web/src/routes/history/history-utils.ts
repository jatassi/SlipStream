import { format, isToday, isYesterday, parseISO } from 'date-fns'

import type { HistoryEntry, HistoryEventType } from '@/types'

export type MediaFilter = 'all' | 'movie' | 'episode'

export type DayGroup = { key: string; label: string; entries: HistoryEntry[] }

function dayLabel(date: Date): string {
  if (isToday(date)) {
    return 'Today'
  }
  if (isYesterday(date)) {
    return 'Yesterday'
  }
  return format(date, 'EEEE, d MMMM yyyy')
}

/** History arrives newest first, so consecutive entries of one day form a group. */
export function groupByDay(entries: HistoryEntry[]): DayGroup[] {
  const groups: DayGroup[] = []
  for (const entry of entries) {
    const date = parseISO(entry.createdAt)
    const key = format(date, 'yyyy-MM-dd')
    const last = groups.at(-1)
    if (last?.key === key) {
      last.entries.push(entry)
    } else {
      groups.push({ key, label: dayLabel(date), entries: [entry] })
    }
  }
  return groups
}

export const DATE_PRESETS = [
  { value: 'all', label: 'All Time' },
  { value: 'today', label: 'Today' },
  { value: '7days', label: 'Last 7 Days' },
  { value: '30days', label: 'Last 30 Days' },
  { value: '90days', label: 'Last 90 Days' },
] as const

export type DatePreset = (typeof DATE_PRESETS)[number]['value']

export function getAfterDate(preset: DatePreset): string | undefined {
  if (preset === 'all') {
    return undefined
  }
  const daysMap: Record<Exclude<DatePreset, 'all'>, number> = {
    today: 0,
    '7days': 7,
    '30days': 30,
    '90days': 90,
  }
  const now = new Date()
  now.setDate(now.getDate() - daysMap[preset])
  now.setHours(0, 0, 0, 0)
  return now.toISOString()
}

function str(value: unknown): string | undefined {
  if (typeof value === 'string' && value.length > 0) {
    return value
  }
  return undefined
}

type DetailTextFn = (data: Record<string, unknown>, source: string | undefined) => string

const detailTextByEvent: Partial<Record<HistoryEventType, DetailTextFn>> = {
  autosearch_download: getAutosearchDownloadText,
  autosearch_failed: (data) => str(data.error) ?? 'Search failed',
  imported: (data, source) =>
    str(data.finalFilename) ?? str(data.originalFilename) ?? source ?? '-',
  import_failed: (data) => str(data.error) ?? 'Import failed',
  status_changed: getStatusChangedText,
  file_renamed: getFileRenamedText,
  slot_assigned: getSlotEventText,
  slot_reassigned: getSlotEventText,
  slot_unassigned: getSlotEventText,
}

export function getDetailsText(item: HistoryEntry): string {
  const data = item.data as Record<string, unknown> | undefined
  if (!data) {
    return item.source ?? '-'
  }
  const fn = detailTextByEvent[item.eventType]
  return fn ? fn(data, item.source) : (item.source ?? '-')
}

function getAutosearchDownloadText(
  data: Record<string, unknown>,
  source: string | undefined,
): string {
  const release = str(data.releaseName) ?? source ?? '-'
  if (data.isUpgrade && data.newQuality) {
    return `${release} (upgrade to ${str(data.newQuality) ?? 'unknown'})`
  }
  if (data.isUpgrade) {
    return `${release} (upgrade)`
  }
  return release
}

function getStatusChangedText(
  data: Record<string, unknown>,
  source: string | undefined,
): string {
  const from = str(data.from)
  const to = str(data.to)
  if (from && to) {
    return `${from} \u2192 ${to}`
  }
  return source ?? '-'
}

function getFileRenamedText(
  data: Record<string, unknown>,
  source: string | undefined,
): string {
  const oldName = str(data.old_filename)
  const newName = str(data.new_filename)
  if (oldName && newName) {
    return `${oldName} \u2192 ${newName}`
  }
  return source ?? '-'
}

function getSlotEventText(data: Record<string, unknown>): string {
  return str(data.slotName) ?? '-'
}


