import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { anEntry, entriesFrom, historyServer } from "../test/history";
import { cinema } from "../test/library";
import { renderApp } from "../test/render-app";
import { sinceOf } from "./day-heading";

const types = getListContentTypesMockHandler({ contentTypes: [cinema] });

const dayMs = 86_400_000;

function openHistory(server: ReturnType<typeof historyServer>, path = "/activity/history") {
  return renderApp({ path, handlers: [types, ...server.handlers] });
}

const rows = () => page.getByRole("main").getByRole("listitem");

test("entries sit under Today, Yesterday and their weekday and date, each at its time of day", async () => {
  const now = new Date();
  const older = new Date(now.getTime() - 9 * dayMs);
  openHistory(
    historyServer([
      anEntry(3, now.toISOString()),
      anEntry(2, new Date(now.getTime() - dayMs).toISOString()),
      anEntry(1, older.toISOString()),
    ]),
  );
  for (const heading of ["Today", "Yesterday"]) {
    // oxlint-disable-next-line no-await-in-loop -- each heading is checked on the same page
    await expect.element(page.getByRole("list", { name: heading })).toBeVisible();
  }
  const weekday = older.toLocaleDateString(undefined, { weekday: "long" });
  await expect
    .element(page.getByRole("list", { name: new RegExp(`^${weekday} ${older.getDate()} `, "u") }))
    .toBeVisible();
  const today = page.getByRole("list", { name: "Today" });
  await expect.element(today).toMatchTextContent(/Grabbed The Quiet Orchard 3/u);
  await expect.element(today).toMatchTextContent(/\d{1,2}:\d{2} [AP]M/u);
});

test("Load more reads the page before the last one shown, with no entry twice", async () => {
  const server = historyServer(entriesFrom(120, 120, new Date()));
  openHistory(server);
  await expect.poll(() => rows().all().length).toBe(50);
  await userEvent.click(page.getByRole("button", { name: "Load more" }));
  await expect.poll(() => rows().all().length).toBe(100);
  await userEvent.click(page.getByRole("button", { name: "Load more" }));
  await expect.poll(() => rows().all().length).toBe(120);
  await expect.element(page.getByRole("button", { name: "Load more" })).not.toBeInTheDocument();
  expect(server.asked.map((asked) => asked.get("before"))).toEqual([null, "71", "21"]);
  const words = rows()
    .all()
    .map((row) => row.element().textContent);
  expect(new Set(words).size).toBe(120);
});

test("the filters stay in the address, and Last 7 days asks from the browser's midnight 6 days back", async () => {
  const server = historyServer([anEntry(1, new Date().toISOString())]);
  const app = openHistory(server);
  await userEvent.click(page.getByRole("button", { name: "Grabs", exact: true }));
  await userEvent.click(page.getByRole("button", { name: "Grab failed" }));
  await userEvent.click(page.getByRole("radio", { name: "Last 7 days" }));
  await expect
    .poll(() => app.router.state.location.search)
    .toEqual({ groups: ["grabs"], kinds: ["grab_failed"], since: "7d" });
  const last = () => server.asked.at(-1);
  await expect.poll(() => last()?.get("since")).toBe(sinceOf("7d", new Date()));
  expect(last()?.get("groups")).toBe("grabs");
  expect(last()?.get("kinds")).toBe("grab_failed");
});

test("with nothing matching, Clear filters lists every entry again", async () => {
  const server = historyServer([anEntry(1, new Date().toISOString())]);
  openHistory(server, "/activity/history?groups=%5B%22health%22%5D");
  await expect.element(page.getByText("No entries match these filters.")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "Clear filters" }));
  await expect.element(page.getByText("Grabbed The Quiet Orchard 1")).toBeVisible();
  expect(server.asked.at(-1)?.get("groups")).toBeNull();
});

test("an empty History says what shows there", async () => {
  openHistory(historyServer([], { sentence: null }));
  await expect
    .element(
      page.getByText(
        "History is empty. Grabs, Imports and Works added or deleted show here as they happen.",
      ),
    )
    .toBeVisible();
});

test("a Clear with filters on empties the list at once and says History is empty", async () => {
  const server = historyServer([anEntry(1, new Date().toISOString())]);
  openHistory(server, "/activity/history?groups=%5B%22grabs%22%5D");
  await expect.element(page.getByText("Grabbed The Quiet Orchard 1")).toBeVisible();
  await userEvent.click(page.getByRole("button", { name: "History options" }));
  await userEvent.click(page.getByRole("menuitem", { name: "Clear History…" }));
  await userEvent.click(page.getByRole("button", { name: "Clear History", exact: true }));
  await expect
    .element(
      page.getByText(
        "History is empty. Grabs, Imports and Works added or deleted show here as they happen.",
      ),
    )
    .toBeVisible();
  expect(server.cleared).toBe(1);
});

test("Details opens an entry in full in place, with a link to its Work and none for a deleted one", async () => {
  openHistory(
    historyServer([
      anEntry(2, new Date().toISOString()),
      anEntry(1, new Date().toISOString(), { words: "Deleted 12 Works from Movies", work: null }),
    ]),
  );
  const today = page.getByRole("list", { name: "Today" });
  const [about, deleted] = [today.getByRole("listitem").nth(0), today.getByRole("listitem").nth(1)];
  await userEvent.click(
    about.getByRole("button", { name: "Details, Grabbed The Quiet Orchard 2", exact: true }),
  );
  await expect.element(about.getByText("It replaced WEBDL-1080p.")).toBeVisible();
  await expect
    .element(about.getByRole("link", { name: "Open The Quiet Orchard" }))
    .toHaveAttribute("href", expect.stringMatching(/\/works\//u));
  await userEvent.click(
    deleted.getByRole("button", { name: "Details, Deleted 12 Works from Movies", exact: true }),
  );
  await expect.element(deleted.getByText("It replaced WEBDL-1080p.")).toBeVisible();
  await expect.element(deleted.getByRole("link")).not.toBeInTheDocument();
});

test("Details on a movie's Imported entry lists its file by quality alone", async () => {
  const imported = anEntry(1, new Date().toISOString(), {
    kind: "imported",
    group: "imports",
    words: "Imported The Quiet Orchard",
    tone: "good",
  });
  openHistory(
    historyServer([imported], {
      one: (entry) => ({
        entry,
        lines: ["WEBDL-1080p · 1.0 GB"],
        files: [
          { items: [], quality: "WEBDL-1080p", placedBy: "link", replaced: "", replacedLabel: "" },
        ],
        stages: [],
        health: [],
      }),
    }),
  );
  await userEvent.click(
    page.getByRole("button", { name: "Details, Imported The Quiet Orchard", exact: true }),
  );
  await expect
    .element(page.getByRole("main").getByRole("heading", { name: "Files" }))
    .toBeVisible();
  await expect
    .element(page.getByRole("main").getByText("WEBDL-1080p", { exact: true }))
    .toBeVisible();
});

test("the sentence leads with the failed Grabs and links them to Jobs", async () => {
  openHistory(
    historyServer([anEntry(1, new Date().toISOString())], {
      sentence: { words: "2 Grabs failed and need attention.", needsYou: 2 },
    }),
  );
  await expect
    .element(page.getByRole("link", { name: "2 Grabs failed and need attention." }))
    .toHaveAttribute("href", "/activity");
});
