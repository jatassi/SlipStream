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
