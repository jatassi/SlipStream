import type { QueryClient } from "@tanstack/react-query";

import type { CalendarAnswer } from "../api/gen/model";
import { calendarPaths } from "../calendar/calendar-query";
import type { LiveEffect, LiveEffects, ProductEventName } from "./types";

/** The Works the Calendar answers list, of the spans a page shows or of every cached span. */
function worksOf(queryClient: QueryClient, shown: boolean): Set<string> {
  const ids = new Set<string>();
  const queries = queryClient
    .getQueryCache()
    .findAll({ ...calendarPaths, ...(shown && { type: "active" as const }) });
  for (const { queryKey } of queries) {
    for (const { rows } of queryClient.getQueryData<CalendarAnswer>(queryKey)?.days ?? []) {
      for (const { work } of rows) {
        ids.add(work.id);
      }
    }
  }
  return ids;
}

/** Any of the Works is in a span a page shows. */
const inView = (queryClient: QueryClient, workIds: readonly string[]) => {
  const shown = worksOf(queryClient, true);
  return workIds.some((id) => shown.has(id));
};

/** Every one of the Works is in some cached span, so none of them is newly dated. */
const seen = (queryClient: QueryClient, workIds: readonly string[]) => {
  const cached = worksOf(queryClient, false);
  return workIds.every((id) => cached.has(id));
};

/**
 * The Calendar reads again only for a change it can show: a Work in view, a
 * Work no cached span lists, which may now have a date in view, or a change
 * to every type. A Grab's progress tick changes no state.
 */
export const calendarEffects = {
  "works.changed": ({ workIds }, { queryClient, refresh }) => {
    if (inView(queryClient, workIds) || !seen(queryClient, workIds)) {
      refresh(calendarPaths);
    }
  },
  "works.removed": ({ workIds }, { queryClient, refresh }) => {
    if (inView(queryClient, workIds)) {
      refresh(calendarPaths);
    }
  },
  "works.type-changed": (_data, { refresh }) => {
    refresh(calendarPaths);
  },
  "works.content-types-changed": (_data, { refresh }) => {
    refresh(calendarPaths);
  },
  "files.changed": ({ workId }, { queryClient, refresh }) => {
    if (inView(queryClient, [workId])) {
      refresh(calendarPaths);
    }
  },
  "grabs.changed": ({ workId, progress }, { queryClient, refresh }) => {
    if (!progress && inView(queryClient, [workId])) {
      refresh(calendarPaths);
    }
  },
  "targets.changed": ({ workIds }, { queryClient, refresh }) => {
    if (inView(queryClient, workIds)) {
      refresh(calendarPaths);
    }
  },
} satisfies Partial<LiveEffects>;

const both =
  <N extends ProductEventName>(own: LiveEffect<N>, more: LiveEffect<N>): LiveEffect<N> =>
  (data, ctx) => {
    own(data, ctx);
    more(data, ctx);
  };

/** The table with the Calendar's effects run after its own. */
export const withCalendar = (table: LiveEffects): LiveEffects => ({
  ...table,
  "works.changed": both(table["works.changed"], calendarEffects["works.changed"]),
  "works.removed": both(table["works.removed"], calendarEffects["works.removed"]),
  "works.type-changed": both(table["works.type-changed"], calendarEffects["works.type-changed"]),
  "works.content-types-changed": both(
    table["works.content-types-changed"],
    calendarEffects["works.content-types-changed"],
  ),
  "files.changed": both(table["files.changed"], calendarEffects["files.changed"]),
  "grabs.changed": both(table["grabs.changed"], calendarEffects["grabs.changed"]),
  "targets.changed": both(table["targets.changed"], calendarEffects["targets.changed"]),
});
