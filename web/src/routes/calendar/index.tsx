import { useState, useMemo } from 'react'
import {
  format,
  startOfMonth,
  endOfMonth,
  startOfWeek,
  endOfWeek,
  addDays,
} from 'date-fns'
import { CalendarDays, CalendarRange, List } from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { LoadingState } from '@/components/data/LoadingState'
import { ErrorState } from '@/components/data/ErrorState'
import { useCalendarEvents } from '@/hooks'
import {
  CalendarMonthView,
  CalendarWeekView,
  CalendarAgendaView,
} from '@/components/calendar'
import type { CalendarView } from '@/types/calendar'

export function CalendarPage() {
  const [view, setView] = useState<CalendarView>('month')
  const [currentDate, setCurrentDate] = useState(new Date())

  // Calculate date range based on view
  const dateRange = useMemo(() => {
    if (view === 'month') {
      const monthStart = startOfMonth(currentDate)
      const monthEnd = endOfMonth(currentDate)
      // Include days from prev/next months shown in calendar grid
      return {
        start: format(startOfWeek(monthStart), 'yyyy-MM-dd'),
        end: format(endOfWeek(monthEnd), 'yyyy-MM-dd'),
      }
    } else if (view === 'week') {
      const weekStart = startOfWeek(currentDate)
      const weekEnd = endOfWeek(currentDate)
      return {
        start: format(weekStart, 'yyyy-MM-dd'),
        end: format(weekEnd, 'yyyy-MM-dd'),
      }
    } else {
      // Agenda: show 30 days from today
      return {
        start: format(new Date(), 'yyyy-MM-dd'),
        end: format(addDays(new Date(), 30), 'yyyy-MM-dd'),
      }
    }
  }, [view, currentDate])

  const { data: events, isLoading, isError, refetch } = useCalendarEvents(dateRange)

  const handleViewChange = (newView: string[]) => {
    if (newView.length > 0) {
      setView(newView[0] as CalendarView)
    }
  }

  if (isLoading) {
    return (
      <div>
        <PageHeader title="Calendar" />
        <LoadingState variant="list" />
      </div>
    )
  }

  if (isError) {
    return (
      <div>
        <PageHeader title="Calendar" />
        <ErrorState onRetry={refetch} />
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col">
      <PageHeader
        title="Calendar"
        description="Upcoming releases and air dates"
        actions={
          <ToggleGroup
            value={[view]}
            onValueChange={handleViewChange}
          >
            <ToggleGroupItem value="month" aria-label="Month view">
              <CalendarDays className="size-4" />
            </ToggleGroupItem>
            <ToggleGroupItem value="week" aria-label="Week view">
              <CalendarRange className="size-4" />
            </ToggleGroupItem>
            <ToggleGroupItem value="agenda" aria-label="Agenda view">
              <List className="size-4" />
            </ToggleGroupItem>
          </ToggleGroup>
        }
      />

      <div className="flex-1 min-h-0">
        {view === 'month' && (
          <CalendarMonthView
            events={events || []}
            currentDate={currentDate}
            onDateChange={setCurrentDate}
            loading={isLoading}
          />
        )}
        {view === 'week' && (
          <CalendarWeekView
            events={events || []}
            currentDate={currentDate}
            onDateChange={setCurrentDate}
            loading={isLoading}
          />
        )}
        {view === 'agenda' && (
          <CalendarAgendaView
            events={events || []}
            loading={isLoading}
          />
        )}
      </div>
    </div>
  )
}
