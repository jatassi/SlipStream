import type { QueryClient } from "@tanstack/react-query";

import { getGetCalendarSuspenseQueryOptions } from "../api/gen/query/calendar/calendar";
import { ensureQuery } from "../api/query-client";
import { browserToday, weekStart } from "../format/days";
import { everyPath } from "../live/refresh";
import { type CalendarSearch, defaultView, paramsOf, placeOf } from "./routed";

/** Every span of the Calendar a page or the cache holds. */
export const calendarPaths = everyPath(/^\/api\/v1\/calendar$/u);

/** The span the address names, read before the page shows, with the browser's day as its guess at the Controller's. */
export const loadCalendar = (
  queryClient: QueryClient,
  search: CalendarSearch,
  contentType: string | undefined,
) =>
  ensureQuery(
    queryClient,
    getGetCalendarSuspenseQueryOptions(
      paramsOf(
        placeOf(search, search.view ?? defaultView(), browserToday(), weekStart()),
        contentType,
      ),
    ),
  );
