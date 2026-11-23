import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { longShelf } from "../../test/anthology";
import { renderApp } from "../../test/render-app";
import {
  aCell,
  aGroupState,
  aWaiting,
  aWorkSearching,
  queuedBehind,
  searchingNow,
} from "../../test/searching";
import { shelfCells } from "../../test/shelf-cells";
import { asPhone } from "../../test/viewport";
import { workServer } from "../../test/work-server";

const levels = () => page.getByRole("region", { name: "Volumes" });

function open() {
  const work = { ...longShelf, id: crypto.randomUUID() };
  const waiting = aCell(3);
  // Only the waiting Story is missing, so every level but its own stays closed near the top.
  const cells = shelfCells(work, (item) => (item.id === waiting.itemId ? "missing" : "met"));
  for (const cell of cells.cells) {
    if (cell.itemId === waiting.itemId) {
      Object.assign(cell, { waitUntil: "2026-11-14T18:30:00Z", candidates: 2 });
    }
  }
  const server = workServer({
    work,
    startCells: cells,
    queue: aWorkSearching({
      waits: [aWaiting(3)],
      groups: longShelf.levels
        .filter(({ aside }) => !aside)
        .map((_, i) => {
          const n = i + 1;
          return n === 1 ? searchingNow(n) : n === 2 ? queuedBehind(n) : aGroupState(n);
        }),
    }),
  });
  renderApp({ path: `/works/${work.id}`, handlers: server.handlers });
  return { server, waiting };
}

test("a searching level's header reads Searching, and its Search shows the spinner", async () => {
  open();
  await expect
    .element(levels().getByRole("button", { name: "Search Volume 1" }))
    .toHaveAttribute("aria-busy", "true");
  await expect
    .element(levels().getByRole("button", { name: "Search Volume 2" }))
    .not.toHaveAttribute("aria-busy");
  await expect.element(levels().getByText("Searching", { exact: true })).toBeVisible();
});

test("a queued level's state word opens what holds it", async () => {
  open();
  await userEvent.hover(levels().getByRole("button", { name: "Queued", exact: true }));
  await expect
    .element(page.getByRole("dialog", { name: "Queued" }))
    .toMatchTextContent(/140 Works.+Search all/u);
});

test("a level with a waiting cell reads Waiting, and its menu's Grab now ends that wait", async () => {
  const { server, waiting } = open();
  await expect
    .element(levels().getByRole("button", { name: "Waiting", exact: true }))
    .toBeVisible();
  await userEvent.click(levels().getByRole("button", { name: "More actions for Volume 3" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Grab now" }));
  await expect.poll(() => server.grabbed).toEqual([{ cells: [waiting] }]);
  await expect.element(page.getByText("Grabbing 1 release")).toBeVisible();
});

test("a level with nothing waiting has no Grab now", async () => {
  open();
  await userEvent.click(levels().getByRole("button", { name: "More actions for Volume 2" }));
  await expect.element(page.getByRole("menuitem", { name: "Manual search" })).toBeVisible();
  expect(page.getByRole("menuitem", { name: "Grab now" }).elements()).toHaveLength(0);
});

test("on a phone a level's tally reads whole beside its state word", async () => {
  await asPhone();
  open();
  const header = levels().getByRole("heading", { name: /Volume 1/u });
  await expect.element(header.getByText("Searching", { exact: true })).toBeVisible();
  const tally = header.getByText(/^\d+ of \d+ met$/u).element();
  expect(tally.scrollWidth).toBeLessThanOrEqual(tally.clientWidth);
});
