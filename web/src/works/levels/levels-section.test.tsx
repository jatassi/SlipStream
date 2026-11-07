import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { FilesState, WorksItem, WorksWorkMonitored, WorksWorkView } from "../../api/gen/model";
import { longShelf } from "../../test/anthology";
import { frames } from "../../test/frames";
import { SearchWorkBody } from "../../api/gen/zod/grabs/grabs.zod";
import { SetWorkMonitoredBody } from "../../api/gen/zod/works/works.zod";
import { renderApp } from "../../test/render-app";
import { shelfCells } from "../../test/shelf-cells";
// oxlint-disable-next-line import/max-dependencies -- the rows answer Monitor and Search, so the test parses both bodies
import { workServer } from "../../test/work-server";

const volumeOf = (item: WorksItem) => longShelf.levels.findIndex(({ id }) => id === item.levelId);
const withdrawn = "2026-10-01T00:00:00Z";

// Each test opens its own Work, since a page remembers its open levels for the session.
function open(
  stateOf: (item: WorksItem) => FilesState | undefined,
  given: WorksWorkView = longShelf,
) {
  const work = { ...given, id: crypto.randomUUID() };
  const puts: WorksWorkMonitored[] = [];
  const monitored = http.put("*/api/v1/works/:workId/monitored", async ({ request }) => {
    puts.push(SetWorkMonitoredBody.parse(await request.json()));
    return new HttpResponse(null, { status: 204 });
  });
  const scopes: unknown[] = [];
  const searched = http.post("*/api/v1/works/:workId/search", async ({ request }) => {
    const scope = SearchWorkBody.parse(await request.json());
    scopes.push(scope);
    const count = scope.levelIds.length > 0 ? 49 : scope.itemIds.length;
    return HttpResponse.json(
      { searchIds: Array.from({ length: count }, () => crypto.randomUUID()) },
      { status: 202 },
    );
  });
  const server = workServer({ work, startCells: shelfCells(work, stateOf) });
  renderApp({ path: `/works/${work.id}`, handlers: [monitored, searched, ...server.handlers] });
  return { puts, scopes };
}

const levels = () => page.getByRole("region", { name: "Volumes" });
const header = (label: string) => levels().getByRole("button", { name: label, exact: true });
const drawnRows = () => document.querySelectorAll("[data-index]").length;

test("a 1,200-Item Work with every level open draws only the rows near the view", async () => {
  open(() => "missing");
  await expect.element(header("Volume 1")).toHaveAttribute("aria-expanded", "true");
  await expect.element(levels().getByRole("switch", { name: "Monitor V01S01" })).toBeVisible();
  expect(longShelf.items).toHaveLength(1200);
  expect(drawnRows()).toBeGreaterThan(10);
  expect(drawnRows()).toBeLessThan(60);
});

async function scrollTo(locator: ReturnType<typeof page.getByRole>) {
  for (let step = 0; step < 80 && locator.elements().length === 0; step++) {
    window.scrollBy(0, 400);
    // oxlint-disable-next-line no-await-in-loop -- each step scrolls from where the last one drew
    await frames(2);
  }
}

test("a met level starts closed and draws no rows until it opens", async () => {
  open((item) => (volumeOf(item) === 1 ? "missing" : "met"));
  await expect.element(header("Volume 2")).toHaveAttribute("aria-expanded", "true");
  await expect.element(header("Volume 1")).toHaveAttribute("aria-expanded", "false");
  await expect
    .element(levels().getByRole("heading", { name: "Volume 1 49 Met 49 of 49 met" }))
    .toBeVisible();
  expect(levels().getByRole("switch", { name: "Monitor V01S01" }).elements()).toHaveLength(0);
  await expect.element(levels().getByRole("switch", { name: "Monitor V02S01" })).toBeVisible();

  await userEvent.click(header("Volume 2"));
  await userEvent.click(header("Volume 1"));

  await expect.element(levels().getByRole("switch", { name: "Monitor V01S01" })).toBeVisible();
  expect(levels().getByRole("switch", { name: "Monitor V02S01" }).elements()).toHaveLength(0);
});

test("a level's header shows its bar and tally, and Items nobody monitors read Not monitored", async () => {
  open((item) => {
    if (item.aside) {
      return "missing";
    }
    return volumeOf(item) === 0 && item.position <= 6 ? "met" : "unreleased";
  });
  await expect
    .element(levels().getByRole("heading", { name: "Volume 1 6 Met, 43 Unreleased 6 of 49 met" }))
    .toBeVisible();
  const extras = levels().getByRole("heading", { name: "Extras No cells Not monitored" });
  await scrollTo(extras);
  await userEvent.click(header("Extras"));
  const extra = levels().getByRole("button", { name: "X01 1080p, Missing" });
  await scrollTo(extra);
  await expect.element(extra).toBeVisible();
  await expect.element(levels()).toMatchTextContent(/X02Story X02Not monitored/u);
});

test("a withdrawn Item with a file stays at the end of its level, and its withdrawn level trails the rest", async () => {
  const [gone] = longShelf.items;
  const volumeTwo = longShelf.levels[1];
  const work: WorksWorkView = {
    ...longShelf,
    levels: longShelf.levels.map((level) =>
      level === volumeTwo ? { ...level, withdrawnAt: withdrawn } : level,
    ),
    items: longShelf.items.map((item) =>
      item === gone || volumeOf(item) === 1 ? { ...item, withdrawnAt: withdrawn } : item,
    ),
  };
  open((item) => {
    if (item.id === gone?.id || volumeOf(item) === 1) {
      return "met";
    }
    return volumeOf(item) === 0 ? "missing" : "unreleased";
  }, work);

  const goneRow = levels().getByText("No longer listed by Fixture source").first();
  await scrollTo(goneRow);
  await expect
    .element(levels())
    .toMatchTextContent(
      /V01S49Story V01S49.*V01S01Story V01S01No longer listed by Fixture source/u,
    );
  expect(levels().getByRole("switch", { name: "Monitor V01S01" }).elements()).toHaveLength(0);
  const trailing = levels().getByRole("heading", {
    name: "Volume 2 No cells No longer listed by Fixture source",
  });
  await scrollTo(trailing);
  await expect
    .element(levels())
    .toMatchTextContent(/Not monitoredVolume 2No longer listed by Fixture source$/u);
});

test("a level's and an Item's Monitor switch send their ids and read back what the server took", async () => {
  const { puts } = open(() => "missing");
  const [volumeOne] = longShelf.levels;
  const [first] = longShelf.items;
  const level = levels().getByRole("switch", { name: "Monitor Volume 1" });
  const item = levels().getByRole("switch", { name: "Monitor V01S02" });
  await expect.element(level).toHaveAttribute("aria-checked", "true");

  await userEvent.click(levels().getByRole("switch", { name: "Monitor V01S01" }));
  await expect.element(level).toHaveAttribute("aria-checked", "mixed");
  expect(puts).toEqual([{ levelIds: [], itemIds: [first?.id], on: false }]);

  await userEvent.click(level);
  await expect.element(level).toHaveAttribute("aria-checked", "true");
  await userEvent.click(level);
  await expect.element(item).toHaveAttribute("aria-checked", "false");
  expect(puts.slice(1)).toEqual([
    { levelIds: [volumeOne?.id], itemIds: [], on: true },
    { levelIds: [volumeOne?.id], itemIds: [], on: false },
  ]);
  await expect.element(header("Volume 1")).toHaveAttribute("aria-expanded", "true");
  await expect.element(levels()).toMatchTextContent(/V01S02Story V01S02Not monitored/u);
});

test("hovering a square shows its cell, and Details shows the Item's own fields", async () => {
  open(() => "missing");
  await userEvent.hover(levels().getByRole("button", { name: "V01S02 1080p, Missing" }));
  await expect
    .element(page.getByRole("dialog", { name: "V01S02 1080p" }))
    .toHaveTextContent("V01S02 1080pMissing");

  await userEvent.click(levels().getByRole("button", { name: "Details of V01S02" }));
  await expect
    .element(page.getByRole("dialog", { name: "V01S02 Story V01S02" }))
    .toHaveTextContent(
      "V01S02 Story V01S02PublishedJanuary 3, 2020Fixture source lists this story as v1s2.",
    );
});

test("a level's Search searches its Items and an Item's Search the Item, each toasting the count", async () => {
  const { scopes } = open(() => "missing");
  const [volumeOne] = longShelf.levels;
  const [, second] = longShelf.items;

  await userEvent.click(levels().getByRole("button", { name: "Search Volume 1" }));
  await expect.element(page.getByText("Searching 49 stories")).toBeVisible();
  await userEvent.click(levels().getByRole("button", { name: "Search V01S02" }));
  await expect.element(page.getByText("Searching 1 story")).toBeVisible();
  expect(scopes).toEqual([
    { levelIds: [volumeOne?.id], itemIds: [] },
    { levelIds: [], itemIds: [second?.id] },
  ]);
});
