import { apiFetch, buildQueryString } from './client'
import type { CalendarEvent, CalendarRequest } from '@/types/calendar'

export const calendarApi = {
  getEvents: (params: CalendarRequest) =>
    apiFetch<CalendarEvent[]>(`/calendar${buildQueryString(params)}`),
}
