import type { CalendarEvent, CalendarRequest } from '@/types/calendar'

import { apiFetch, buildQueryString } from './client'

export const calendarApi = {
  getEvents: (params: CalendarRequest) =>
    apiFetch<CalendarEvent[]>(`/calendar${buildQueryString(params)}`),
}
