import type { InvalidateQueryFilters } from "@tanstack/react-query";
import { QueryClientProvider, useSuspenseQuery } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { Suspense } from "react";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { CalendarAnswer, GetCalendarParams } from "../api/gen/model";
import { getGetCalendarSuspenseQueryOptions } from "../api/gen/query/calendar/calendar";
import { createQueryClient, ensureQuery } from "../api/query-client";
import { calendarDayWords, isoDay } from "../format/days";
import { calendarTypesHandler, orchardOpening, volumeIssue } from "../test/calendar";
// oxlint-disable-next-line import/max-dependencies -- the effects run against a real QueryClient, MSW and React
import { renderApp } from "../test/render-app";
import { renderKit } from "../test/render-kit";
import { worker } from "../test/worker";
import { effects } from "./effects";
import { refresh } from "./refresh";

const november: GetCalendarParams = { from: "2026-11-01", to: "2026-12-13" };
const january: GetCalendarParams = { from: "2027-01-03", to: "2027-02-14" };
const orchardId = orchardOpening().work.id;
const shelfId = volumeIssue().work.id;

const answer = (rows: CalendarAnswer["days"][number]["rows"]): CalendarAnswer => ({
  today: "2026-11-17",
  nextDayAt: new Date(Date.now() + 3_600_000).toISOString(),
  sentence: null,
  days: rows.length === 0 ? [] : [{ day: "2026-11-17", rows }],
});

function Shown() {
  const { days } = useSuspenseQuery(getGetCalendarSuspenseQueryOptions(november)).data;
  return (
    <p>{days.flatMap(({ rows }) => rows.map(({ work }) => work.label)).join(", ") || "Empty"}</p>
  );
}

/** November on screen with the Orchard in it, and January cached but not shown, with The Long Shelf. */
async function mount() {
  const reads: string[] = [];
  worker.use(
    http.get("*/api/v1/calendar", ({ request }) => {
      const from = new URL(request.url).searchParams.get("from") ?? "";
      reads.push(from);
      return HttpResponse.json(
        answer(from === november.from ? [orchardOpening()] : [volumeIssue()]),
      );
    }),
  );
  const queryClient = createQueryClient(() => {});
  await ensureQuery(queryClient, getGetCalendarSuspenseQueryOptions(january));
  const ctx = {
    queryClient,
    refresh: (filters?: InvalidateQueryFilters) => {
      void refresh(queryClient, filters);
    },
  };
  renderKit(
    <QueryClientProvider client={queryClient}>
      <Suspense fallback={<p>Loading</p>}>
        <Shown />
      </Suspense>
    </QueryClientProvider>,
  );
  await expect.element(page.getByText("The Quiet Orchard")).toBeVisible();
  reads.length = 0;
  return { reads, ctx };
}

const settle = () =>
  new Promise<void>((resolve) => {
    setTimeout(resolve, 300);
  });

test("a progress tick and a change to a Work out of view read nothing", async () => {
  const { reads, ctx } = await mount();
  effects["grabs.changed"]({ grabId: "g", workId: orchardId, progress: true }, ctx);
  effects["files.changed"]({ workId: shelfId }, ctx);
  effects["works.changed"]({ workIds: [shelfId] }, ctx);
  effects["targets.changed"]({ workIds: [shelfId] }, ctx);
  await settle();
  expect(reads).toEqual([]);
});

test("a change to a Work in view reads the shown span once", async () => {
  const { reads, ctx } = await mount();
  effects["files.changed"]({ workId: orchardId }, ctx);
  await expect.poll(() => reads).toEqual([november.from]);
});

test("a works.changed for a Work in no cached span reads, since it may now have a date in view", async () => {
  const { reads, ctx } = await mount();
  effects["works.changed"]({ workIds: ["0199e1f2-0000-7000-8000-00000000abcd"] }, ctx);
  await expect.poll(() => reads).toEqual([november.from]);
});

test("a burst of works.changed reads once, then once more after it", async () => {
  const { reads, ctx } = await mount();
  for (let index = 0; index < 6; index++) {
    effects["works.changed"]({ workIds: [orchardId] }, ctx);
  }
  await expect.poll(() => reads).toEqual([november.from, november.from]);
  await settle();
  expect(reads).toEqual([november.from, november.from]);
});

const tomorrowMarked = () =>
  page
    .getByRole("table")
    .getByRole("cell")
    .filter({ hasText: calendarDayWords(isoDay("2026-11-18"), isoDay("2026-11-17")) });

test("the page reads again when the Controller's next day starts, and Today moves", async () => {
  let asked = 0;
  worker.use(
    http.get("*/api/v1/calendar", () => {
      asked++;
      const next: CalendarAnswer = {
        ...answer([]),
        today: asked === 1 ? "2026-11-17" : "2026-11-18",
        nextDayAt: new Date(Date.now() + (asked === 1 ? 200 : 3_600_000)).toISOString(),
      };
      return HttpResponse.json(next);
    }),
  );
  renderApp({
    path: "/library/calendar?view=month&month=2026-11",
    handlers: [calendarTypesHandler()],
  });
  await expect.element(tomorrowMarked()).toHaveAttribute("aria-current", "date");
  expect(asked).toBe(2);
});
