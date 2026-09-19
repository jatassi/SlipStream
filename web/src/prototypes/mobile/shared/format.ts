import type { MediaKind, MediaStatus, QueueState } from './types'

export function formatGb(gb: number): string {
  if (gb === 0) {
    return '—'
  }
  if (gb >= 1000) {
    return `${(gb / 1000).toFixed(2)} TB`
  }
  return `${gb.toFixed(1)} GB`
}

export function formatSpeed(mbps: number): string {
  return `${mbps.toFixed(1)} MB/s`
}

export function formatEta(minutes: number): string {
  if (minutes <= 0) {
    return '<1m'
  }
  if (minutes < 60) {
    return `${minutes}m`
  }
  return `${Math.floor(minutes / 60)}h ${minutes % 60}m`
}

export const STATUS_LABEL: Record<MediaStatus, string> = {
  available: 'Available',
  missing: 'Missing',
  downloading: 'Downloading',
  upgradable: 'Upgradable',
  unreleased: 'Unreleased',
  failed: 'Failed',
}

export function statusColor(status: MediaStatus): string {
  return `var(--status-${status})`
}

export const QUEUE_LABEL: Record<QueueState, string> = {
  downloading: 'Downloading',
  paused: 'Paused',
  importing: 'Importing',
  queued: 'Queued',
}

export function kindColor(kind: MediaKind, shade = 500): string {
  return kind === 'movie' ? `var(--movie-${shade})` : `var(--tv-${shade})`
}

export const KIND_TEXT: Record<MediaKind, string> = {
  movie: 'text-movie-400',
  series: 'text-tv-400',
}

export const KIND_BG: Record<MediaKind, string> = {
  movie: 'bg-movie-500',
  series: 'bg-tv-500',
}

export const KIND_LABEL: Record<MediaKind, string> = {
  movie: 'Movie',
  series: 'Series',
}
