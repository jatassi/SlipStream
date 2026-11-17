import { http, HttpResponse } from "msw";

import type { CalendarAnswer, CalendarDay, CalendarRow, WorksContentType } from "../api/gen/model";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { anthologies, anthologyId } from "./anthology";
import { cinema, orchard, orchardItemId } from "./library";

/** Anthologies with an issue date, so the Calendar has two dated kinds. */
export const datedAnthologies: WorksContentType = {
  ...anthologies,
  dated: [{ key: "issued", label: "Issue date" }],
};

export const calendarTypes = [cinema, datedAnthologies] as const;

/** The Quiet Orchard in cinemas, missing. */
export const orchardOpening = (over: Partial<CalendarRow> = {}): CalendarRow => ({
  work: { id: orchard.id, contentType: cinema.id, label: orchard.label, posterUrl: null },
  kind: "opening",
  kindLabel: "In cinemas",
  items: [{ id: orchardItemId, label: orchard.label }],
  itemsWords: "",
  state: "missing",
  met: 0,
  of: 1,
  monitored: true,
  ...over,
});

const stories = (count: number) =>
  Array.from({ length: count }, (_, index) => ({
    id: `9b1d3f5a-7c9e-4b1d-8f3a-5c7e9b1d3f${String(index).padStart(2, "0")}`,
    label: `Story ${String(index + 1)}`,
  }));

/** Volume 2 of The Long Shelf, ten Stories issued on one day, 8 of them met. */
export const volumeIssue = (over: Partial<CalendarRow> = {}): CalendarRow => ({
  work: {
    id: anthologyId,
    contentType: datedAnthologies.id,
    label: "The Long Shelf",
    posterUrl: null,
  },
  kind: "issued",
  kindLabel: "Issue date",
  items: stories(10),
  itemsWords: "Volume 2 · Stories 1 to 10",
  state: "missing",
  met: 8,
  of: 10,
  monitored: true,
  ...over,
});

export const calendarTypesHandler = () =>
  getListContentTypesMockHandler({ contentTypes: [...calendarTypes] });

/** The Controller's day the answers name, a Tuesday. */
export const controllersToday = "2026-11-17";

/**
 * Serves GET /calendar from `days`, each answer holding the days inside the
 * span it was asked for, and records every request's parameters in `asked`.
 */
export function calendarServer({
  days = [],
  today = controllersToday,
  sentence = null,
}: {
  days?: readonly CalendarDay[];
  today?: string;
  sentence?: CalendarAnswer["sentence"];
} = {}) {
  const asked: URLSearchParams[] = [];
  return {
    asked,
    handlers: [
      calendarTypesHandler(),
      http.get("*/api/v1/calendar", ({ request }) => {
        const params = new URL(request.url).searchParams;
        asked.push(params);
        const from = params.get("from") ?? "";
        const to = params.get("to") ?? "";
        const answer: CalendarAnswer = {
          today,
          nextDayAt: new Date(Date.now() + 3_600_000).toISOString(),
          sentence,
          days: days.filter(({ day }) => day >= from && day < to),
        };
        return HttpResponse.json(answer);
      }),
    ],
  };
}
