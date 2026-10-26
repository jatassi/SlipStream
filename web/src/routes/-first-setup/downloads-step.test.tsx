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
import { homeId } from "../../test/nodes";
import { renderApp } from "../../test/render-app";
import { mediaVolumes } from "../../test/roots";

const tb = 1024 ** 4;

const onVolume = (mount: string, name = mount): DownloadclientsClientCard => ({
  ...homeCard,
  downloads: { mount, name, total: 2 * tb, free: 0.5 * tb, sampledAt: "2026-10-19T14:05:00Z" },
});

function open(card: DownloadclientsClientCard, volumes: RootsNodeVolumes[]) {
  const server = firstSetup([done("roots"), todo("downloads"), done("targets")]);
  renderApp({
    path: "/setup/downloads",
    handlers: [
      ...server.handlers,
      ...downloadClients({ cards: [card], thresholds }).handlers,
      ...nodeSystems({ configs: { [homeId]: withTorrent() } }).handlers,
      ...usableFolders,
      http.get("*/api/v1/volumes", () => HttpResponse.json({ nodes: volumes })),
    ],
  });
  return server;
}

const recommendation =
  "Downloads are on a different drive from the library, so every import copies the whole file. Move the downloads folder to the library's drive to make imports instant.";

test("a Node whose downloads sit on another drive from its library roots leads with the move", async () => {
  open(onVolume("/data", "/data"), [{ ...mediaVolumes, nodeId: homeId }]);
  await expect.element(stepHeading("Downloads")).toBeVisible();
  const row = page.getByRole("listitem", { name: "home" });
  await expect.element(row.getByText(recommendation)).toBeVisible();
  await expect
    .element(row.getByRole("textbox", { name: "Downloads folder" }))
    .toHaveValue("/data/home/downloads");
  await expect.element(row.getByText("512 GB free on /data.")).toBeVisible();
});

test("downloads on the library's drive need no move", async () => {
  open(onVolume("/srv/media"), [{ ...mediaVolumes, nodeId: homeId }]);
  await expect.element(page.getByText("512 GB free on /srv/media.")).toBeVisible();
  await expect.element(page.getByText(recommendation)).not.toBeInTheDocument();
});

test("a Node with no library root gets no drive advice", async () => {
  open(onVolume("/data"), []);
  await expect.element(page.getByText("512 GB free on /data.")).toBeVisible();
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
          [homeId]: withTorrent({
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
