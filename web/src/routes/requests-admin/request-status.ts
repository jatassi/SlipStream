import type { MediaStatus } from '@/components/media/media-status'
import type { SegmentedOption } from '@/components/ui/segmented'
import { getStatusConfig } from '@/lib/request-status-config'
import type { RequestStatus } from '@/types'

export type QueueSegment = 'pending' | 'approved' | 'downloading' | 'available' | 'denied'

export const QUEUE_SEGMENTS: SegmentedOption<QueueSegment>[] = [
  { value: 'pending', label: 'Pending' },
  { value: 'approved', label: 'Approved' },
  { value: 'downloading', label: 'Downloading' },
  { value: 'available', label: 'Available' },
  { value: 'denied', label: 'Denied' },
]

const SEGMENT_STATUSES: Record<QueueSegment, RequestStatus[]> = {
  pending: ['pending'],
  approved: ['approved', 'searching'],
  downloading: ['downloading', 'failed'],
  available: ['available'],
  denied: ['denied', 'cancelled'],
}

export function segmentOf(status: RequestStatus): QueueSegment | undefined {
  return QUEUE_SEGMENTS.map((option) => option.value).find((segment) =>
    SEGMENT_STATUSES[segment].includes(status),
  )
}

const STATUS_CONFIG = getStatusConfig()

export const REQUEST_STATUS_LABEL: Record<RequestStatus, string> = Object.fromEntries(
  (Object.keys(STATUS_CONFIG) as RequestStatus[]).map((status) => [status, STATUS_CONFIG[status].label]),
) as Record<RequestStatus, string>

const REQUEST_STATUS_TONE: Record<RequestStatus, MediaStatus> = {
  pending: 'missing',
  approved: 'upgradable',
  searching: 'upgradable',
  downloading: 'downloading',
  available: 'available',
  denied: 'failed',
  failed: 'failed',
  cancelled: 'unreleased',
}

export function requestStatusTone(status: RequestStatus): MediaStatus {
  return REQUEST_STATUS_TONE[status]
}
