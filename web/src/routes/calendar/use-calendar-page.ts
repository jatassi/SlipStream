import { useState } from 'react'

import { addDays, endOfMonth, endOfWeek, format, startOfMonth, startOfWeek } from 'date-fns'

import { useCalendarEvents } from '@/hooks'
import { useUIStore } from '@/stores'
import type { CalendarView } from '@/types/calendar'

const LIST_DAYS = 30

function rangeFor(view: CalendarView, month: Date) {
  if (view === 'month') {
    return {
      start: format(startOfWeek(startOfMonth(month)), 'yyyy-MM-dd'),
      end: format(endOfWeek(endOfMonth(month)), 'yyyy-MM-dd'),
    }
  }
  const today = new Date()
  return {
    start: format(today, 'yyyy-MM-dd'),
    end: format(addDays(today, LIST_DAYS), 'yyyy-MM-dd'),
  }
}

export function useCalendarPage() {
  const [view, setView] = useState<CalendarView>('month')
  const [month, setMonth] = useState(() => startOfMonth(new Date()))
  const [selected, setSelected] = useState(() => format(new Date(), 'yyyy-MM-dd'))
  const globalLoading = useUIStore((s) => s.globalLoading)
  const { data, isLoading, isError, refetch } = useCalendarEvents(rangeFor(view, month))

  const changeMonth = (next: Date) => {
    setMonth(next)
    setSelected(format(next, 'yyyy-MM-dd'))
  }

  return {
    view,
    setView,
    month,
    changeMonth,
    selected,
    setSelected,
    events: data ?? [],
    isLoading: isLoading || globalLoading,
    isError,
    refetch,
  }
}
