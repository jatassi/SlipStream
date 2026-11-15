import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { FilesNeed, GrabsGrab } from "../../api/gen/model";
import { getListGrabsMockHandler } from "../../api/gen/query/grabs/grabs.msw";
import { atWords, pastAtWords } from "../../format";
import { longShelf } from "../../test/anthology";
import { failedImport } from "../../test/grabs";
import { orchardId, orchardItemId } from "../../test/library";
import { renderApp } from "../../test/render-app";
import { aStopped, aWorkSearching } from "../../test/searching";
// oxlint-disable-next-line import/max-dependencies -- the rows read a many-Item Work's cells and a one-Item Work's failed Grab
import { shelfCells } from "../../test/shelf-cells";
import { keeper } from "../../test/targets";
import { failedCells } from "../../test/work-cells";
import { workServer } from "../../test/work-server";

const now = new Date("2026-11-14T12:00:00Z");
const reading = { now, zone: undefined };

beforeEach(() => {
  vi.useFakeTimers({ toFake: ["Date"], now });
});
afterEach(() => {
  vi.useRealTimers();
});

const needsYou = () => page.getByRole("region", { name: "Needs you" });

test("a Not found row merges the Items into a range, says when SlipStream searches again, and Search searches them", async () => {
  const work = { ...longShelf, id: crypto.randomUUID() };
  const itemIds = work.items.slice(0, 8).map(({ id }) => id);
  const lastSearchedAt = "2026-11-09T09:12:00Z";
  const nextSearchAt = "2026-11-15T14:00:00Z";
  const notFound: FilesNeed = {
    kind: "not_found",
    itemIds,
    edition: "any",
    lastSearchedAt,
    nextSearchAt,
  };
  const server = workServer({
    work,
    startCells: { ...shelfCells(work, () => "missing"), needsYou: [notFound] },
  });
  renderApp({ path: `/works/${work.id}`, handlers: server.handlers });

  const row = needsYou().getByRole("list", { name: "Not found" });
  await expect
    .element(row)
    .toHaveTextContent(
      `V01S01 to V01S08 not foundFound nothing ${pastAtWords(lastSearchedAt, reading)}. SlipStream searches again ${atWords(nextSearchAt, reading)}.SearchManual search`,
    );
  await expect.element(needsYou().getByRole("heading")).toHaveTextContent("Needs you1");
  await userEvent.click(row.getByRole("button", { name: "Search", exact: true }));
  await expect.element(page.getByText("Searching 8 stories")).toBeVisible();
  expect(server.searched).toEqual([work.id]);
});

test("Not found is a quiet row on the Work page: its words alone, and the nav carries no alert", async () => {
  const notFound: FilesNeed = {
    kind: "not_found",
    itemIds: [orchardItemId],
    edition: "any",
    lastSearchedAt: "2026-11-14T09:12:00Z",
    nextSearchAt: null,
  };
  const server = workServer({
    startCells: { ...failedCells("unused"), needsYou: [notFound] },
  });
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  const row = needsYou().getByRole("list", { name: "Not found" });
  await expect
    .element(row)
    .toHaveTextContent(
      `The Quiet Orchard not foundFound nothing ${pastAtWords(notFound.lastSearchedAt, reading)}.SearchManual search`,
    );
  expect(
    page
      .getByRole("navigation", { name: "Main" })
      .getByText(/needs attention|failed/u)
      .elements(),
  ).toHaveLength(0);
});

test("a stopped cell's card counts its failed Grabs, names Backlog search's next time and lists each failure", async () => {
  const grab: GrabsGrab = { ...failedImport, workId: orchardId, items: [orchardItemId] };
  const stop = aStopped(1, {
    cell: { itemId: orchardItemId, edition: "any", targetId: keeper.id },
  });
  const server = workServer({
    startCells: failedCells(grab.id),
    queue: aWorkSearching({ groups: [], stops: [stop] }),
  });
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [getListGrabsMockHandler({ grabs: [grab] }), ...server.handlers],
  });
  const card = needsYou();
  await expect
    .element(card)
    .toMatchTextContent(
      new RegExp(
        `3 Grabs failed here in 2 days\\. SlipStream searches again at Backlog pace, next ${atWords(stop.nextSearchAt ?? "", reading)}\\.Failures`,
        "u",
      ),
    );
  await expect.element(card.getByRole("button", { name: "Search again" })).toBeVisible();
  await expect.element(card.getByRole("button", { name: "Manual search" })).toBeVisible();
});
