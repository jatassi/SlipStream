import { useQuery } from '@tanstack/react-query'
import { calendarApi } from '@/api'
import type { CalendarRequest } from '@/types/calendar'

export const calendarKeys = {
  all: ['calendar'] as const,
  events: (params: CalendarRequest) => [...calendarKeys.all, 'events', params] as const,
}

export function useCalendarEvents(params: CalendarRequest) {
  return useQuery({
    queryKey: calendarKeys.events(params),
    queryFn: () => calendarApi.getEvents(params),
    enabled: !!params.start && !!params.end,
  })
}
