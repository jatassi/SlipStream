import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { FilesNeed, GrabsGrab } from "../../api/gen/model";
import { getListGrabsMockHandler } from "../../api/gen/query/grabs/grabs.msw";
import { longShelf } from "../../test/anthology";
import { failedImport } from "../../test/grabs";
import { orchardId, orchardItemId } from "../../test/library";
import { renderApp } from "../../test/render-app";
import { aGroupState, aStopped, aWorkSearching } from "../../test/searching";
import { shelfCells } from "../../test/shelf-cells";
// oxlint-disable-next-line import/max-dependencies -- the rows read a many-Item Work's cells and a one-Item Work's failed Grab
import { keeper } from "../../test/targets";
import { failedCells } from "../../test/work-cells";
import { workServer } from "../../test/work-server";
import { whenWords } from "../../ui/time-words";

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
  const group = aGroupState(1, { nextSearchAt: "2026-11-15T14:00:00Z" });
  const { lastSearchedAt, nextSearchAt } = group;
  const notFound: FilesNeed = {
    kind: "not_found",
    itemIds,
    edition: "any",
    groupKey: group.key,
  };
  const server = workServer({
    work,
    startCells: { ...shelfCells(work, () => "missing"), needsYou: [notFound] },
    queue: aWorkSearching({ groups: [group] }),
    pressed: { groups: [{ key: group.key, label: group.label, itemIds, state: "queued" }] },
  });
  renderApp({ path: `/works/${work.id}`, handlers: server.handlers });

  const row = needsYou().getByRole("list", { name: "Not found" });
  await expect
    .element(row)
    .toHaveTextContent(
      `V01S01 to V01S08 not foundFound nothing ${whenWords(lastSearchedAt ?? "", "at", reading)}. SlipStream searches again ${whenWords(nextSearchAt ?? "", "next", reading)}.SearchManual search`,
    );
  await expect.element(needsYou().getByRole("heading")).toHaveTextContent("Needs you1");
  await userEvent.click(row.getByRole("button", { name: "Search", exact: true }));
  await expect.element(page.getByText("Searching 8 stories")).toBeVisible();
  expect(server.searched).toEqual([work.id]);
});

test("Not found is a quiet row on the Work page: its words alone, and the nav carries no alert", async () => {
  const group = aGroupState(1, {
    lastSearchedAt: "2026-11-14T09:12:00Z",
    nextSearchAt: null,
  });
  const notFound: FilesNeed = {
    kind: "not_found",
    itemIds: [orchardItemId],
    edition: "any",
    groupKey: group.key,
  };
  const server = workServer({
    startCells: { ...failedCells("unused"), needsYou: [notFound] },
    queue: aWorkSearching({ groups: [group] }),
  });
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  const row = needsYou().getByRole("list", { name: "Not found" });
  await expect
    .element(row)
    .toHaveTextContent(
      `The Quiet Orchard not foundFound nothing ${whenWords(group.lastSearchedAt ?? "", "at", reading)}.SearchManual search`,
    );
  expect(
    page
      .getByRole("navigation", { name: "Main" })
      .getByText(/needs attention|failed/u)
      .elements(),
  ).toHaveLength(0);
});

test("after Search again, a Grab that failed again while its search found nothing fitting reads as the hero does, listed under Grabs", async () => {
  const grab: GrabsGrab = {
    ...failedImport,
    workId: orchardId,
    items: [orchardItemId],
    failure: {
      stage: "importing",
      cause: "The download holds no video file.",
      recovery: "Search again to pick another release.",
      message: "The download has no video file.",
      current: true,
      code: "no_video_file",
      needsYou: false,
    },
  };
  const group = aGroupState(1, {
    lastResult: "nothing_acceptable",
    lastFound: 4,
    nextSearchAt: null,
  });
  const notFound: FilesNeed = {
    kind: "not_found",
    itemIds: [orchardItemId],
    edition: "any",
    groupKey: group.key,
  };
  const server = workServer({
    startCells: { ...failedCells("unused"), needsYou: [notFound] },
    queue: aWorkSearching({ groups: [group] }),
  });
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [getListGrabsMockHandler({ grabs: [grab] }), ...server.handlers],
  });
  await expect
    .element(needsYou())
    .toMatchTextContent(/The Quiet Orchard not found.+4 releases.+none fits a Target/u);
  const grabs = page.getByRole("region", { name: "Grabs" });
  await expect.element(grabs.getByRole("heading")).toHaveTextContent("Grabs1");
  expect(grabs.getByText(/under Needs you/u).elements()).toHaveLength(0);
});

test("a stopped card says a cause its failures share once", async () => {
  const cause = "The download holds no video file.";
  const grab: GrabsGrab = {
    ...failedImport,
    workId: orchardId,
    items: [orchardItemId],
    failure: {
      stage: "importing",
      code: "no_video_file",
      cause: "The download holds no video file.",
      recovery: "Search again to pick another release.",
      message: cause,
      needsYou: true,
      current: true,
    },
  };
  const stop = aStopped(1, {
    cell: { itemId: orchardItemId, edition: "any", targetId: keeper.id },
    failures: [7, 8, 9].map((hour) => ({
      code: "no_video_file" as const,
      cause: "The download holds no video file.",
      at: `2026-11-14T0${String(hour)}:00:00Z`,
      grabId: null,
    })),
  });
  const server = workServer({
    startCells: failedCells(grab.id),
    queue: aWorkSearching({ groups: [], stops: [stop] }),
  });
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [getListGrabsMockHandler({ grabs: [grab] }), ...server.handlers],
  });
  await expect.element(needsYou().getByText(cause)).toBeVisible();
  expect(needsYou().getByText(cause).elements()).toHaveLength(1);
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
        `3 Grabs failed for this file in 2 days\\. Backlog search looks for it again ${whenWords(stop.nextSearchAt ?? "", "next", reading)}\\.Failures`,
        "u",
      ),
    );
  await expect.element(card.getByRole("button", { name: "Search again" })).toBeVisible();
  await expect.element(card.getByRole("button", { name: "Manual search" })).toBeVisible();
  await expect
    .element(page.getByRole("region", { name: "Grabs" }))
    .toMatchTextContent(/^Grabs0.+Needs you/u);
});
