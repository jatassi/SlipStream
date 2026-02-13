import { useMemo } from 'react'
import { format, parseISO, isToday, isPast, isFuture } from 'date-fns'
import { cn } from '@/lib/utils'
import { Skeleton } from '@/components/ui/skeleton'
import { CalendarEventCard } from './CalendarEventCard'
import type { CalendarEvent } from '@/types/calendar'

interface CalendarAgendaViewProps {
  events: CalendarEvent[]
  loading?: boolean
}

export function CalendarAgendaView({ events, loading }: CalendarAgendaViewProps) {
  const groupedEvents = useMemo(() => {
    const groups = new Map<string, CalendarEvent[]>()

    // Sort events by date
    const sorted = [...events].sort((a, b) => a.date.localeCompare(b.date))

    sorted.forEach((event) => {
      if (!groups.has(event.date)) {
        groups.set(event.date, [])
      }
      groups.get(event.date)!.push(event)
    })

    return Array.from(groups.entries())
  }, [events])

  if (loading) {
    return (
      <div className="space-y-6">
        {Array.from({ length: 7 }, (_, i) => (
          <div key={i}>
            <div className="flex items-center gap-3 py-2 mb-2">
              <Skeleton className="w-14 h-14 rounded-lg" />
              <div className="space-y-1.5">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-3 w-32" />
              </div>
            </div>
            <div className="space-y-2 pl-[70px]">
              <div className="rounded-lg border-l-4 border-l-muted-foreground/20 bg-muted/30 p-2 space-y-1.5">
                <div className="flex items-center gap-1">
                  <Skeleton className="size-3 rounded-full shrink-0" />
                  <Skeleton className="h-3.5 w-48" />
                </div>
                <Skeleton className="h-3 w-32" />
                <div className="flex gap-1">
                  <Skeleton className="h-4 w-12 rounded-full" />
                  <Skeleton className="h-4 w-10 rounded-full" />
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>
    )
  }

  if (groupedEvents.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-muted-foreground">
        No upcoming events in this date range
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {groupedEvents.map(([dateStr, dayEvents]) => {
        const date = parseISO(dateStr)
        const isCurrentDay = isToday(date)
        const isPastDay = isPast(date) && !isCurrentDay
        const isFutureDay = isFuture(date)

        return (
          <div key={dateStr}>
            <div
              className={cn(
                'sticky top-0 bg-background/95 supports-[backdrop-filter]:backdrop-blur-sm z-10 py-2 mb-2',
                'flex items-center gap-3'
              )}
            >
              <div
                className={cn(
                  'flex flex-col items-center justify-center w-14 h-14 rounded-lg',
                  isCurrentDay && 'bg-primary text-primary-foreground',
                  isPastDay && 'bg-muted text-muted-foreground',
                  isFutureDay && !isCurrentDay && 'bg-secondary'
                )}
              >
                <span className="text-xs uppercase">{format(date, 'EEE')}</span>
                <span className="text-xl font-bold">{format(date, 'd')}</span>
              </div>
              <div>
                <div className={cn(
                  'font-semibold',
                  isCurrentDay && 'text-primary'
                )}>
                  {isCurrentDay ? 'Today' : format(date, 'EEEE')}
                </div>
                <div className="text-sm text-muted-foreground">
                  {format(date, 'MMMM d, yyyy')}
                </div>
              </div>
              <div className="ml-auto text-sm text-muted-foreground">
                {dayEvents.length} {dayEvents.length === 1 ? 'event' : 'events'}
              </div>
            </div>

            <div className="space-y-2 pl-[70px]">
              {dayEvents.map((event) => (
                <CalendarEventCard
                  key={`${event.mediaType}-${event.id}-${event.eventType}`}
                  event={event}
                />
              ))}
            </div>
          </div>
        )
      })}
    </div>
  )
}
