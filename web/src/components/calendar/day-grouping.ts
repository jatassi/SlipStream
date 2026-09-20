import { format, isToday, isTomorrow, parseISO } from 'date-fns'

import type { CalendarEvent } from '@/types/calendar'

export function dayHeader(date: string): string {
  const parsed = parseISO(date)
  if (isToday(parsed)) {
    return 'Today'
  }
  if (isTomorrow(parsed)) {
    return 'Tomorrow'
  }
  return format(parsed, 'EEEE, MMMM d')
}

export function groupEventsByDay(events: CalendarEvent[]): [string, CalendarEvent[]][] {
  const groups = new Map<string, CalendarEvent[]>()
  for (const event of events.toSorted((a, b) => a.date.localeCompare(b.date))) {
    const day = groups.get(event.date)
    if (day === undefined) {
      groups.set(event.date, [event])
    } else {
      day.push(event)
    }
  }
  return [...groups.entries()]
}
