import type { CalendarDay } from "./calendarDay.ts";
import type { CalendarSentence } from "./calendarSentence.ts";

export interface CalendarAnswer {
  /** Only the days with rows, in order */
  days: CalendarDay[];
  /** When the Controller's next day starts */
  nextDayAt: string;
  /** What needs attention, else what comes out in the next 7 days; null when neither has anything */
  sentence: CalendarSentence | null;
  /** The Controller's day */
  today: string;
}
