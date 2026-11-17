import { z } from "zod";

import type { GetCalendarParams } from "../api/gen/model";
import {
  type IsoDay,
  type Month,
  addDays,
  isoDay,
  monthKey,
  monthOf,
  monthSpan,
} from "../format/days";
import { desktopQuery } from "../shell/media";

export const calendarViews = ["month", "list"] as const;
export type CalendarView = (typeof calendarViews)[number];

const monthPattern = /^(\d{4})-(\d{2})$/u;

/**
 * The Calendar's place in its address. A month (`2026-11`) or a list's first
 * day left out is the Controller's; `kinds` is a comma list, all when left out.
 */
export const calendarSearch = z.object({
  view: z.enum(calendarViews).optional().catch(undefined),
  month: z.string().regex(monthPattern).optional().catch(undefined),
  from: z
    .string()
    .regex(/^\d{4}-\d{2}-\d{2}$/u)
    .optional()
    .catch(undefined),
  kinds: z.string().optional().catch(undefined),
  unmonitored: z.boolean().optional().catch(undefined),
});

export type CalendarSearch = z.infer<typeof calendarSearch>;

/** List on a phone, so its rows aren't squeezed into seven columns, and Month otherwise. */
export const defaultView = (): CalendarView =>
  matchMedia(desktopQuery).matches ? "month" : "list";

/** The span a list or month reads: its first day and the day after its last, 42 days at most. */
export type CalendarPlace = {
  readonly view: CalendarView;
  readonly month: Month;
  readonly from: IsoDay;
  readonly to: IsoDay;
  readonly kinds: readonly string[] | undefined;
  readonly unmonitored: boolean;
};

export const listDays = 42;

function monthOfSearch(month: string | undefined): Month | undefined {
  const parts = month === undefined ? null : monthPattern.exec(month);
  return parts === null ? undefined : { year: Number(parts[1]), month: Number(parts[2]) };
}

export function placeOf(
  search: CalendarSearch,
  view: CalendarView,
  today: IsoDay,
  weekStart: number,
): CalendarPlace {
  const kinds = search.kinds?.split(",").filter((kind) => kind !== "");
  const shared = {
    view,
    kinds: kinds === undefined || kinds.length === 0 ? undefined : kinds,
    unmonitored: search.unmonitored === true,
  };
  if (view === "month") {
    const month = monthOfSearch(search.month) ?? monthOf(today);
    return { ...shared, month, ...monthSpan(month, weekStart) };
  }
  const from = search.from === undefined ? today : isoDay(search.from);
  return { ...shared, month: monthOf(from), from, to: addDays(from, listDays) };
}

export const paramsOf = (
  { from, to, kinds, unmonitored }: CalendarPlace,
  contentType: string | undefined,
): GetCalendarParams => ({
  from,
  to,
  ...(contentType !== undefined && { contentType }),
  ...(kinds !== undefined && { kinds: [...kinds] }),
  ...(unmonitored && { unmonitored }),
});

/**
 * The address naming the Controller's day once its answer says it differs
 * from the browser's guess, or undefined when the address already names a
 * place or the guess was right.
 */
export function controllersPlace(
  search: CalendarSearch,
  place: CalendarPlace,
  today: IsoDay,
): CalendarSearch | undefined {
  if (place.view === "month") {
    const month = monthKey(monthOf(today));
    return search.month === undefined && month !== monthKey(place.month)
      ? { ...search, month }
      : undefined;
  }
  return search.from === undefined && today !== place.from ? { ...search, from: today } : undefined;
}
