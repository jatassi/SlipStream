import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import {
  addDays,
  betweenWords,
  calendarDayWords,
  daysBetween,
  isoDay,
  monthSpan,
  weekStart,
  weekdayNames,
} from "../format/days";
import { calendarServer, controllersToday, orchardOpening, volumeIssue } from "../test/calendar";
import { renderApp } from "../test/render-app";
import { asPhone } from "../test/viewport";

const today = isoDay(controllersToday);

const spans = (asked: readonly URLSearchParams[]) =>
  asked.map((params) =>
    daysBetween(isoDay(params.get("from") ?? ""), isoDay(params.get("to") ?? "")),
  );

test("the month starts on the locale's first weekday, marks the Controller's Today and reads one six-week span", async () => {
  const server = calendarServer({ days: [{ day: controllersToday, rows: [orchardOpening()] }] });
  renderApp({ path: "/library/calendar?view=month&month=2026-11", handlers: server.handlers });
  const grid = page.getByRole("table", { name: "November 2026" });
  const { from } = monthSpan({ year: 2026, month: 11 }, weekStart());
  await expect
    .element(grid.getByRole("columnheader").first())
    .toHaveTextContent(weekdayNames(from).short);
  expect(from.slice(5)).toBe(weekStart() === 1 ? "10-26" : "11-01");
  const marked = grid.getByRole("cell").filter({ hasText: calendarDayWords(today) });
  await expect.element(marked).toHaveAttribute("aria-current", "date");
  await expect.element(marked.getByRole("link", { name: /The Quiet Orchard/u })).toBeVisible();
  expect(server.asked.map((params) => [params.get("from"), params.get("to")])).toEqual([
    [from, addDays(from, 42)],
  ]);
});

test("the week starts on Monday when the browser can't say", () => {
  const original = Intl.Locale;
  // A Locale with no week info, as an engine without it has.
  class Unsaid extends original {
    constructor(tag: string) {
      super(tag);
      Object.defineProperty(this, "getWeekInfo", { value: undefined });
      Object.defineProperty(this, "weekInfo", { value: undefined });
    }
  }
  Object.defineProperty(Intl, "Locale", { value: Unsaid, configurable: true });
  try {
    expect(weekStart("en-US")).toBe(1);
  } finally {
    Object.defineProperty(Intl, "Locale", { value: original, configurable: true });
  }
  expect(weekStart("en-US")).toBe(0);
  expect(weekStart("en-GB")).toBe(1);
});

test("a month left out of the address moves to the Controller's month", async () => {
  const server = calendarServer({ today: "2031-03-04" });
  const { router } = renderApp({ path: "/library/calendar?view=month", handlers: server.handlers });
  await expect.poll(() => router.state.location.search).toMatchObject({ month: "2031-03" });
  await expect.element(page.getByRole("heading", { name: "March 2031" })).toBeVisible();
  await expect
    .element(page.getByText("Nothing in your library comes out in March 2031."))
    .toBeVisible();
});

test("a day with more rows than its cell fits shows +3 more, which opens the whole day", async () => {
  const rows = Array.from({ length: 5 }, (_, index) =>
    orchardOpening({
      work: {
        ...orchardOpening().work,
        id: `${String(index)}-orchard`,
        label: `Orchard ${String(index + 1)}`,
      },
    }),
  );
  const server = calendarServer({ days: [{ day: controllersToday, rows }] });
  renderApp({ path: "/library/calendar?view=month&month=2026-11", handlers: server.handlers });
  await userEvent.click(page.getByRole("button", { name: "+3 more" }));
  const day = page.getByRole("dialog", { name: calendarDayWords(today) });
  await expect.element(day.getByRole("link", { name: "Orchard 5" })).toBeVisible();
  expect(day.getByRole("listitem").elements()).toHaveLength(5);
});

test("a phone opens the list, and its month shows one mark a day and the tapped day's rows", async () => {
  await asPhone();
  const server = calendarServer({
    days: [{ day: "2026-11-20", rows: [volumeIssue()] }],
  });
  const { router } = renderApp({ path: "/library/calendar", handlers: server.handlers });
  await expect.poll(() => router.state.location.search).toMatchObject({ view: "list" });
  await expect.element(page.getByRole("link", { name: "The Long Shelf" })).toBeVisible();

  await userEvent.click(page.getByRole("link", { name: "Month" }));
  const friday = isoDay("2026-11-20");
  const mark = page.getByRole("button", { name: `${calendarDayWords(friday)} Missing` });
  await expect.element(mark).toBeVisible();
  await expect
    .element(
      page
        .getByRole("region", { name: calendarDayWords(today) })
        .getByText("Nothing in your library comes out on this day."),
    )
    .toBeVisible();
  await userEvent.click(mark);
  await expect
    .element(
      page
        .getByRole("region", { name: calendarDayWords(friday) })
        .getByRole("link", { name: "The Long Shelf" }),
    )
    .toBeVisible();
});

test("the list pages six weeks at a time and names its range when empty", async () => {
  const server = calendarServer();
  const { router } = renderApp({
    path: "/library/calendar?view=list&from=2026-11-14",
    handlers: server.handlers,
  });
  const first = isoDay("2026-11-14");
  await expect
    .element(
      page.getByText(
        `Nothing in your library comes out between ${betweenWords(first, addDays(first, 41))}.`,
      ),
    )
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Show later" }));
  await expect.poll(() => router.state.location.search).toMatchObject({ from: "2026-12-26" });
  await userEvent.click(page.getByRole("button", { name: "Show earlier" }));
  await userEvent.click(page.getByRole("button", { name: "Show earlier" }));
  await expect.poll(() => router.state.location.search).toMatchObject({ from: "2026-10-03" });
  expect(server.asked.map((params) => params.get("from"))).toEqual([
    "2026-11-14",
    "2026-12-26",
    "2026-10-03",
  ]);
  expect(Math.max(...spans(server.asked))).toBe(42);
});

test("the address keeps the kinds and Show unmonitored, and each request asks for them", async () => {
  const server = calendarServer();
  const { router } = renderApp({
    path: "/library/calendar?view=month&month=2026-11",
    handlers: server.handlers,
  });
  await userEvent.click(page.getByRole("button", { name: "In cinemas" }));
  await expect.poll(() => router.state.location.search).toMatchObject({ kinds: "issued" });
  await userEvent.click(page.getByRole("switch", { name: "Show unmonitored" }));
  await expect
    .poll(() => router.state.location.search)
    .toMatchObject({ kinds: "issued", unmonitored: true, month: "2026-11", view: "month" });
  await expect
    .poll(() => server.asked.at(-1)?.toString())
    .toMatch(/kinds=issued&unmonitored=true$/u);
  await userEvent.click(page.getByRole("button", { name: "Next month" }));
  await expect
    .poll(() => router.state.location.search)
    .toMatchObject({ month: "2026-12", kinds: "issued" });
  expect(Math.max(...spans(server.asked))).toBeLessThanOrEqual(42);
});

test("what needs attention leads the page and links to Wanted", async () => {
  const server = calendarServer({
    days: [{ day: controllersToday, rows: [orchardOpening()] }],
    sentence: { needsYou: true, words: "1 film is out and still missing." },
  });
  renderApp({
    path: "/library/calendar/cinema?view=list&from=2026-11-14",
    handlers: server.handlers,
  });
  await expect
    .element(page.getByRole("link", { name: "1 film is out and still missing." }))
    .toHaveAttribute("href", "/library/wanted/cinema");
  expect(server.asked[0]?.get("contentType")).toBe("cinema");
});
