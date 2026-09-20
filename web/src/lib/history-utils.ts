import type { LucideIcon } from 'lucide-react'
import {
  AlertCircle,
  ArrowDownToLine,
  CheckCircle2,
  FileEdit,
  Layers,
  PackageCheck,
  RefreshCw,
  Search,
  XCircle,
} from 'lucide-react'

import type { HistoryEntry, HistoryEventType } from '@/types'

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

export type HistoryEventLook = { label: string; className: string; icon: LucideIcon }

const IMPORTED_LOOK = { label: 'Imported', className: 'bg-emerald-600', icon: CheckCircle2 }
const GRABBED_LOOK = { label: 'Grabbed', className: 'bg-tv-600', icon: ArrowDownToLine }
const UPGRADED_LOOK = { label: 'Upgraded', className: 'bg-movie-600', icon: RefreshCw }
const FAILED_LOOK = { label: 'Failed', className: 'bg-red-600', icon: XCircle }

/** Tile colour, glyph and short label shared by the Dashboard's Recent group and History. */
export function eventLook(entry: HistoryEntry): HistoryEventLook {
  if (isUpgradeEvent(entry.data as Record<string, unknown> | undefined)) {
    return UPGRADED_LOOK
  }
  if (entry.eventType === 'imported') {
    return IMPORTED_LOOK
  }
  if (entry.eventType === 'grabbed' || entry.eventType === 'autosearch_download') {
    return GRABBED_LOOK
  }
  if (
    entry.eventType === 'failed' ||
    entry.eventType === 'autosearch_failed' ||
    entry.eventType === 'import_failed'
  ) {
    return FAILED_LOOK
  }
  return {
    label: eventTypeLabels[entry.eventType],
    className: 'bg-zinc-600',
    icon: ArrowDownToLine,
  }
}
