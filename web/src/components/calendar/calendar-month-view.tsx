import {
  addMonths,
  eachDayOfInterval,
  endOfMonth,
  endOfWeek,
  format,
  isSameMonth,
  isToday,
  parseISO,
  startOfMonth,
  startOfWeek,
  subMonths,
} from 'date-fns'
import { ChevronLeft, ChevronRight } from 'lucide-react'

import { cn } from '@/lib/utils'
import type { CalendarEvent } from '@/types/calendar'

import { CalendarDayGroup } from './calendar-day-group'
import { eventDotClass, eventKey } from './event-presentation'

const WEEK_DAY_INITIALS = ['S', 'M', 'T', 'W', 'T', 'F', 'S']
const MAX_DOTS = 3

type MonthViewProps = {
  events: CalendarEvent[]
  month: Date
  onMonthChange: (month: Date) => void
  selected: string
  onSelect: (date: string) => void
}

const STEP =
  'press size-tap focus-visible:ring-ring flex items-center justify-center rounded-full outline-none focus-visible:ring-[3px]'

function eventsByDate(events: CalendarEvent[]): Map<string, CalendarEvent[]> {
  const map = new Map<string, CalendarEvent[]>()
  for (const event of events) {
    const day = map.get(event.date)
    if (day === undefined) {
      map.set(event.date, [event])
    } else {
      day.push(event)
    }
  }
  return map
}

function MonthHeader({ month, onMonthChange }: Pick<MonthViewProps, 'month' | 'onMonthChange'>) {
  return (
    <div className="px-screen flex items-center justify-between pb-2">
      <h2 className="text-title font-semibold">{format(month, 'MMMM yyyy')}</h2>
      <div className="flex items-center">
        <button
          type="button"
          aria-label="Previous month"
          className={STEP}
          onClick={() => {
            onMonthChange(subMonths(month, 1))
          }}
        >
          <ChevronLeft className="size-5" />
        </button>
        <button
          type="button"
          aria-label="Next month"
          className={STEP}
          onClick={() => {
            onMonthChange(addMonths(month, 1))
          }}
        >
          <ChevronRight className="size-5" />
        </button>
      </div>
    </div>
  )
}

function WeekDayHeader() {
  return (
    <div className="grid grid-cols-7" aria-hidden="true">
      {WEEK_DAY_INITIALS.map((day, index) => (
        <span
          key={`${day}-${index.toString()}`}
          className="text-caption text-muted-foreground py-1 text-center font-semibold"
        >
          {day}
        </span>
      ))}
    </div>
  )
}

function DayDots({ events }: { events: CalendarEvent[] }) {
  return (
    <span className="mt-0.5 flex h-1.5 items-center justify-center gap-0.5">
      {events.slice(0, MAX_DOTS).map((event) => (
        <span
          key={eventKey(event)}
          className={cn('size-1.5 rounded-full', eventDotClass(event))}
        />
      ))}
    </span>
  )
}

function dayLabel(day: Date, count: number): string {
  const date = format(day, 'MMMM d')
  if (count === 0) {
    return `${date}, no releases`
  }
  if (count === 1) {
    return `${date}, 1 release`
  }
  return `${date}, ${count.toString()} releases`
}

function DayCell({
  day,
  month,
  events,
  selected,
  onSelect,
}: {
  day: Date
  month: Date
  events: CalendarEvent[]
  selected: boolean
  onSelect: (date: string) => void
}) {
  const key = format(day, 'yyyy-MM-dd')
  const outside = !isSameMonth(day, month)
  return (
    <button
      type="button"
      aria-label={dayLabel(day, events.length)}
      aria-pressed={selected}
      onClick={() => {
        onSelect(key)
      }}
      className="press min-h-tap focus-visible:ring-ring flex min-w-0 flex-col items-center justify-center rounded-[10px] py-1 outline-none focus-visible:ring-[3px]"
    >
      <span
        className={cn(
          'nums text-body flex size-7 items-center justify-center rounded-full font-medium',
          outside && 'text-muted-foreground/60',
          isToday(day) && !selected && 'text-primary font-semibold',
          selected && 'bg-foreground text-background font-semibold',
        )}
      >
        {format(day, 'd')}
      </span>
      <DayDots events={events} />
    </button>
  )
}

export function CalendarMonthView({
  events,
  month,
  onMonthChange,
  selected,
  onSelect,
}: MonthViewProps) {
  const byDate = eventsByDate(events)
  const days = eachDayOfInterval({
    start: startOfWeek(startOfMonth(month)),
    end: endOfWeek(endOfMonth(month)),
  })

  return (
    <>
      <MonthHeader month={month} onMonthChange={onMonthChange} />
      <div className="px-screen pb-5">
        <WeekDayHeader />
        <div className="grid grid-cols-7">
          {days.map((day) => {
            const key = format(day, 'yyyy-MM-dd')
            return (
              <DayCell
                key={key}
                day={day}
                month={month}
                events={byDate.get(key) ?? []}
                selected={key === selected}
                onSelect={onSelect}
              />
            )
          })}
        </div>
      </div>
      <CalendarDayGroup
        header={format(parseISO(selected), 'EEEE, MMMM d')}
        events={byDate.get(selected) ?? []}
      />
    </>
  )
}
