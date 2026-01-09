import { useMemo } from 'react'
import {
  format,
  startOfMonth,
  endOfMonth,
  eachDayOfInterval,
  isSameMonth,
  isToday,
  addMonths,
  subMonths,
  startOfWeek,
  endOfWeek,
} from 'date-fns'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { CalendarEventCard } from './CalendarEventCard'
import type { CalendarEvent } from '@/types/calendar'

interface CalendarMonthViewProps {
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
      map.get(dateKey)!.push(event)
    })
    return map
  }, [events])

  const handlePrevMonth = () => onDateChange(subMonths(currentDate, 1))
  const handleNextMonth = () => onDateChange(addMonths(currentDate, 1))
  const handleToday = () => onDateChange(new Date())

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Button variant="outline" size="icon" onClick={handlePrevMonth}>
            <ChevronLeft className="size-4" />
          </Button>
          <Button variant="outline" size="icon" onClick={handleNextMonth}>
            <ChevronRight className="size-4" />
          </Button>
          <h2 className="text-xl font-semibold ml-2">
            {format(currentDate, 'MMMM yyyy')}
          </h2>
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
              className="py-2 text-center text-sm font-medium text-muted-foreground border-r last:border-r-0"
            >
              {day}
            </div>
          ))}
        </div>

        <div className="grid grid-cols-7 auto-rows-fr min-h-[600px]">
          {days.map((day) => {
            const dateKey = format(day, 'yyyy-MM-dd')
            const dayEvents = eventsByDate.get(dateKey) || []
            const isCurrentMonth = isSameMonth(day, currentDate)
            const isCurrentDay = isToday(day)

            return (
              <div
                key={dateKey}
                className={cn(
                  'border-r border-b p-1 min-h-[120px] last:border-r-0',
                  !isCurrentMonth && 'bg-muted/30',
                  isCurrentDay && 'bg-primary/5'
                )}
              >
                <div
                  className={cn(
                    'text-sm font-medium mb-1 w-7 h-7 flex items-center justify-center rounded-full',
                    isCurrentDay && 'bg-primary text-primary-foreground',
                    !isCurrentMonth && 'text-muted-foreground'
                  )}
                >
                  {format(day, 'd')}
                </div>

                <div className="space-y-1 overflow-y-auto max-h-[90px]">
                  {loading ? (
                    <div className="h-6 bg-muted animate-pulse rounded" />
                  ) : (
                    dayEvents.slice(0, 3).map((event) => (
                      <CalendarEventCard
                        key={`${event.mediaType}-${event.id}-${event.eventType}`}
                        event={event}
                        compact
                      />
                    ))
                  )}
                  {dayEvents.length > 3 && (
                    <div className="text-xs text-muted-foreground text-center">
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
