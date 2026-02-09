import type { HistoryEventType } from '@/types'

export const eventTypeColors: Record<HistoryEventType, 'default' | 'secondary' | 'destructive' | 'outline'> = {
  grabbed: 'default',
  imported: 'secondary',
  deleted: 'destructive',
  failed: 'destructive',
  file_renamed: 'outline',
  autosearch_download: 'default',
  autosearch_failed: 'destructive',
  import_failed: 'destructive',
  status_changed: 'outline',
}

export const eventTypeLabels: Record<HistoryEventType, string> = {
  grabbed: 'Grabbed',
  imported: 'Imported',
  deleted: 'Deleted',
  failed: 'Failed',
  file_renamed: 'File Renamed',
  autosearch_download: 'Auto Download',
  autosearch_failed: 'Auto Failed',
  import_failed: 'Import Failed',
  status_changed: 'Status Changed',
}

/** Event types shown in the filter dropdown. */
export const filterableEventTypes: { value: HistoryEventType; label: string }[] = [
  { value: 'autosearch_download', label: 'Auto Download' },
  { value: 'autosearch_failed', label: 'Auto Failed' },
  { value: 'imported', label: 'Imported' },
  { value: 'import_failed', label: 'Import Failed' },
  { value: 'file_renamed', label: 'File Renamed' },
  { value: 'status_changed', label: 'Status Changed' },
]

/** Check whether a history entry represents an upgrade (from data fields). */
export function isUpgradeEvent(data: Record<string, unknown> | undefined): boolean {
  if (!data) return false
  return Boolean(data.isUpgrade)
}
