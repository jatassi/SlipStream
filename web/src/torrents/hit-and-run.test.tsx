import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

import type { DownloadclientsPreview, DownloadclientsTorrentRow } from "../api/gen/model";
import { liveServer } from "../test/live-server";
import { renderApp } from "../test/render-app";
import { actionsServer, seedingOf, torrent, torrentsServer } from "../test/torrents";

const day = 86_400;
const seedpool = { id: "5e000000-0000-4000-8000-000000000001", name: "seedpool" };

const owingSeedpool = seedingOf({
  time: { value: 6 * day, rule: 10 * day, goal: 14 * day, margin: 0 },
  state: "owing",
  owes: { seconds: 4 * day, bytes: 0 },
  clears: "time",
});

const seed = (name: string, more: Partial<DownloadclientsTorrentRow> = {}) =>
  torrent({
    state: "seeding",
    tabs: ["seeding", "all"],
    name,
    done: 1_000_000_000,
    tracker: seedpool,
    seeding: owingSeedpool,
    ...more,
  });

const harbor = seed("Harbor.2019.1080p");
const quay = seed("Quay.2020.1080p");
const unowed = seed("Public.2021.1080p", { tracker: null, seeding: seedingOf() });

const preview = (hitAndRun: DownloadclientsPreview["hitAndRun"]): DownloadclientsPreview => ({
  cancels: [],
  stopSeeding: [],
  frees: 0,
  linked: false,
  hitAndRun,
});
const hit = (row: DownloadclientsTorrentRow, seconds: number) => ({
  torrentId: row.id,
  tracker: seedpool,
  owes: { seconds, bytes: 0 },
  clears: "time" as const,
});

const table = () => page.getByRole("table", { name: "Torrents" });
const notices = () => page.getByRole("region", { name: "Notifications" });

function open(actions: ReturnType<typeof actionsServer>) {
  renderApp({
    path: "/activity/torrents?tab=all",
    handlers: [
      actions.handler,
      ...liveServer().handlers,
      ...torrentsServer([harbor, quay, unowed]).handlers,
    ],
  });
}

async function fromMenu(name: string, action: string) {
  await userEvent.click(table().getByRole("button", { name: `${name} actions` }));
  await userEvent.click(page.getByRole("menuitem", { name: action }));
}

async function pickAll() {
  await userEvent.click(table().getByRole("checkbox", { name: "Pick Harbor.2019.1080p" }));
  await userEvent.click(table().getByRole("checkbox", { name: "Pick Public.2021.1080p" }), {
    modifiers: ["Shift"],
  });
  return page.getByRole("toolbar", { name: "Picked torrents" });
}

test("pausing a torrent that owes seedpool asks first and says what the pause costs", async () => {
  const actions = actionsServer();
  open(actions);
  await fromMenu("Harbor.2019.1080p", "Pause");
  const dialog = page.getByRole("dialog", { name: "Pause Harbor.2019.1080p?" });
  await expect
    .element(dialog)
    .toMatchTextContent(
      "seedpool counts no seeding time while it's paused. Resume it before seedpool counts a hit and run.",
    );
  expect(actions.bodies).toEqual([]);
  await userEvent.click(dialog.getByRole("button", { name: "Pause" }));
  await expect.element(notices().getByText("Pausing Harbor.2019.1080p.")).toBeVisible();
  expect(actions.bodies).toEqual([{ action: "pause", ids: [harbor.id] }]);
});

test("pausing a torrent that owes nothing acts at once", async () => {
  const actions = actionsServer();
  open(actions);
  await fromMenu("Public.2021.1080p", "Pause");
  await expect.element(notices().getByText("Pausing Public.2021.1080p.")).toBeVisible();
  expect(actions.bodies).toEqual([{ action: "pause", ids: [unowed.id] }]);
});

test("a bulk pause names the torrents that still owe seedpool", async () => {
  const actions = actionsServer();
  open(actions);
  const bar = await pickAll();
  await userEvent.click(bar.getByRole("button", { name: "Pause" }));
  const dialog = page.getByRole("dialog", { name: "Pause 3 torrents?" });
  await expect
    .element(dialog)
    .toMatchTextContent(
      "seedpool counts no seeding time for Harbor.2019.1080p and Quay.2020.1080p while they're paused. Resume them before seedpool counts a hit and run.",
    );
  await userEvent.click(dialog.getByRole("button", { name: "Pause" }));
  await expect.element(notices().getByText("Pausing 3 torrents.")).toBeVisible();
});

test("removing a torrent that owes says it counts as a hit and run, from the dry run", async () => {
  const actions = actionsServer({ preview: preview([hit(harbor, 4 * day)]) });
  open(actions);
  await fromMenu("Harbor.2019.1080p", "Remove");
  await expect
    .element(page.getByRole("dialog", { name: "Remove Harbor.2019.1080p?" }))
    .toMatchTextContent(
      "Removing it now counts as a hit and run on seedpool. It still owes 4 days of seeding.",
    );
  expect(actions.bodies).toEqual([{ action: "remove", ids: [harbor.id], dryRun: true }]);
});

test("a bulk remove lists each torrent the dry run says still owes seedpool", async () => {
  const actions = actionsServer({
    preview: preview([hit(harbor, 4 * day), hit(quay, 2 * day + 3 * 3600)]),
  });
  open(actions);
  const bar = await pickAll();
  await userEvent.click(bar.getByRole("button", { name: "Remove" }));
  await expect
    .element(page.getByRole("dialog", { name: "Remove 3 torrents?" }))
    .toMatchTextContent(
      "Removing these now counts as a hit and run on seedpool for each:" +
        "Harbor.2019.1080pOwes 4 days of seeding" +
        "Quay.2020.1080pOwes 2 days 3 h of seeding",
    );
});
