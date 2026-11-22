import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { calendarServer, controllersToday, orchardOpening, volumeIssue } from "../test/calendar";
import { renderApp } from "../test/render-app";

const row = (label: string) =>
  page.getByRole("listitem").filter({ has: page.getByRole("link", { name: label }) });

function listOf(rows: Parameters<typeof calendarServer>[0]) {
  const server = calendarServer(rows);
  return renderApp({
    path: `/library/calendar?view=list&from=${controllersToday}`,
    handlers: server.handlers,
  });
}

test("a row of several Items names them by level and kind, tallies their state, and opens to list them", async () => {
  const { router } = listOf({ days: [{ day: controllersToday, rows: [volumeIssue()] }] });
  const shelf = row("The Long Shelf");
  await expect.element(shelf).toMatchTextContent(/Volume 2 · Stories 1 to 10 · Issue date/u);
  await expect.element(shelf).toMatchTextContent(/Missing · 8 of 10 met/u);
  await userEvent.click(shelf.getByRole("button", { name: "Stories" }));
  const stories = shelf.getByRole("list", { name: "Stories" }).getByRole("listitem");
  await expect.element(stories.first()).toMatchTextContent("Story 1");
  expect(stories.elements()).toHaveLength(10);
  await userEvent.click(shelf.getByRole("link", { name: "The Long Shelf" }));
  await expect.poll(() => router.state.location.pathname).toBe(`/works/${volumeIssue().work.id}`);
});

test("an unmonitored row adds Not monitored, and a row with no Target shows no state", async () => {
  listOf({
    days: [
      {
        day: controllersToday,
        rows: [
          volumeIssue({ monitored: false, state: "met", met: 10 }),
          orchardOpening({ state: null, of: 0 }),
        ],
      },
    ],
  });
  const shelf = row("The Long Shelf");
  await expect.element(shelf).toMatchTextContent(/Met\s*Not monitored/u);
  expect(shelf.element().textContent).not.toMatch(/of 10 met/u);
  const orchard = row("The Quiet Orchard");
  await expect.element(orchard).toMatchTextContent(/In cinemas/u);
  expect(orchard.element().textContent).not.toMatch(/Missing|Met|Failed|Unreleased/u);
});

test("a month's cell marks an unmonitored row Not monitored and titles each row with its whole words", async () => {
  const server = calendarServer({
    days: [{ day: controllersToday, rows: [volumeIssue({ monitored: false })] }],
  });
  renderApp({ path: "/library/calendar?view=month&month=2026-11", handlers: server.handlers });
  const shelf = page.getByRole("table").getByRole("link", { name: /The Long Shelf/u });
  await expect.element(shelf).toMatchTextContent(/Missing\s*Not monitored/u);
  await expect
    .element(shelf)
    .toHaveAttribute("title", "The Long Shelf · Volume 2 · Stories 1 to 10 · Issue date");
});
