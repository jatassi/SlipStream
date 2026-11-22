import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { anthologies } from "../test/anthology";
import { torrentLeech } from "../test/indexers";
import { cinema } from "../test/library";
import { renderApp } from "../test/render-app";
import {
  aQueueStatus,
  aWantedPage,
  aWantedRow,
  failedQueued,
  missingRange,
  upgradeSearching,
  wantedServer,
} from "../test/wanted";

const types = getListContentTypesMockHandler({ contentTypes: [cinema, anthologies] });

function tomorrowAt(hour: number): string {
  const at = new Date();
  at.setDate(at.getDate() + 1);
  at.setHours(hour, 0, 0, 0);
  return at.toISOString();
}

const status = aQueueStatus({
  nextBacklogAt: tomorrowAt(22),
  limits: [
    {
      indexerId: torrentLeech.id,
      indexerName: torrentLeech.name,
      until: tomorrowAt(14),
      starved: false,
    },
  ],
});

const rowsOf = (count: number) =>
  Array.from({ length: count }, (_, n) =>
    aWantedRow({ work: { ...missingRange.work, id: `work-${n}`, label: `Work ${n}` } }),
  );

function openWanted(
  path = "/library/wanted",
  server = wantedServer([
    aWantedPage(
      [
        missingRange,
        failedQueued,
        { ...upgradeSearching, work: { ...upgradeSearching.work, id: "w3", label: "Ember Lane" } },
      ],
      { status },
    ),
  ]),
) {
  const app = renderApp({ path, handlers: [types, ...server.handlers] });
  return { app, server };
}

const chip = (name: string) =>
  page.getByRole("group", { name: "Reasons" }).getByRole("button", { name });

test("Wanted opens on what the queue is doing, its parts joined by dots, a limit linking to its Indexer", async () => {
  openWanted();
  const sentence = page.getByText(/^3 searching/u);
  await expect.element(sentence).toBeVisible();
  expect(sentence.element().closest("p")?.textContent).toMatch(
    /^3 searching · 142 queued · Next Backlog search tomorrow at 10:00 PM · Waiting for TorrentLeech's search limit until /u,
  );
  await expect
    .element(page.getByRole("link", { name: /^Waiting for TorrentLeech's search limit until /u }))
    .toHaveAttribute("href", `/settings/indexers/${torrentLeech.id}`);
});

test("the reason chips start on every reason but File gone, and pressing one asks again", async () => {
  const { app, server } = openWanted();
  await expect.element(chip("Failed")).toHaveAttribute("aria-pressed", "true");
  await expect.element(chip("Missing")).toHaveAttribute("aria-pressed", "true");
  await expect.element(chip("Upgrade available")).toHaveAttribute("aria-pressed", "true");
  await expect.element(chip("File gone")).toHaveAttribute("aria-pressed", "false");
  expect(server.asked.at(-1)?.get("reason")).toBe("failed,missing,upgrade");

  await userEvent.click(chip("File gone"));
  await expect.element(chip("File gone")).toHaveAttribute("aria-pressed", "true");
  await expect.poll(() => server.asked.at(-1)?.get("reason")).toBe("failed,missing,upgrade,gone");
  expect(app.router.state.location.search).toEqual({ reason: "failed,missing,upgrade,gone" });

  await userEvent.click(chip("Failed"));
  await userEvent.click(chip("Missing"));
  await userEvent.click(chip("Upgrade available"));
  await userEvent.click(chip("File gone"));
  await expect.element(chip("File gone")).toHaveAttribute("aria-pressed", "true");
});

test("the sort menu asks for the rows in the chosen order", async () => {
  const { server } = openWanted();
  await expect.element(page.getByRole("button", { name: "Sort: Next search" })).toBeVisible();
  expect(server.asked.at(-1)?.get("sort")).toBe("next");

  await userEvent.click(page.getByRole("button", { name: "Sort: Next search" }));
  await userEvent.click(page.getByRole("menuitemcheckbox", { name: "Last search" }));
  await expect.element(page.getByRole("button", { name: "Sort: Last search" })).toBeVisible();
  expect(server.asked.at(-1)?.get("sort")).toBe("last");
});

test("Search all searches the filtered list at once at 100 films", async () => {
  const server = wantedServer([aWantedPage(rowsOf(3), { total: 100 })], {
    several: { queued: 100, merged: 0, leftGone: 0, works: [] },
  });
  openWanted("/library/wanted/cinema", server);
  await userEvent.click(page.getByRole("button", { name: "Search 100 films" }));
  await expect.element(page.getByText("Queued 100 films for search.")).toBeVisible();
  expect(server.searched).toEqual([
    { wanted: { contentType: "cinema", reasons: ["failed", "missing", "upgrade"] } },
  ]);
});

test("Search all asks first above 100 Works, and Cancel sends nothing", async () => {
  const server = wantedServer([aWantedPage(rowsOf(3), { total: 1204 })], {
    several: { queued: 1192, merged: 0, leftGone: 12, works: [] },
  });
  openWanted("/library/wanted", server);
  await userEvent.click(page.getByRole("button", { name: "Search 1,204 Works" }));
  const dialog = page.getByRole("dialog", { name: "Search 1,204 Works?" });
  await expect
    .element(dialog.getByText("They wait for each Indexer's limit and can take several days."))
    .toBeVisible();
  await userEvent.click(dialog.getByRole("button", { name: "Cancel" }));
  await expect.element(dialog).not.toBeInTheDocument();
  expect(server.searched).toEqual([]);

  await userEvent.click(page.getByRole("button", { name: "Search 1,204 Works" }));
  await userEvent.click(dialog.getByRole("button", { name: "Search" }));
  await expect
    .element(
      page.getByText(
        "Queued 1,192 Works for search. 12 Works with gone files weren't queued. Search each one from its own page.",
      ),
    )
    .toBeVisible();
  expect(server.searched).toEqual([{ wanted: { reasons: ["failed", "missing", "upgrade"] } }]);
});

test("a search that queues nothing says why", async () => {
  const server = wantedServer([aWantedPage(rowsOf(2))], {
    several: { queued: 0, merged: 2, leftGone: 0, works: [] },
  });
  openWanted("/library/wanted", server);
  await userEvent.click(page.getByRole("button", { name: "Search 2 Works" }));
  await expect
    .element(
      page.getByText(
        "Nothing to search. Each of these Works is already queued or has only gone files.",
      ),
    )
    .toBeVisible();
});

test("with nothing queued there is no Cancel, and an empty list says there is nothing to search for", async () => {
  openWanted(
    "/library/wanted",
    wantedServer([
      aWantedPage([], { status: aQueueStatus({ searching: 0, queued: 0, nextBacklogAt: null }) }),
    ]),
  );
  await expect.element(page.getByText("Nothing to search for.")).toBeVisible();
  await expect.element(page.getByText(/^Nothing queued · Backlog search is off$/u)).toBeVisible();
  expect(page.getByRole("button", { name: "Cancel queued searches" }).query()).toBeNull();
  await expect.element(page.getByRole("button", { name: "Search 0 Works" })).toBeDisabled();
});

test("the headers name the Content Type's nouns, Work where types mix, and Items only for a type of many", async () => {
  openWanted("/library/wanted/anthology");
  const table = page.getByRole("table", { name: "Wanted" });
  await expect.element(table.getByRole("columnheader", { name: "Anthology" })).toBeVisible();
  await expect.element(table.getByRole("columnheader", { name: "Stories" })).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Search 3 anthologies" })).toBeVisible();
});

test("on All the Work header is Work, and a film's Wanted has no Items column", async () => {
  const { app } = openWanted();
  const table = page.getByRole("table", { name: "Wanted" });
  await expect.element(table.getByRole("columnheader", { name: "Work" })).toBeVisible();
  await expect.element(table.getByRole("columnheader", { name: "Stories" })).toBeVisible();
  await app.router.navigate({
    to: "/library/wanted/$contentType",
    params: { contentType: "cinema" },
  });
  await expect.element(table.getByRole("columnheader", { name: "Film" })).toBeVisible();
  expect(table.getByRole("columnheader", { name: "Stories" }).query()).toBeNull();
});

test("Load more reads the next page by its cursor", async () => {
  const { server } = openWanted(
    "/library/wanted",
    wantedServer([
      aWantedPage([missingRange], { next: "two", total: 2 }),
      aWantedPage([aWantedRow({ work: { ...missingRange.work, id: "w2", label: "Second Work" } })]),
    ]),
  );
  await userEvent.click(page.getByRole("button", { name: "Load more" }));
  await expect.element(page.getByRole("link", { name: "Second Work" })).toBeVisible();
  expect(server.asked.at(-1)?.get("cursor")).toBe("two");
  expect(page.getByRole("button", { name: "Load more" }).query()).toBeNull();
});

test("the Library nav row stays current on Wanted of one Content Type", async () => {
  openWanted("/library/wanted/anthology");
  const nav = page.getByRole("navigation", { name: "Main" });
  await expect.element(page.getByRole("heading", { name: "Wanted" })).toBeVisible();
  await expect
    .element(nav.getByRole("link", { name: "Library" }))
    .toHaveAttribute("aria-current", "page");
  expect(
    nav.getByRole("link", { name: "Anthologies" }).element().getAttribute("aria-current"),
  ).toBeNull();
});
