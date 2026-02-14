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

import { CalendarEventCard } from './CalendarEventCard'

type CalendarMonthViewProps = {
  events: CalendarEvent[]
  currentDate: Date
  onDateChange: (date: Date) => void
  loading?: boolean
}

export function CalendarMonthView({
  events,
  currentDate,
  onDateChange,
  loading,
}: CalendarMonthViewProps) {
  const monthStart = startOfMonth(currentDate)
  const monthEnd = endOfMonth(currentDate)
  const calendarStart = startOfWeek(monthStart)
  const calendarEnd = endOfWeek(monthEnd)

  const days = eachDayOfInterval({ start: calendarStart, end: calendarEnd })
  const weekDays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

  const eventsByDate = useMemo(() => {
    const map = new Map<string, CalendarEvent[]>()
    events.forEach((event) => {
      const dateKey = event.date
      if (!map.has(dateKey)) {
        map.set(dateKey, [])
      }
      const dateEvents = map.get(dateKey)
      if (dateEvents) {
        dateEvents.push(event)
      }
    })
    return map
  }, [events])

  const handlePrevMonth = () => onDateChange(subMonths(currentDate, 1))
  const handleNextMonth = () => onDateChange(addMonths(currentDate, 1))
  const handleToday = () => onDateChange(new Date())

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="mb-4 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Button variant="outline" size="icon" onClick={handlePrevMonth}>
            <ChevronLeft className="size-4" />
          </Button>
          <Button variant="outline" size="icon" onClick={handleNextMonth}>
            <ChevronRight className="size-4" />
          </Button>
          <h2 className="ml-2 text-xl font-semibold">{format(currentDate, 'MMMM yyyy')}</h2>
        </div>
        <Button variant="outline" onClick={handleToday}>
          Today
        </Button>
      </div>

      {/* Calendar Grid */}
      <div className="flex-1 overflow-auto">
        <div className="grid grid-cols-7 border-b">
          {weekDays.map((day) => (
            <div
              key={day}
              className="text-muted-foreground border-r py-2 text-center text-sm font-medium last:border-r-0"
            >
              {day}
            </div>
          ))}
        </div>

        <div className="grid min-h-[600px] auto-rows-fr grid-cols-7">
          {days.map((day) => {
            const dateKey = format(day, 'yyyy-MM-dd')
            const dayEvents = eventsByDate.get(dateKey) || []
            const isCurrentMonth = isSameMonth(day, currentDate)
            const isCurrentDay = isToday(day)

            return (
              <div
                key={dateKey}
                className={cn(
                  'min-h-[120px] border-r border-b p-1 last:border-r-0',
                  !isCurrentMonth && 'bg-muted/30',
                  isCurrentDay && 'bg-primary/5',
                )}
              >
                <div
                  className={cn(
                    'mb-1 flex h-7 w-7 items-center justify-center rounded-full text-sm font-medium',
                    isCurrentDay && 'bg-primary text-primary-foreground',
                    !isCurrentMonth && 'text-muted-foreground',
                  )}
                >
                  {format(day, 'd')}
                </div>

                <div className="max-h-[90px] space-y-1 overflow-y-auto">
                  {loading ? (
                    <div className="border-l-muted-foreground/20 bg-muted/30 rounded-lg border-l-4 p-1">
                      <div className="flex items-center gap-1">
                        <Skeleton className="size-3 shrink-0 rounded-full" />
                        <Skeleton className="h-3 w-full" />
                      </div>
                    </div>
                  ) : (
                    dayEvents
                      .slice(0, 3)
                      .map((event) => (
                        <CalendarEventCard
                          key={`${event.mediaType}-${event.id}-${event.eventType}`}
                          event={event}
                          compact
                        />
                      ))
                  )}
                  {dayEvents.length > 3 && (
                    <div className="text-muted-foreground text-center text-xs">
                      +{dayEvents.length - 3} more
                    </div>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
