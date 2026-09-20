export type MediaStatus =
  | 'available'
  | 'missing'
  | 'downloading'
  | 'upgradable'
  | 'unreleased'
  | 'failed'

export const MEDIA_STATUS_LABEL: Record<MediaStatus, string> = {
  available: 'Available',
  missing: 'Missing',
  downloading: 'Downloading',
  upgradable: 'Upgradable',
  unreleased: 'Unreleased',
  failed: 'Failed',
}

export function mediaStatusColor(status: MediaStatus): string {
  return `var(--status-${status})`
}

type StatusCountLike = {
  unreleased: number
  missing: number
  downloading: number
  failed: number
  upgradable: number
  available: number
}

const AGGREGATE_PRIORITY: MediaStatus[] = [
  'downloading',
  'failed',
  'missing',
  'upgradable',
  'available',
]

export function aggregateMediaStatus(counts: StatusCountLike): MediaStatus {
  return AGGREGATE_PRIORITY.find((status) => counts[status] > 0) ?? 'unreleased'
}
