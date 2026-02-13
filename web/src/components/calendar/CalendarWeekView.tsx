import { useMemo } from 'react'
import {
  format,
  eachDayOfInterval,
  startOfWeek,
  endOfWeek,
  addWeeks,
  subWeeks,
  isToday,
} from 'date-fns'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { CalendarEventCard } from './CalendarEventCard'
import type { CalendarEvent } from '@/types/calendar'

interface CalendarWeekViewProps {
  events: CalendarEvent[]
  currentDate: Date
  onDateChange: (date: Date) => void
  loading?: boolean
}

export function CalendarWeekView({
  events,
  currentDate,
  onDateChange,
  loading,
}: CalendarWeekViewProps) {
  const weekStart = startOfWeek(currentDate)
  const weekEnd = endOfWeek(currentDate)
  const days = eachDayOfInterval({ start: weekStart, end: weekEnd })

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

  const handlePrevWeek = () => onDateChange(subWeeks(currentDate, 1))
  const handleNextWeek = () => onDateChange(addWeeks(currentDate, 1))
  const handleToday = () => onDateChange(new Date())

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Button variant="outline" size="icon" onClick={handlePrevWeek}>
            <ChevronLeft className="size-4" />
          </Button>
          <Button variant="outline" size="icon" onClick={handleNextWeek}>
            <ChevronRight className="size-4" />
          </Button>
          <h2 className="text-xl font-semibold ml-2">
            {format(weekStart, 'MMM d')} - {format(weekEnd, 'MMM d, yyyy')}
          </h2>
        </div>
        <Button variant="outline" onClick={handleToday}>
          Today
        </Button>
      </div>

      {/* Week Grid */}
      <div className="flex-1 overflow-auto">
        <div className="grid grid-cols-7 gap-4">
          {days.map((day) => {
            const dateKey = format(day, 'yyyy-MM-dd')
            const dayEvents = eventsByDate.get(dateKey) || []
            const isCurrentDay = isToday(day)

            return (
              <div
                key={dateKey}
                className={cn(
                  'rounded-lg border p-3 min-h-[500px]',
                  'supports-[backdrop-filter]:backdrop-blur-sm',
                  isCurrentDay && 'border-primary bg-primary/5'
                )}
              >
                <div className="mb-3 text-center">
                  <div className="text-xs text-muted-foreground uppercase">
                    {format(day, 'EEE')}
                  </div>
                  <div
                    className={cn(
                      'text-2xl font-bold',
                      isCurrentDay && 'text-primary'
                    )}
                  >
                    {format(day, 'd')}
                  </div>
                </div>

                <div className="space-y-2">
                  {loading ? (
                    <div className="rounded-lg border-l-4 border-l-muted-foreground/20 bg-muted/30 p-2 space-y-1.5">
                      <div className="flex items-center gap-1">
                        <Skeleton className="size-3 rounded-full shrink-0" />
                        <Skeleton className="h-3.5 w-3/4" />
                      </div>
                      <Skeleton className="h-3 w-1/2" />
                      <div className="flex gap-1">
                        <Skeleton className="h-4 w-12 rounded-full" />
                        <Skeleton className="h-4 w-10 rounded-full" />
                      </div>
                    </div>
                  ) : dayEvents.length > 0 ? (
                    dayEvents.map((event) => (
                      <CalendarEventCard
                        key={`${event.mediaType}-${event.id}-${event.eventType}`}
                        event={event}
                      />
                    ))
                  ) : (
                    <div className="text-center text-sm text-muted-foreground py-4">
                      No events
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
