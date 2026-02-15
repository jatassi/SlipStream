function parseGoDuration(
  duration: string,
): { hours: number; minutes: number; seconds: number } | null {
  const match = /^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$/.exec(duration)
  if (!match) {
    return null
  }
  return {
    hours: Number.parseInt(match[1] || '0'),
    minutes: Number.parseInt(match[2] || '0'),
    seconds: Number.parseInt(match[3] || '0'),
  }
}

function to12Hour(hour: number): number {
  if (hour === 0) {
    return 12
  }
  return hour > 12 ? hour - 12 : hour
}

function formatTime(hour: number, minute: number): string {
  if (hour === 0 && minute === 0) {
    return 'midnight'
  }
  if (hour === 12 && minute === 0) {
    return 'noon'
  }
  const period = hour >= 12 ? 'PM' : 'AM'
  const displayMinute = minute.toString().padStart(2, '0')
  return `${to12Hour(hour)}:${displayMinute} ${period}`
}

function formatDurationPart(value: number, singular: string, plural: string): string {
  return value === 1 ? singular : `${value} ${plural}`
}

function parseEveryDirective(cron: string): string | null {
  if (!cron.startsWith('@every ')) {
    return null
  }
  const duration = parseGoDuration(cron.slice(7))
  if (!duration) {
    return cron
  }
  const { hours, minutes } = duration
  if (hours > 0 && minutes === 0) {
    return formatDurationPart(hours, 'Every hour', 'hours')
  }
  if (hours === 0 && minutes > 0) {
    return formatDurationPart(minutes, 'Every minute', 'minutes')
  }
  if (hours > 0 && minutes > 0) {
    return `Every ${hours}h ${minutes}m`
  }
  return cron
}

function formatIntervalMinutes(minute: string, allWild: boolean, hourIsWild: boolean): string | null {
  if (!minute.startsWith('*/') || !hourIsWild || !allWild) {
    return null
  }
  const interval = Number.parseInt(minute.slice(2))
  if (Number.isNaN(interval)) {
    return null
  }
  return formatDurationPart(interval, 'Every minute', 'minutes')
}

function formatIntervalHours(minute: string, hour: string, allWild: boolean): string | null {
  if (minute !== '0' || !hour.startsWith('*/') || !allWild) {
    return null
  }
  const interval = Number.parseInt(hour.slice(2))
  if (Number.isNaN(interval)) {
    return null
  }
  return formatDurationPart(interval, 'Every hour', 'hours')
}

type CronContext = {
  minute: string
  minuteNum: number
  hourNum: number
  hourIsWild: boolean
  allWild: boolean
  dayOfMonth: string
  month: string
  dayOfWeek: string
}

function formatHourlyAt(ctx: CronContext): string | null {
  if (Number.isNaN(ctx.minuteNum) || !ctx.hourIsWild || !ctx.allWild) {
    return null
  }
  if (ctx.minuteNum === 0) {
    return 'Every hour'
  }
  return `Every hour at :${ctx.minute.padStart(2, '0')}`
}

function formatDaily(ctx: CronContext): string | null {
  if (Number.isNaN(ctx.minuteNum) || Number.isNaN(ctx.hourNum) || !ctx.allWild) {
    return null
  }
  return `Daily at ${formatTime(ctx.hourNum, ctx.minuteNum)}`
}

const DAY_NAMES = [
  'Sundays',
  'Mondays',
  'Tuesdays',
  'Wednesdays',
  'Thursdays',
  'Fridays',
  'Saturdays',
]

function formatWeekly(ctx: CronContext): string | null {
  if (ctx.dayOfMonth !== '*' || ctx.month !== '*' || ctx.dayOfWeek === '*') {
    return null
  }
  const dayNum = Number.parseInt(ctx.dayOfWeek)
  const valid = !Number.isNaN(dayNum) && dayNum >= 0 && dayNum <= 6
  if (!valid || Number.isNaN(ctx.hourNum) || Number.isNaN(ctx.minuteNum)) {
    return null
  }
  return `Weekly on ${DAY_NAMES[dayNum]} at ${formatTime(ctx.hourNum, ctx.minuteNum)}`
}

function parseCronFields(cron: string): string {
  const parts = cron.split(' ')
  if (parts.length !== 5) {
    return cron
  }

  const [minute, hour, dayOfMonth, month, dayOfWeek] = parts
  const ctx: CronContext = {
    minute,
    minuteNum: Number.parseInt(minute),
    hourNum: Number.parseInt(hour),
    hourIsWild: hour === '*',
    allWild: dayOfMonth === '*' && month === '*' && dayOfWeek === '*',
    dayOfMonth,
    month,
    dayOfWeek,
  }

  return (
    formatIntervalMinutes(minute, ctx.allWild, ctx.hourIsWild) ??
    formatIntervalHours(minute, hour, ctx.allWild) ??
    formatHourlyAt(ctx) ??
    formatDaily(ctx) ??
    formatWeekly(ctx) ??
    cron
  )
}

export function cronToPlainEnglish(cron: string): string {
  return parseEveryDirective(cron) ?? parseCronFields(cron)
}

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
    return 'Never'
  }

  const date = new Date(dateString)
  const now = new Date()
  const diffMs = date.getTime() - now.getTime()
  const diffMins = Math.round(diffMs / 60_000)
  const diffHours = Math.round(diffMs / 3_600_000)
  const diffDays = Math.round(diffMs / 86_400_000)

  if (Math.abs(diffMins) < 1) {
    return 'Just now'
  }

  if (diffMins > 0) {
    return formatFutureTime(diffMins, diffHours, diffDays)
  }

  return formatPastTime(diffMins, diffHours, diffDays)
}
