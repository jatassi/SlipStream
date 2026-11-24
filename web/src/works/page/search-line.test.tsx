import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { FilesCellList, GrabsWorkSearching } from "../../api/gen/model";
import { orchardId, orchardItemId } from "../../test/library";
import { renderApp } from "../../test/render-app";
import {
  aGroupState,
  aWaiting,
  aWorkSearching,
  queuedBehind,
  searchingNow,
  searchingStarved,
} from "../../test/searching";
import { starter } from "../../test/targets";
import { missing } from "../../test/work-cells";
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

function open(searching: GrabsWorkSearching, startCells: FilesCellList = missing) {
  const server = workServer({ queue: searching, startCells });
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  return { server, line: page.getByRole("status", { name: "Search" }) };
}

const led = (group: ReturnType<typeof aGroupState>) =>
  aWorkSearching({ groups: [group], lead: { groupKey: group.key, of: 1, wait: null } });

test("a Work whose group searches reads Searching with a spinner", async () => {
  const { line } = open(led(searchingNow(1)));
  await expect.element(line).toHaveTextContent("Searching");
  expect(line.element().querySelector("svg.animate-spinner")).not.toBeNull();
});

test("a queued Work reads Queued, and its popover says what holds it", async () => {
  const { line } = open(led(queuedBehind(1)));
  await userEvent.click(line.getByRole("button", { name: "Queued" }));
  await expect
    .element(page.getByRole("dialog", { name: "Why it's queued" }))
    .toMatchTextContent(/140 Works.+Search all/u);
});

test("a searching Work whose line waits for a starved Indexer says so in its popover", async () => {
  const { line } = open(led(searchingStarved(1)));
  await userEvent.click(line.getByRole("button", { name: "Searching" }));
  await expect
    .element(page.getByRole("dialog", { name: "Why it waits" }))
    .toMatchTextContent(/waits for TorrentLeech.+RSS sync/u);
});

test("a waiting Work offers Grab now, which ends every wait of the Work", async () => {
  const wait = aWaiting(1);
  const { line, server } = open(
    aWorkSearching({ groups: [], waits: [wait], lead: { groupKey: null, of: 0, wait } }),
  );
  await expect
    .element(line)
    .toHaveTextContent(
      `Waiting for a better release ${whenWords(wait.until, "until", reading)}Grab now`,
    );
  await userEvent.click(line.getByRole("button", { name: "Grab now" }));
  await expect.element(page.getByText("Grabbing 1 release")).toBeVisible();
  expect(server.grabbed).toEqual([{}]);
});

test("a search that found Releases no Target takes offers Manual search", async () => {
  const at = "2026-11-14T09:12:00Z";
  const { line } = open(
    aWorkSearching({
      groups: [
        aGroupState(1, { lastResult: "nothing_acceptable", lastFound: 14, lastSearchedAt: at }),
      ],
    }),
  );
  await expect
    .element(line)
    .toMatchTextContent(
      /^Found 14 releases, but none fits a Target\. Next search .+Manual search$/u,
    );
});

test("a search no Indexer could run links to Health", async () => {
  const { line } = open(aWorkSearching({ groups: [aGroupState(1, { lastResult: "no_indexer" })] }));
  await expect
    .element(line.getByRole("link", { name: "Open Health" }))
    .toHaveAttribute("href", "/health");
});

test("the line follows the queue as it moves", async () => {
  const at = "2026-11-14T09:12:00Z";
  const { line, server } = open(
    aWorkSearching({ groups: [aGroupState(1, { lastSearchedAt: at, nextSearchAt: null })] }),
  );
  await expect.element(line).toHaveTextContent(`Found nothing ${whenWords(at, "at", reading)}.`);
  server.queueMoves(led(searchingNow(1)));
  await expect.element(line).toHaveTextContent("Searching");
});

test("a Work nobody searched shows no line", async () => {
  open(aWorkSearching({ groups: [] }));
  await expect.element(page.getByRole("button", { name: "Search", exact: true })).toBeVisible();
  expect(page.getByRole("status", { name: "Search" }).elements()).toHaveLength(0);
});

test("a waiting cell says how many Releases it holds, and its Grab now ends that wait", async () => {
  const until = "2026-11-14T22:40:00Z";
  const [first, second] = missing.cells;
  if (first === undefined || second === undefined) {
    throw new Error("missing has two cells");
  }
  const { server } = open(aWorkSearching({ groups: [] }), {
    ...missing,
    cells: [{ ...first, waitUntil: until, candidates: 2 }, second],
  });
  await expect
    .element(
      page.getByText(
        `2 releases found so far. SlipStream grabs the best one ${whenWords(until, "next", reading)} unless a better one arrives.`,
      ),
    )
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Grab now" }));
  await expect
    .poll(() => server.grabbed)
    .toEqual([{ cells: [{ itemId: orchardItemId, edition: "any", targetId: starter.id }] }]);
});

test("a cell whose file is gone says so", async () => {
  const [first, second] = missing.cells;
  if (first === undefined || second === undefined) {
    throw new Error("missing has two cells");
  }
  open(aWorkSearching({ groups: [] }), { ...missing, cells: [{ ...first, gone: true }, second] });
  await expect.element(page.getByText("File gone")).toBeVisible();
});
