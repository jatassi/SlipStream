import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { useEffect } from "react";

import type { CalendarAnswer } from "../api/gen/model";
import { getGetCalendarSuspenseQueryOptions } from "../api/gen/query/calendar/calendar";
import { browserToday, isoDay, weekStart } from "../format/days";
import { refresh } from "../live/refresh";
import { calendarPaths } from "./calendar-query";
import {
  type CalendarPlace,
  type CalendarSearch,
  controllersPlace,
  defaultView,
  paramsOf,
  placeOf,
} from "./routed";

/**
 * The span the address names and the Controller's answer for it. The
 * browser's day stands in for the Controller's until the answer names it,
 * then a place the address leaves out moves to the Controller's day. The
 * answer is read again when the Controller's next day starts.
 */
export function useCalendar(
  search: CalendarSearch,
  contentType: string | undefined,
  onSearch: (next: CalendarSearch) => void,
): { place: CalendarPlace; answer: CalendarAnswer } {
  const queryClient = useQueryClient();
  const place = placeOf(search, search.view ?? defaultView(), browserToday(), weekStart());
  const answer = useSuspenseQuery(
    getGetCalendarSuspenseQueryOptions(paramsOf(place, contentType)),
  ).data;
  const controllers = controllersPlace(search, place, isoDay(answer.today));
  useEffect(() => {
    if (controllers !== undefined) {
      onSearch(controllers);
    }
  }, [controllers, onSearch]);
  useEffect(() => {
    const timer = setTimeout(
      () => {
        void refresh(queryClient, calendarPaths);
      },
      Math.max(0, Date.parse(answer.nextDayAt) - Date.now()),
    );
    return () => {
      clearTimeout(timer);
    };
  }, [queryClient, answer.nextDayAt]);
  return { place, answer };
}
