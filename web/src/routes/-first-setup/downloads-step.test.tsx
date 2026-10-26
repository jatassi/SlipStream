import { HttpResponse, http } from "msw";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { DownloadclientsClientCard, RootsNodeVolumes } from "../../api/gen/model";
import {
  downloadClients,
  homeCard,
  thresholds,
  usableFolders,
  withTorrent,
} from "../../test/download-clients";
import { done, firstSetup, stepHeading, todo } from "../../test/first-setup";
import { nodeSystems } from "../../test/node-system";
import { problem } from "../../test/handlers";
import { renderApp } from "../../test/render-app";
import { mediaVolumes } from "../../test/roots";

const tb = 1024 ** 4;

const onVolume = (mount: string, name = mount): DownloadclientsClientCard => ({
  ...homeCard,
  downloads: { mount, name, total: 2 * tb, free: 0.5 * tb, sampledAt: "2026-10-19T14:05:00Z" },
});

function open(
  card: DownloadclientsClientCard,
  volumes: RootsNodeVolumes[],
  config = nodeSystems({ configs: { [homeCard.nodeId]: withTorrent() } }).handlers,
) {
  const server = firstSetup([done("roots"), todo("downloads"), done("targets")]);
  renderApp({
    path: "/setup/downloads",
    handlers: [
      ...server.handlers,
      ...downloadClients({ cards: [card], thresholds }).handlers,
      ...config,
      ...usableFolders,
      http.get("*/api/v1/volumes", () => HttpResponse.json({ nodes: volumes })),
    ],
  });
  return server;
}

const recommendation =
  "Downloads are on a different drive from the library, so every import copies the whole file. Move the downloads folder to the library's drive to make imports instant.";

test("a Node whose downloads sit on another drive from its library roots leads with the move", async () => {
  open(onVolume("/data", "/data"), [{ ...mediaVolumes, nodeId: homeCard.nodeId }]);
  await expect.element(stepHeading("Downloads")).toBeVisible();
  const row = page.getByRole("listitem", { name: "home" });
  await expect.element(row.getByText(recommendation)).toBeVisible();
  await expect
    .element(row.getByRole("textbox", { name: "Downloads folder" }))
    .toHaveValue("/data/home/downloads");
  await expect.element(row.getByText("/data has 512 GB free.")).toBeVisible();
});

test("downloads on the library's drive need no move", async () => {
  open(onVolume("/srv/media"), [{ ...mediaVolumes, nodeId: homeCard.nodeId }]);
  await expect.element(page.getByText("/srv/media has 512 GB free.")).toBeVisible();
  await expect.element(page.getByText(recommendation)).not.toBeInTheDocument();
});

test("a Node with no library root gets no drive advice", async () => {
  open(onVolume("/data"), []);
  await expect.element(page.getByText("/data has 512 GB free.")).toBeVisible();
  await expect.element(page.getByText(recommendation)).not.toBeInTheDocument();
});

test("a downloads folder saved but not yet in use offers the Node's restart", async () => {
  const server = firstSetup([todo("downloads")]);
  renderApp({
    path: "/setup/downloads",
    handlers: [
      ...server.handlers,
      ...downloadClients({ cards: [homeCard], thresholds }).handlers,
      ...nodeSystems({
        configs: {
          [homeCard.nodeId]: withTorrent({
            downloads: { value: "/srv/media/downloads", pendingRestart: true },
          }),
        },
      }).handlers,
      ...usableFolders,
      http.get("*/api/v1/volumes", () => HttpResponse.json({ nodes: [] })),
    ],
  });
  await expect
    .element(page.getByText("Restart home to use the new downloads folder"))
    .toBeVisible();
  await expect.element(page.getByRole("button", { name: "Restart now" })).toBeVisible();
});

test("a Node whose settings don't load costs only its row, and the step can still be skipped", async () => {
  let failing = true;
  const working = nodeSystems({ configs: { [homeCard.nodeId]: withTorrent() } }).handlers;
  const server = open(onVolume("/data"), [], [
    http.get("*/api/v1/nodes/:id/config", () =>
      failing ? problem(500, "internal", "SlipStream couldn't read the settings") : undefined,
    ),
    ...working,
  ]);
  const row = page.getByRole("listitem", { name: "home" });
  await expect
    .element(
      row.getByText(
        "home's downloads folder didn't load. Try again, or skip this step and set it later in Nodes.",
      ),
    )
    .toBeVisible();
  await expect.element(stepHeading("Downloads")).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Back" })).toBeVisible();
  await expect.element(page.getByRole("button", { name: "Finish setup" })).toBeVisible();

  failing = false;
  await row.getByRole("button", { name: "Try again" }).click();
  await expect
    .element(row.getByRole("textbox", { name: "Downloads folder" }))
    .toHaveValue("/data/home/downloads");

  await page.getByRole("button", { name: "Skip" }).click();
  await expect.element(stepHeading("What SlipStream grabs")).toBeVisible();
  expect(server.writes).toEqual([{ id: "downloads", outcome: "skipped" }]);
});

test("an offline Node's row says it shows its folder once it reconnects", async () => {
  open(onVolume("/data"), [], [
    http.get("*/api/v1/nodes/:id/config", () => problem(503, "node_offline", "The Node is offline")),
  ]);
  await expect
    .element(page.getByText("home is offline. Its downloads folder shows here when it reconnects."))
    .toBeVisible();
  await expect.element(page.getByRole("button", { name: "Skip" })).toBeVisible();
});
