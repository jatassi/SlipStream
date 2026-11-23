import { afterEach, expect, test } from "vitest";
import { page } from "vitest/browser";

import type { HealthVolume } from "../api/gen/model";
import { downloading, importingPack, waitingForSeedbox } from "../test/grabs";
import { mediaLevels } from "../test/health-levels";
import { fillingVolume, healthServer, reportOf } from "../test/health";
import {
  arrival,
  bareWork,
  countsAre,
  failedGrab,
  harvestWork,
  homeServer,
  homeWork,
  job,
  need,
  nodeTorrents,
  orchardWork,
  renderHome,
  summaryOf,
  waitOf,
} from "../test/home";
import { home, nodesAre, seedbox } from "../test/nodes";
import { renderApp } from "../test/render-app";
import { asPhone, restoreViewport } from "../test/viewport";

afterEach(restoreViewport);

const gib = 1024 ** 3;
const volume = (more: Partial<HealthVolume>): HealthVolume => ({ ...mediaLevels, ...more });
const ago = (minutes: number) => new Date(Date.now() - minutes * 60_000).toISOString();

const region = (name: string) => page.getByRole("region", { name });

test("Needs you shows three failure cards, then Signal rows, waiting Callouts and pending imports", async () => {
  const failures = ["09:05", "09:04", "09:03", "09:02"].map((at, index) =>
    need(failedGrab(`2026-10-12T${at}:00Z`), homeWork(`Failed ${index + 1}`), null),
  );
  renderHome(
    summaryOf({
      needs: [...failures, need(waitingForSeedbox, bareWork, waitOf("node_away"))],
      waiting: 3,
    }),
    [countsAre({ pendingImports: 3 }), nodesAre(home, seedbox)],
  );
  const needs = page.getByRole("region", { name: "Needs you" });
  await expect.element(needs.getByRole("article", { name: "Failed 1" })).toBeVisible();
  expect(needs.getByRole("article").all()).toHaveLength(3);
  await expect.element(needs.getByRole("button", { name: /Failed 4/u })).toBeVisible();
  await expect.element(needs.getByText(/ seedbox$/u)).toBeVisible();
  await expect.element(needs.getByText(/^3 pending imports/u)).toBeVisible();
  await expect
    .element(needs.getByRole("link", { name: "See all 7 in Activity" }))
    .toHaveAttribute("href", "/activity");
});

test("each empty section shows no rows, and Downloading is left out with no Downloading Node", async () => {
  renderApp({ path: "/", handlers: homeServer(summaryOf()).handlers });
  await expect.element(region("Recent")).toMatchTextContent("90 days");
  expect(page.getByRole("table", { name: "Jobs" }).query()).toBeNull();
  expect(page.getByRole("list", { name: "Recent" }).query()).toBeNull();
  expect(region("Storage").getByRole("list").query()).toBeNull();
  await expect.element(region("Jobs")).not.toBeEmptyDOMElement();
  await expect.element(page.getByText("Everything is working.")).toBeVisible();
  await expect.element(page.getByRole("heading", { name: "Downloading" })).not.toBeInTheDocument();
});

test("Jobs keep the server's order, name the Node with two Downloading Nodes, and link to Activity", async () => {
  const pack = {
    ...importingPack,
    import: importingPack.import && { ...importingPack.import, waiting: 0 },
  };
  const server = homeServer(
    summaryOf({ jobs: [job(pack, harvestWork), job(downloading, orchardWork)], moving: 30 }),
    [
      nodeTorrents(home.id, { torrents: 1, downloading: 1 }),
      nodeTorrents(seedbox.id, { torrents: 2, seeding: 2 }),
    ],
  );
  renderApp({
    path: "/",
    handlers: [
      nodesAre(home, seedbox, { ...seedbox, id: downloading.nodeId, name: "attic" }),
      ...server.handlers,
    ],
  });
  const rows = page.getByRole("table", { name: "Jobs" }).getByRole("row");
  await expect
    .element(rows.nth(1))
    .toMatchTextContent(/^Orchard Harvest.*Importing 6 of 10 episodes/u);
  await expect.element(rows.nth(2)).toMatchTextContent(/^The Quiet Orchard.*attic/u);
  await expect
    .element(page.getByRole("link", { name: "See all 30 jobs" }))
    .toHaveAttribute("href", "/activity");
});

test("Recent words each arrival and links to its Work and to History's imports", async () => {
  renderHome(
    summaryOf({
      arrivals: [
        arrival(orchardWork, "Upgraded to Remux-2160p from WEBDL-1080p", ago(12)),
        arrival(harvestWork, "Imported 10 episodes · 2 upgrades", ago(60)),
      ],
    }),
  );
  const recent = page.getByRole("list", { name: "Recent" });
  await expect
    .element(recent.getByRole("link", { name: /^The Quiet Orchard/u }))
    .toMatchTextContent(/Upgraded to Remux-2160p from WEBDL-1080p.+12 minutes/u);
  await expect
    .element(recent.getByRole("link", { name: /^Orchard Harvest/u }))
    .toHaveAttribute("href", `/works/${harvestWork.id}`);
  await expect
    .element(page.getByRole("link", { name: "See all in History" }))
    .toHaveAttribute("href", "/activity/history?groups=%5B%22imports%22%5D");
});

test("Downloading counts each Node in Torrents' words, linking its problems", async () => {
  const server = homeServer(summaryOf(), [
    nodeTorrents(home.id, {
      torrents: 128,
      downloading: 5,
      seeding: 120,
      failures: 1,
      warnings: 1,
    }),
    nodeTorrents(seedbox.id),
  ]);
  renderApp({
    path: "/",
    handlers: [
      nodesAre(home, seedbox),
      countsAre({ downloads: { done: 5, wanted: 10, allPaused: true } }),
      ...server.handlers,
    ],
  });
  const nodes = page.getByRole("list", { name: "Downloading Nodes" });
  await expect
    .element(nodes.getByRole("listitem").first())
    .toHaveTextContent("home·5 downloading·120 seeding·1 failure·1 warning");
  await expect.element(nodes.getByRole("listitem").last()).toHaveTextContent("seedbox·no torrents");
  await expect
    .element(nodes.getByRole("link", { name: "1 warning" }))
    .toHaveAttribute("href", `/activity/torrents?tab=problems&node=${home.id}`);
  await expect.element(page.getByText("All downloads are paused.")).toBeVisible();
});

test("Health lines each category with a problem, linking to it on the Health page", async () => {
  renderHome(summaryOf(), healthServer(reportOf({ storage: [fillingVolume] })).handlers);
  await expect
    .element(
      page
        .getByRole("list", { name: "Health problems" })
        .getByRole("link", { name: "Storage · 1 warning" }),
    )
    .toHaveAttribute("href", "/health#storage");
});

test("Storage puts the worst Node and volume first, and dates a volume that isn't live", async () => {
  const health = healthServer(reportOf());
  health.state.volumes = [
    volume({ mount: "/a", name: "/a", free: 900 * gib }),
    volume({ mount: "/b", name: "/b", free: 400 * gib }),
    volume({
      nodeId: seedbox.id,
      nodeName: "seedbox",
      mount: "/c",
      name: "/c",
      level: "warn",
      live: false,
    }),
  ];
  renderHome(summaryOf(), [nodesAre(home, seedbox), ...health.handlers]);
  const lists = page.getByRole("region", { name: "Storage" }).getByRole("list");
  await expect.element(lists.first()).toHaveAccessibleName("Volumes on seedbox");
  await expect.element(lists.first()).toMatchTextContent(/^Warning\/c.* as of /u);
  await expect.element(lists.last()).toMatchTextContent(/^\/b.*\/a/u);
});

test("on phone, Home is one column in story order under a large title", async () => {
  await asPhone();
  renderHome(summaryOf({ jobs: [job(downloading, orchardWork)] }));
  await expect.element(page.getByRole("heading", { name: "Home", level: 1 })).toBeVisible();
  await expect.element(page.getByRole("heading", { name: "Storage", level: 2 })).toBeVisible();
  expect(
    page
      .getByRole("heading", { level: 2 })
      .all()
      .map((title) => title.element().textContent),
  ).toEqual(["Jobs1", "Recent", "Downloading", "Health", "Storage"]);
});
