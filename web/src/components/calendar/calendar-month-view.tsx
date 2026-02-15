import { useMemo } from 'react'

import {
  addMonths,
  eachDayOfInterval,
  endOfMonth,
  endOfWeek,
  format,
  isSameMonth,
  isToday,
  startOfMonth,
  startOfWeek,
  subMonths,
} from 'date-fns'
import { ChevronLeft, ChevronRight } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import type { CalendarEvent } from '@/types/calendar'

import { CalendarEventCard } from './calendar-event-card'

type CalendarMonthViewProps = {
  events: CalendarEvent[]
  currentDate: Date
  onDateChange: (date: Date) => void
  loading?: boolean
}

const WEEK_DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

function DayCellContent({ loading, dayEvents }: { loading?: boolean; dayEvents: CalendarEvent[] }) {
  if (loading) {
    return (
      <div className="border-l-muted-foreground/20 bg-muted/30 rounded-lg border-l-4 p-1">
        <div className="flex items-center gap-1">
          <Skeleton className="size-3 shrink-0 rounded-full" />
          <Skeleton className="h-3 w-full" />
        </div>
      </div>
    )
  }
  return (
    <>
      {dayEvents.slice(0, 3).map((event) => (
        <CalendarEventCard key={`${event.mediaType}-${event.id}-${event.eventType}`} event={event} compact />
      ))}
      {dayEvents.length > 3 && (
        <div className="text-muted-foreground text-center text-xs">+{dayEvents.length - 3} more</div>
      )}
    </>
  )
}

function MonthHeader({ currentDate, onDateChange }: { currentDate: Date; onDateChange: (date: Date) => void }) {
  return (
    <div className="mb-4 flex items-center justify-between">
      <div className="flex items-center gap-2">
        <Button variant="outline" size="icon" onClick={() => onDateChange(subMonths(currentDate, 1))}>
          <ChevronLeft className="size-4" />
        </Button>
        <Button variant="outline" size="icon" onClick={() => onDateChange(addMonths(currentDate, 1))}>
          <ChevronRight className="size-4" />
        </Button>
        <h2 className="ml-2 text-xl font-semibold">{format(currentDate, 'MMMM yyyy')}</h2>
      </div>
      <Button variant="outline" onClick={() => onDateChange(new Date())}>Today</Button>
    </div>
  )
}

function WeekDayHeaders() {
  return (
    <div className="grid grid-cols-7 border-b">
      {WEEK_DAYS.map((day) => (
        <div key={day} className="text-muted-foreground border-r py-2 text-center text-sm font-medium last:border-r-0">{day}</div>
      ))}
    </div>
  )
}

export function CalendarMonthView({ events, currentDate, onDateChange, loading }: CalendarMonthViewProps) {
  const monthStart = startOfMonth(currentDate)
  const monthEnd = endOfMonth(currentDate)
  const days = eachDayOfInterval({ start: startOfWeek(monthStart), end: endOfWeek(monthEnd) })

  const eventsByDate = useMemo(() => {
    const map = new Map<string, CalendarEvent[]>()
    events.forEach((event) => {
      const key = event.date
      if (!map.has(key)) {map.set(key, [])}
      map.get(key)?.push(event)
    })
    return map
  }, [events])

  return (
    <div className="flex h-full flex-col">
      <MonthHeader currentDate={currentDate} onDateChange={onDateChange} />
      <div className="flex-1 overflow-auto">
        <WeekDayHeaders />
        <div className="grid min-h-[600px] auto-rows-fr grid-cols-7">
          {days.map((day) => {
            const dateKey = format(day, 'yyyy-MM-dd')
            const dayEvents = eventsByDate.get(dateKey) ?? []
            const isCurrentMonth = isSameMonth(day, currentDate)
            const isCurrentDay = isToday(day)
            return (
              <div key={dateKey} className={cn('min-h-[120px] border-r border-b p-1 last:border-r-0', !isCurrentMonth && 'bg-muted/30', isCurrentDay && 'bg-primary/5')}>
                <div className={cn('mb-1 flex h-7 w-7 items-center justify-center rounded-full text-sm font-medium', isCurrentDay && 'bg-primary text-primary-foreground', !isCurrentMonth && 'text-muted-foreground')}>
                  {format(day, 'd')}
                </div>
                <div className="max-h-[90px] space-y-1 overflow-y-auto">
                  <DayCellContent loading={loading} dayEvents={dayEvents} />
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
