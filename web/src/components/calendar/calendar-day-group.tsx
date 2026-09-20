import { createElement } from 'react'

import { Group, IconTile, Row } from '@/components/grouped-list'
import type { CalendarEvent } from '@/types/calendar'

import {
  eventDetail,
  eventHref,
  eventIcon,
  eventKey,
  eventStatusLabel,
  eventTileClass,
  eventTitle,
} from './event-presentation'

export function CalendarEventRow({ event }: { event: CalendarEvent }) {
  const href = eventHref(event)
  return (
    <Row
      href={href}
      chevron={href !== undefined}
      leading={
        <IconTile className={eventTileClass(event)}>{createElement(eventIcon(event))}</IconTile>
      }
      title={eventTitle(event)}
      subtitle={eventDetail(event)}
      trailing={<span className="text-footnote">{eventStatusLabel(event)}</span>}
    />
  )
}

export function CalendarDayGroup({ header, events }: { header: string; events: CalendarEvent[] }) {
  if (events.length === 0) {
    return (
      <Group header={header}>
        <Row title="No releases" />
      </Group>
    )
  }
  return (
    <Group header={header}>
      {events.map((event) => (
        <CalendarEventRow key={eventKey(event)} event={event} />
      ))}
    </Group>
  )
}
