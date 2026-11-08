import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { DownloadclientsSeedingView, DownloadclientsTorrentRow } from "../api/gen/model";
import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";
import { detailOf, seedboxId, seedingOf, torrent, torrentsServer } from "../test/torrents";

const day = 86_400;
const seedpool = { id: "5e000000-0000-4000-8000-000000000001", name: "seedpool" };

// Owing seedpool 3 days 4 h past its 10-day rule and 2 h margin, on a ratio of 0.4.
const owing = (more: Partial<DownloadclientsSeedingView> = {}) =>
  seedingOf({
    ratio: { value: 0.4, goal: 2 },
    time: { value: 7 * day, rule: 10 * day + 7200, goal: 14 * day, margin: 7200 },
    state: "owing",
    owes: { seconds: 3 * day + 4 * 3600, bytes: 0 },
    clears: "time",
    ...more,
  });

const seed = (name: string, seeding: DownloadclientsSeedingView, more = {}) =>
  torrent({
    state: "seeding",
    tabs: ["seeding", "all"],
    name,
    done: 1_000_000_000,
    tracker: seedpool,
    seeding,
    ...more,
  });

const gb = 1.2 * 1024 ** 3;
const either = owing({
  ratio: { value: 0.4, rule: 1, goal: 2 },
  owes: { seconds: 3 * day + 4 * 3600, bytes: gb },
  clears: "either",
});

// Owing seedpool under no rule anyone knows, so its goal never acts.
const unruled = (endless: DownloadclientsSeedingView["endless"]) =>
  seedingOf({ ratio: { value: 0.4, goal: 2 }, state: "owing", clears: "none", endless });

const table = () => page.getByRole("table", { name: "Torrents" });
const rowOf = (name: string) => table().getByRole("row", { name: new RegExp(`^${name}`, "u") });

function open(
  rows: readonly DownloadclientsTorrentRow[],
  path = "/activity/torrents?tab=all",
  details?: Parameters<typeof torrentsServer>[1],
) {
  renderApp({
    path,
    handlers: [...liveServer().handlers, ...torrentsServer(rows, details).handlers],
  });
}

test.each([
  ["owes seeding time", owing(), "Owes 3 days 4 h of seeding"],
  ["owes either one", either, "Owes 3 days 4 h of seeding or 1.2 GB of upload"],
  [
    "owes both",
    { ...either, clears: "both" as const },
    "Owes 3 days 4 h of seeding and 1.2 GB of upload",
  ],
  [
    "seeds Forever past its rule",
    seedingOf({
      ratio: { value: 0.8 },
      time: { value: 12 * day, rule: 10 * day + 7200, margin: 7200 },
      state: "met",
      combine: "forever",
      endless: "forever",
      clears: "time",
    }),
    "Seeds with no end",
  ],
])("a seeding row that %s says so over its lanes", async (_case, seeding, line) => {
  open([seed("Harbor.2019.1080p", seeding)]);
  await expect.element(rowOf("Harbor")).toMatchTextContent(line);
  await expect.element(rowOf("Harbor").getByRole("meter", { name: "Ratio" })).toBeVisible();
  await expect.element(rowOf("Harbor").getByRole("meter", { name: "Seeding time" })).toBeVisible();
});

test("a row owing seedpool's time has a tick only on the time lane, amber while at risk", async () => {
  open([
    seed("Harbor.2019.1080p", owing()),
    seed("Quay.2020.1080p", owing({ atRisk: "yes" }), { state: "paused", hold: "wanted" }),
  ]);
  const harbor = rowOf("Harbor");
  await expect
    .element(harbor.getByRole("meter", { name: "Ratio" }))
    .toHaveAttribute("aria-valuetext", "0.4, goal 2.0");
  await expect
    .element(harbor.getByRole("meter", { name: "Seeding time" }))
    .toHaveAttribute("aria-valuetext", "7 days, rule 10 days 2 h, goal 14 days");
  await expect.element(harbor.getByText("At risk")).not.toBeInTheDocument();

  const quay = rowOf("Quay");
  await expect.element(quay).toMatchTextContent("At riskOwes 3 days 4 h of seeding");
  await expect
    .element(quay.getByRole("meter", { name: "Seeding time" }))
    .toHaveAttribute("aria-valuetext", "7 days, rule 10 days 2 h, at risk, goal 14 days");
});

test("a torrent whose goal is met shows a green Goal met chip and its full lanes", async () => {
  const met = seed(
    "Harbor.2019.1080p",
    seedingOf({
      ratio: { value: 2.3, rule: 1, goal: 2 },
      time: { value: 15 * day, goal: 14 * day },
      state: "met",
      goalMet: true,
      clears: "ratio",
    }),
    { tabs: ["seeding", "goal-met", "all"] },
  );
  open([met], "/activity/torrents?tab=goal-met");
  await expect.element(page.getByRole("radio", { name: "Goal met 1" })).toBeChecked();
  await expect.element(rowOf("Harbor")).toMatchTextContent("Goal met");
  const ratio = rowOf("Harbor").getByRole("meter", { name: "Ratio" });
  await expect.element(ratio).toHaveAttribute("aria-valuenow", "2");
  await expect.element(ratio).toHaveAttribute("aria-valuemax", "2");
});

const inspector = (name: string) => page.getByRole("complementary", { name });

test("the Inspector says what the torrent owes seedpool and what its goal does once met", async () => {
  const harbor = seed("Harbor.2019.1080p", owing());
  open([harbor], `/activity/torrents/${harbor.id}?tab=all`);
  const seeding = inspector("Harbor.2019.1080p");
  await expect
    .element(seeding)
    .toMatchTextContent(
      "Seeding" +
        "Trackerseedpool" +
        "Ratio0.4 of 2.0Seeding time7 days of 14 days" +
        "seedpool requires 10 days of seeding for each torrent. Ratio doesn't count." +
        "It still owes 3 days 4 h of seeding." +
        "This includes 2 h past seedpool's rule." +
        "Its goal is a ratio of 2.0 or 14 days of seeding, whichever comes first. Once it's met, SlipStream removes it and deletes its download copy.",
    );
  await expect
    .element(seeding.getByRole("meter", { name: "Seeding time" }))
    .toHaveAttribute("aria-valuetext", "7 days, rule 10 days 2 h, goal 14 days");
});

test.each([
  [
    "at risk",
    owing({ atRisk: "yes" }),
    "At riskIt stopped seeding before seedpool's rule was met. Seed it again before seedpool counts a hit and run.",
  ],
  [
    "on an offline Node",
    owing({ atRisk: "unknown" }),
    "seedbox is offline, so whether this torrent is seeding is unknown until seedbox reconnects.",
  ],
  [
    "under no known rule",
    unruled("no-rule"),
    "seedpool's Releases don't state a seeding rule, so SlipStream never stops this torrent. Set seedpool's rule so that its goal can act.",
  ],
  [
    "resumed after its goal paused it",
    seedingOf({ state: "met", action: "pause", endless: "resumed" }),
    "It seeds with no end, because you resumed it after its goal was met.",
  ],
])("the Inspector of a torrent %s says so", async (_case, seeding, sentence) => {
  const harbor = seed("Harbor.2019.1080p", seeding, { nodeId: seedboxId });
  open([harbor], `/activity/torrents/${harbor.id}?tab=all`);
  await expect.element(inspector("Harbor.2019.1080p")).toMatchTextContent(sentence);
});

test("the Inspector of a private torrent with no known Tracker says it never stops", async () => {
  const orphan = seed("Orphan.2018.1080p", unruled("no-tracker"), { tracker: null });
  open([orphan], `/activity/torrents/${orphan.id}?tab=all`);
  const seeding = inspector("Orphan.2018.1080p");
  await expect
    .element(seeding)
    .toMatchTextContent(
      "TrackerNot known" +
        "Ratio0.4 of 2.0Seeding time0 s of 14 days" +
        "This torrent has no known Tracker, so SlipStream never stops it. Remove it yourself once its Tracker's rule is met.",
    );
});

test("the Inspector's Tracker is a link to that Tracker's Seeding tab", async () => {
  const harbor = seed("Harbor.2019.1080p", owing());
  open([harbor], `/activity/torrents/${harbor.id}?tab=all`);
  await expect
    .element(inspector("Harbor.2019.1080p").getByRole("link", { name: "seedpool" }))
    .toHaveAttribute("href", `/settings/trackers/${seedpool.id}/seeding`);
});

const announcing = (url: string, counts: { seeders?: number; leechers?: number }) => ({
  url,
  tier: 0,
  status: "working" as const,
  answeredAt: "2026-10-12T09:00:00Z",
  ...counts,
});

test("each announce URL reads the swarm counts its last response carried, naming a private torrent's Tracker", async () => {
  const harbor = seed("Harbor.2019.1080p", owing());
  open([harbor], `/activity/torrents/${harbor.id}?tab=all`, (row) =>
    detailOf(row, {
      trackers: [
        announcing("https://tracker.seedpool.org/announce/•••", { seeders: 142, leechers: 3 }),
        announcing("https://backup.seedpool.org/announce/•••", { seeders: 1, leechers: 0 }),
        announcing("https://quiet.seedpool.org/announce/•••", {}),
      ],
    }),
  );
  const urls = page.getByRole("complementary", { name: "Harbor.2019.1080p" });
  await expect.element(urls.getByText("142 seeders and 3 leechers on seedpool.")).toBeVisible();
  await expect.element(urls.getByText("1 seeder and 0 leechers on seedpool.")).toBeVisible();
  expect(urls.getByText(/on seedpool\.$/u).all()).toHaveLength(2);
});

test("a public torrent's swarm counts name the announce host", async () => {
  const harbor = seed("Harbor.2019.1080p", owing());
  open([harbor], `/activity/torrents/${harbor.id}?tab=all`, (row) =>
    detailOf(row, {
      private: false,
      trackers: [announcing("https://open.example/announce", { seeders: 12, leechers: 40 })],
    }),
  );
  await expect.element(page.getByText("12 seeders and 40 leechers on open.example.")).toBeVisible();
});
