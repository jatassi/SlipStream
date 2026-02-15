/**
 * Format bytes to human-readable file size
 */
export function formatBytes(bytes: number, decimals = 2): string {
  if (bytes === 0) {
    return '0 B'
  }

  const k = 1024
  const dm = Math.max(decimals, 0)
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']

  const i = Math.floor(Math.log(bytes) / Math.log(k))

  return `${Number.parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`
}

/**
 * Format duration in minutes to human-readable string
 */
export function formatRuntime(minutes: number): string {
  if (!minutes) {
    return ''
  }
  const hours = Math.floor(minutes / 60)
  const mins = minutes % 60
  if (hours === 0) {
    return `${mins}m`
  }
  if (mins === 0) {
    return `${hours}h`
  }
  return `${hours}h ${mins}m`
}

function pluralize(count: number, singular: string): string {
  return `${count} ${singular}${count > 1 ? 's' : ''} ago`
}

/**
 * Format date to relative time (e.g., "2 days ago")
 */
export function formatRelativeTime(date: string | Date): string {
  const now = new Date()
  const then = new Date(date)
  if (Number.isNaN(then.getTime()) || then.getFullYear() < 1970) {
    return '-'
  }
  const diff = now.getTime() - then.getTime()

  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  const weeks = Math.floor(days / 7)
  const months = Math.floor(days / 30)
  const years = Math.floor(days / 365)

  if (years > 0) {return pluralize(years, 'year')}
  if (months > 0) {return pluralize(months, 'month')}
  if (weeks > 0) {return pluralize(weeks, 'week')}
  if (days > 0) {return pluralize(days, 'day')}
  if (hours > 0) {return pluralize(hours, 'hour')}
  if (minutes > 0) {return pluralize(minutes, 'minute')}
  return 'Just now'
}

/**
 * Format date to localized string
 */
export function formatDate(date: string | Date, options?: Intl.DateTimeFormatOptions): string {
  const d = new Date(date)
  return d.toLocaleDateString(
    undefined,
    options ?? {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    },
  )
}

/**
 * Format date and time
 */
export function formatDateTime(date: string | Date): string {
  const d = new Date(date)
  return d.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/**
 * Format download speed
 */
export function formatSpeed(bytesPerSecond: number): string {
  if (bytesPerSecond === 0) {
    return '0 B/s'
  }
  return `${formatBytes(bytesPerSecond)}/s`
}

/**
 * Format ETA from seconds
 */
export function formatEta(seconds: number): string {
  if (!seconds || seconds <= 0) {
    return '--'
  }

  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)

  if (hours > 0) {
    return `${hours}h ${minutes}m`
  }
  if (minutes > 0) {
    return `${minutes}m ${secs}s`
  }
  return `${secs}s`
}

/**
 * Format percentage
 */
export function formatPercent(value: number, decimals = 1): string {
  return `${value.toFixed(decimals)}%`
}

/**
 * Format episode number (e.g., S01E05)
 */
export function formatEpisodeNumber(season: number, episode: number): string {
  return `S${String(season).padStart(2, '0')}E${String(episode).padStart(2, '0')}`
}

/**
 * Format series title with season/episode info
 */
export function formatSeriesTitle(seriesName: string, season?: number, episode?: number): string {
  if (!season || !episode) {
    return seriesName
  }
  return `${seriesName} - ${formatEpisodeNumber(season, episode)}`
}

/**
 * Format status counts into a summary string for series/season display
 */
export function formatStatusSummary(counts: {
  available: number
  upgradable: number
  unreleased: number
  missing: number
  downloading: number
  failed: number
  total: number
}): string {
  const ready = counts.available + counts.upgradable
  if (ready === counts.total && counts.total > 0) {
    return 'Available'
  }
  if (counts.unreleased === counts.total) {
    return 'Unreleased'
  }
  if (counts.failed > 0) {
    return `${ready}/${counts.total} eps (${counts.failed} failed)`
  }
  if (counts.downloading > 0) {
    return `${ready}/${counts.total} eps (${counts.downloading} downloading)`
  }
  return `${ready}/${counts.total} eps`
}
