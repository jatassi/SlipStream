import { useMemo, useState } from 'react'

import { addDays, endOfMonth, endOfWeek, format, startOfMonth, startOfWeek } from 'date-fns'
import { CalendarDays, CalendarRange, List } from 'lucide-react'

import { CalendarAgendaView, CalendarMonthView, CalendarWeekView } from '@/components/calendar'
import { ErrorState } from '@/components/data/error-state'
import { PageHeader } from '@/components/layout/page-header'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { useCalendarEvents, useGlobalLoading } from '@/hooks'
import type { CalendarView } from '@/types/calendar'

function getDateRange(view: CalendarView, currentDate: Date) {
  if (view === 'month') {
    return {
      start: format(startOfWeek(startOfMonth(currentDate)), 'yyyy-MM-dd'),
      end: format(endOfWeek(endOfMonth(currentDate)), 'yyyy-MM-dd'),
    }
  }
  if (view === 'week') {
    return {
      start: format(startOfWeek(currentDate), 'yyyy-MM-dd'),
      end: format(endOfWeek(currentDate), 'yyyy-MM-dd'),
    }
  }
  return {
    start: format(new Date(), 'yyyy-MM-dd'),
    end: format(addDays(new Date(), 30), 'yyyy-MM-dd'),
  }
}

function ViewToggle({ view, onViewChange }: { view: CalendarView; onViewChange: (v: string[]) => void }) {
  return (
    <ToggleGroup value={[view]} onValueChange={onViewChange}>
      <ToggleGroupItem value="month" aria-label="Month view"><CalendarDays className="size-4" /></ToggleGroupItem>
      <ToggleGroupItem value="week" aria-label="Week view"><CalendarRange className="size-4" /></ToggleGroupItem>
      <ToggleGroupItem value="agenda" aria-label="Agenda view"><List className="size-4" /></ToggleGroupItem>
    </ToggleGroup>
  )
}

export function CalendarPage() {
  const [view, setView] = useState<CalendarView>('month')
  const [currentDate, setCurrentDate] = useState(new Date())
  const dateRange = useMemo(() => getDateRange(view, currentDate), [view, currentDate])

  const globalLoading = useGlobalLoading()
  const { data: events, isLoading: queryLoading, isError, refetch } = useCalendarEvents(dateRange)
  const isLoading = queryLoading || globalLoading
  const safeEvents = events ?? []

  const handleViewChange = (newView: string[]) => {
    if (newView.length > 0) {setView(newView[0] as CalendarView)}
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
    <div className="flex h-full flex-col">
      <PageHeader title="Calendar" description="Upcoming releases and air dates" actions={<ViewToggle view={view} onViewChange={handleViewChange} />} />
      <div className="min-h-0 flex-1">
        {view === 'month' && <CalendarMonthView events={safeEvents} currentDate={currentDate} onDateChange={setCurrentDate} loading={isLoading} />}
        {view === 'week' && <CalendarWeekView events={safeEvents} currentDate={currentDate} onDateChange={setCurrentDate} loading={isLoading} />}
        {view === 'agenda' && <CalendarAgendaView events={safeEvents} loading={isLoading} />}
      </div>
    </div>
  )
}
