import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { GrabsBlockEntry, UnblockInputBody } from "../../../api/gen/model";
import {
  getListBlocklistMockHandler,
  getRemoveFromBlocklistMockHandler,
} from "../../../api/gen/query/grabs/grabs.msw";
import { blockedHash, blocklistPage } from "../../../test/blocklist";
import { orchardId } from "../../../test/library";
import { liveServer } from "../../../test/live-server";
import { renderApp } from "../../../test/render-app";
import { asPhone } from "../../../test/viewport";

const pageSize = 50;

function blocklistServer(entries: readonly GrabsBlockEntry[] = blocklistPage.entries) {
  let held = [...entries];
  const asked: URLSearchParams[] = [];
  const removed: UnblockInputBody[] = [];
  const live = liveServer();
  return {
    asked,
    removed,
    handlers: [
      ...live.handlers,
      getListBlocklistMockHandler(({ request }) => {
        const params = new URL(request.url).searchParams;
        asked.push(params);
        const q = params.get("q")?.toLowerCase() ?? "";
        const offset = Number(params.get("offset") ?? 0);
        const matching = held.filter((entry) => entry.title.toLowerCase().includes(q));
        return { entries: matching.slice(offset, offset + pageSize), total: matching.length };
      }),
      getRemoveFromBlocklistMockHandler(async ({ request }) => {
        // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- the request body is the operation's input
        removed.push((await request.json()) as UnblockInputBody);
      }),
    ],
    changed(next: readonly GrabsBlockEntry[]) {
      held = [...next];
      live.clients[0]?.send({ event: "grabs.blocklist-changed", data: {} });
    },
  };
}

const table = () => page.getByRole("table", { name: "Blocklist" });
const row = (title: string) => table().getByRole("row").filter({ hasText: title });
const [orchard, shelf, flux, gone] = blocklistPage.entries.map((entry) => entry.title);

test("the Blocklist tab follows Jobs and Torrents and lists each entry in words", async () => {
  renderApp({ path: "/activity/blocklist", handlers: blocklistServer().handlers });
  const views = page.getByRole("main").getByRole("navigation", { name: "Activity" });
  await expect
    .element(views.getByRole("link", { name: "Blocklist" }))
    .toHaveAttribute("aria-current", "page");
  expect(
    views
      .getByRole("link")
      .all()
      .map((link) => link.element().textContent),
  ).toEqual(["Jobs", "Torrents", "Blocklist"]);

  await expect
    .element(row(orchard ?? ""))
    .toMatchTextContent(
      "Bluray-1080p · SPARKS · 9.8 GB" +
        "The.Quiet.Orchard.2021.1080p.BluRay.x264-SPARKS" +
        "TorrentLeech reported it unregistered." +
        "TorrentLeech" +
        "The Quiet Orchard",
    );
  await expect
    .element(row(orchard ?? "").getByRole("link", { name: "The Quiet Orchard" }))
    .toHaveAttribute("href", `/works/${orchardId}`);
  await expect
    .element(row(shelf ?? ""))
    .toMatchTextContent(/Ada Okafor marked its download failed\./u);
  await expect.element(row(gone ?? "")).toMatchTextContent(/It never downloaded any data\./u);
  expect(
    row(gone ?? "")
      .getByRole("link")
      .all(),
  ).toHaveLength(0);
  await expect
    .element(row(orchard ?? "").getByRole("time"))
    .toHaveAttribute("datetime", "2026-11-02T08:30:00Z");
});

test("the Blocklist hides the Node and Content Type filters and searches by title", async () => {
  const server = blocklistServer();
  const app = renderApp({ path: "/activity/blocklist", handlers: server.handlers });
  await expect.element(row(orchard ?? "")).toBeVisible();
  await expect.element(page.getByRole("combobox", { name: "Node" })).not.toBeInTheDocument();
  await expect
    .element(page.getByRole("combobox", { name: "Content Type" }))
    .not.toBeInTheDocument();

  await userEvent.fill(page.getByRole("searchbox", { name: "Search the Blocklist" }), "2160p");
  await expect.element(row(flux ?? "")).toBeVisible();
  await expect.element(row(orchard ?? "")).not.toBeInTheDocument();
  await expect.poll(() => app.router.state.location.search).toEqual({ q: "2160p" });
  expect(server.asked.at(-1)?.get("q")).toBe("2160p");

  await userEvent.fill(page.getByRole("searchbox", { name: "Search the Blocklist" }), "nothing");
  await expect
    .element(page.getByText('No Blocklist entry has "nothing" in its title.'))
    .toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Clear search" }));
  await expect.element(row(orchard ?? "")).toBeVisible();
});

test("Remove posts the picked info hashes and the list follows the Blocklist's change", async () => {
  const server = blocklistServer();
  renderApp({ path: "/activity/blocklist", handlers: server.handlers });
  await userEvent.click(page.getByRole("checkbox", { name: `Pick ${orchard ?? ""}` }));
  await userEvent.click(page.getByRole("checkbox", { name: `Pick ${flux ?? ""}` }));
  const bar = page.getByRole("toolbar", { name: "Picked entries" });
  await expect.element(bar).toMatchTextContent(/2 Releases picked/u);

  await userEvent.click(bar.getByRole("button", { name: "Remove" }));
  await expect
    .poll(() => server.removed)
    .toEqual([
      {
        infoHashes: [blockedHash, blocklistPage.entries[2]?.infoHash],
      },
    ]);
  await expect.element(bar).not.toBeInTheDocument();
  await expect.element(row(orchard ?? "")).toBeVisible();

  server.changed(blocklistPage.entries.filter((entry) => ![orchard, flux].includes(entry.title)));
  await expect.element(row(orchard ?? "")).not.toBeInTheDocument();
  await expect.element(row(flux ?? "")).not.toBeInTheDocument();
  await expect.element(row(shelf ?? "")).toBeVisible();
});

test("Pick all picks every entry on the page", async () => {
  renderApp({ path: "/activity/blocklist", handlers: blocklistServer().handlers });
  await userEvent.click(page.getByRole("checkbox", { name: "Pick all" }));
  await expect
    .element(page.getByRole("toolbar", { name: "Picked entries" }))
    .toMatchTextContent(/4 Releases picked/u);
});

test("the Blocklist pages 50 at a time, newest first", async () => {
  const [first] = blocklistPage.entries;
  if (first === undefined) {
    throw new Error("the Blocklist fixture has entries");
  }
  const many = Array.from({ length: 60 }, (_, index) => ({
    ...first,
    infoHash: index.toString(16).padStart(40, "0"),
    title: `Entry.${String(index + 1).padStart(2, "0")}.1080p`,
  }));
  const server = blocklistServer(many);
  const app = renderApp({ path: "/activity/blocklist", handlers: server.handlers });
  const pages = page.getByRole("navigation", { name: "Pages" });
  await expect.element(pages).toMatchTextContent(/1 to 50 of 60/u);
  await expect.element(pages.getByRole("button", { name: "Newer" })).toBeDisabled();
  await expect.element(row("Entry.50.1080p")).toBeVisible();

  await userEvent.click(pages.getByRole("button", { name: "Older" }));
  await expect.element(pages).toMatchTextContent(/51 to 60 of 60/u);
  await expect.element(row("Entry.51.1080p")).toBeVisible();
  await expect.element(row("Entry.01.1080p")).not.toBeInTheDocument();
  expect(app.router.state.location.search).toEqual({ offset: 50 });
  await expect.element(pages.getByRole("button", { name: "Older" })).toBeDisabled();

  await userEvent.click(pages.getByRole("button", { name: "Newer" }));
  await expect.element(row("Entry.01.1080p")).toBeVisible();
  expect(app.router.state.location.search).toEqual({});
});

test("an empty Blocklist says what puts a Release on it", async () => {
  renderApp({ path: "/activity/blocklist", handlers: blocklistServer([]).handlers });
  await expect
    .element(page.getByText(/^The Blocklist is empty\. SlipStream adds a Release when/u))
    .toBeVisible();
  await expect.element(page.getByRole("checkbox", { name: "Pick all" })).not.toBeInTheDocument();
});

test("on a phone the Blocklist is the third segment and its rows stack", async () => {
  await asPhone();
  const app = renderApp({ path: "/activity", handlers: blocklistServer().handlers });
  const views = page.getByRole("radiogroup", { name: "View" });
  await userEvent.click(views.getByRole("radio", { name: "Blocklist" }));
  await expect.poll(() => app.router.state.location.pathname).toBe("/activity/blocklist");
  await expect.element(views.getByRole("radio", { name: "Blocklist" })).toBeChecked();
  await expect.element(row(orchard ?? "")).toBeVisible();
  await expect.element(page.getByRole("dialog")).not.toBeInTheDocument();
});
