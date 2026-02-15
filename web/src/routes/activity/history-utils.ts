import type { HistoryEntry, HistoryEventType } from '@/types'

export type MediaFilter = 'all' | 'movie' | 'episode'

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

export type DetailRow = { label: string; value: string }

export function getDetailRows(item: HistoryEntry): DetailRow[] {
  const data = item.data as Record<string, unknown> | undefined
  if (!data) {
    return []
  }

  switch (item.eventType) {
    case 'autosearch_download': {
      return getAutosearchDownloadRows(data)
    }
    case 'autosearch_failed': {
      return getAutosearchFailedRows(data)
    }
    case 'imported': {
      return getImportedRows(data)
    }
    case 'import_failed': {
      return getImportFailedRows(data)
    }
    case 'status_changed': {
      return getStatusChangedRows(data)
    }
    case 'file_renamed': {
      return getFileRenamedRows(data)
    }
    default: {
      return []
    }
  }
}

function pushIfPresent(rows: DetailRow[], label: string, value: unknown) {
  const s = str(value)
  if (s) {
    rows.push({ label, value: s })
  }
}

function getAutosearchDownloadRows(data: Record<string, unknown>): DetailRow[] {
  const rows: DetailRow[] = []
  pushIfPresent(rows, 'Release', data.releaseName)
  pushIfPresent(rows, 'Indexer', data.indexer)
  pushIfPresent(rows, 'Client', data.clientName)
  pushIfPresent(rows, 'Download ID', data.downloadId)
  pushIfPresent(rows, 'Trigger', data.source)
  pushIfPresent(rows, 'Previous Quality', data.oldQuality)
  pushIfPresent(rows, 'New Quality', data.newQuality)
  return rows
}

function getAutosearchFailedRows(data: Record<string, unknown>): DetailRow[] {
  const rows: DetailRow[] = []
  pushIfPresent(rows, 'Error', data.error)
  pushIfPresent(rows, 'Indexer', data.indexer)
  return rows
}

function getImportedRows(data: Record<string, unknown>): DetailRow[] {
  const rows: DetailRow[] = []
  pushIfPresent(rows, 'Source', data.sourcePath)
  pushIfPresent(rows, 'Destination', data.destinationPath)
  pushIfPresent(rows, 'Original', data.originalFilename)
  pushIfPresent(rows, 'Final', data.finalFilename)
  pushIfPresent(rows, 'Client', data.clientName)
  pushIfPresent(rows, 'Codec', data.codec)
  if (typeof data.size === 'number' && data.size > 0) {
    rows.push({ label: 'Size', value: formatFileSize(data.size) })
  }
  pushIfPresent(rows, 'Previous File', data.previousFile)
  pushIfPresent(rows, 'Error', data.error)
  return rows
}

function getImportFailedRows(data: Record<string, unknown>): DetailRow[] {
  const rows: DetailRow[] = []
  pushIfPresent(rows, 'Error', data.error)
  pushIfPresent(rows, 'Source', data.sourcePath)
  return rows
}

function getStatusChangedRows(data: Record<string, unknown>): DetailRow[] {
  const rows: DetailRow[] = []
  pushIfPresent(rows, 'From', data.from)
  pushIfPresent(rows, 'To', data.to)
  pushIfPresent(rows, 'Reason', data.reason)
  return rows
}

function getFileRenamedRows(data: Record<string, unknown>): DetailRow[] {
  const rows: DetailRow[] = []
  pushIfPresent(rows, 'Old Path', data.source_path)
  pushIfPresent(rows, 'New Path', data.destination_path)
  return rows
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) {
    return '0 B'
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

export type PaginationEntry = number | 'ellipsis-start' | 'ellipsis-end'

export function getPaginationPages(current: number, total: number): PaginationEntry[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1)
  }

  const pages: PaginationEntry[] = [1]

  if (current > 3) {
    pages.push('ellipsis-start')
  }

  const start = Math.max(2, current - 1)
  const end = Math.min(total - 1, current + 1)
  for (let i = start; i <= end; i++) {
    pages.push(i)
  }

  if (current < total - 2) {
    pages.push('ellipsis-end')
  }

  pages.push(total)
  return pages
}
