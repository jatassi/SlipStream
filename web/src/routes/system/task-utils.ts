function formatFutureTime(diffMins: number, diffHours: number, diffDays: number): string {
  if (diffMins < 60) {
    return `in ${diffMins} min`
  }
  if (diffHours < 24) {
    return `in ${diffHours} hours`
  }
  return `in ${diffDays} days`
}

function formatPastTime(diffMins: number, diffHours: number, diffDays: number): string {
  if (Math.abs(diffMins) < 60) {
    return `${Math.abs(diffMins)} min ago`
  }
  if (Math.abs(diffHours) < 24) {
    return `${Math.abs(diffHours)} hours ago`
  }
  return `${Math.abs(diffDays)} days ago`
}

export function formatRelativeTime(dateString?: string): string {
  if (!dateString) {
    return 'never'
  }

  const date = new Date(dateString)
  const now = new Date()
  const diffMs = date.getTime() - now.getTime()
  const diffMins = Math.round(diffMs / 60_000)
  const diffHours = Math.round(diffMs / 3_600_000)
  const diffDays = Math.round(diffMs / 86_400_000)

  if (Math.abs(diffMins) < 1) {
    return 'just now'
  }

  if (diffMins > 0) {
    return formatFutureTime(diffMins, diffHours, diffDays)
  }

  return formatPastTime(diffMins, diffHours, diffDays)
}
