import type { LucideIcon } from 'lucide-react'
import { AlertCircle, FileEdit, Layers, PackageCheck, RefreshCw, Search } from 'lucide-react'

import type { HistoryEventType } from '@/types'

export const eventTypeColors: Record<
  HistoryEventType,
  'default' | 'secondary' | 'destructive' | 'outline'
> = {
  grabbed: 'default',
  imported: 'secondary',
  deleted: 'destructive',
  failed: 'destructive',
  file_renamed: 'outline',
  autosearch_download: 'default',
  autosearch_failed: 'destructive',
  import_failed: 'destructive',
  slot_assigned: 'secondary',
  slot_reassigned: 'secondary',
  slot_unassigned: 'outline',
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
  slot_assigned: 'Slot Assigned',
  slot_reassigned: 'Slot Reassigned',
  slot_unassigned: 'Slot Unassigned',
  status_changed: 'Status Changed',
}

/** Event types shown in the filter dropdown. */
export const filterableEventTypes: { value: HistoryEventType; label: string; icon: LucideIcon }[] =
  [
    { value: 'autosearch_download', label: 'Auto Download', icon: Search },
    { value: 'autosearch_failed', label: 'Auto Failed', icon: AlertCircle },
    { value: 'imported', label: 'Imported', icon: PackageCheck },
    { value: 'import_failed', label: 'Import Failed', icon: AlertCircle },
    { value: 'file_renamed', label: 'File Renamed', icon: FileEdit },
    { value: 'slot_assigned', label: 'Slot Assigned', icon: Layers },
    { value: 'slot_reassigned', label: 'Slot Reassigned', icon: Layers },
    { value: 'slot_unassigned', label: 'Slot Unassigned', icon: Layers },
    { value: 'status_changed', label: 'Status Changed', icon: RefreshCw },
  ]

/** Check whether a history entry represents an upgrade (from data fields). */
export function isUpgradeEvent(data: Record<string, unknown> | undefined): boolean {
  if (!data) {
    return false
  }
  return Boolean(data.isUpgrade)
}
