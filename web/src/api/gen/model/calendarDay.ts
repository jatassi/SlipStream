import type { CalendarRow } from "./calendarRow.ts";

export interface CalendarDay {
  day: string;
  rows: CalendarRow[];
}
