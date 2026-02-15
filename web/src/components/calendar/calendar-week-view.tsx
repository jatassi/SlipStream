import { useMemo } from 'react'

import {
  addWeeks,
  eachDayOfInterval,
  endOfWeek,
  format,
  isToday,
  startOfWeek,
  subWeeks,
} from 'date-fns'
import { ChevronLeft, ChevronRight } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import type { CalendarEvent } from '@/types/calendar'

import { CalendarEventCard } from './calendar-event-card'

function DayContent({ loading, dayEvents }: { loading?: boolean; dayEvents: CalendarEvent[] }) {
  if (loading) {
    return (
      <div className="space-y-2">
        <div className="border-l-muted-foreground/20 bg-muted/30 space-y-1.5 rounded-lg border-l-4 p-2">
          <div className="flex items-center gap-1">
            <Skeleton className="size-3 shrink-0 rounded-full" />
            <Skeleton className="h-3.5 w-3/4" />
          </div>
          <Skeleton className="h-3 w-1/2" />
          <div className="flex gap-1">
            <Skeleton className="h-4 w-12 rounded-full" />
            <Skeleton className="h-4 w-10 rounded-full" />
          </div>
        </div>
      </div>
    )
  }
  if (dayEvents.length > 0) {
    return (
      <div className="space-y-2">
        {dayEvents.map((event) => (
          <CalendarEventCard key={`${event.mediaType}-${event.id}-${event.eventType}`} event={event} />
        ))}
      </div>
    )
  }
  return (
    <div className="space-y-2">
      <div className="text-muted-foreground py-4 text-center text-sm">No events</div>
    </div>
  )
}

function WeekHeader({ weekStart, weekEnd, onDateChange }: { weekStart: Date; weekEnd: Date; onDateChange: (date: Date) => void }) {
  return (
    <div className="mb-4 flex items-center justify-between">
      <div className="flex items-center gap-2">
        <Button variant="outline" size="icon" onClick={() => onDateChange(subWeeks(weekStart, 1))}>
          <ChevronLeft className="size-4" />
        </Button>
        <Button variant="outline" size="icon" onClick={() => onDateChange(addWeeks(weekStart, 1))}>
          <ChevronRight className="size-4" />
        </Button>
        <h2 className="ml-2 text-xl font-semibold">
          {format(weekStart, 'MMM d')} - {format(weekEnd, 'MMM d, yyyy')}
        </h2>
      </div>
      <Button variant="outline" onClick={() => onDateChange(new Date())}>Today</Button>
    </div>
  )
}

type CalendarWeekViewProps = {
  events: CalendarEvent[]
  currentDate: Date
  onDateChange: (date: Date) => void
  loading?: boolean
}

export function CalendarWeekView({ events, currentDate, onDateChange, loading }: CalendarWeekViewProps) {
  const weekStart = startOfWeek(currentDate)
  const weekEnd = endOfWeek(currentDate)
  const days = eachDayOfInterval({ start: weekStart, end: weekEnd })

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
      <WeekHeader weekStart={weekStart} weekEnd={weekEnd} onDateChange={onDateChange} />
      <div className="flex-1 overflow-auto">
        <div className="grid grid-cols-7 gap-4">
          {days.map((day) => {
            const dateKey = format(day, 'yyyy-MM-dd')
            const dayEvents = eventsByDate.get(dateKey) ?? []
            const isCurrentDay = isToday(day)
            return (
              <div key={dateKey} className={cn('min-h-[500px] rounded-lg border p-3 supports-[backdrop-filter]:backdrop-blur-sm', isCurrentDay && 'border-primary bg-primary/5')}>
                <div className="mb-3 text-center">
                  <div className="text-muted-foreground text-xs uppercase">{format(day, 'EEE')}</div>
                  <div className={cn('text-2xl font-bold', isCurrentDay && 'text-primary')}>{format(day, 'd')}</div>
                </div>
                <DayContent loading={loading} dayEvents={dayEvents} />
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
