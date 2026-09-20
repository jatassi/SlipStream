import { Group, Row, RowSkeleton } from '@/components/grouped-list'
import type { CalendarEvent } from '@/types/calendar'

import { CalendarDayGroup } from './calendar-day-group'
import { dayHeader, groupEventsByDay } from './day-grouping'

const SKELETONS = ['day-a', 'day-b', 'day-c'] as const

export function CalendarListView({
  events,
  loading,
}: {
  events: CalendarEvent[]
  loading: boolean
}) {
  if (loading) {
    return (
      <Group>
        {SKELETONS.map((id) => (
          <RowSkeleton key={id} trailing />
        ))}
      </Group>
    )
  }

  const groups = groupEventsByDay(events)
  if (groups.length === 0) {
    return (
      <Group>
        <Row title="Nothing upcoming" subtitle="No releases in the next 30 days" />
      </Group>
    )
  }

  return (
    <>
      {groups.map(([date, dayEvents]) => (
        <CalendarDayGroup key={date} header={dayHeader(date)} events={dayEvents} />
      ))}
    </>
  )
}
