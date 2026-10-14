import { sse } from "msw";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { GrabsGrab, GrabsGrabList, WorksListedWork } from "../../api/gen/model";
import { getListGrabsMockHandler } from "../../api/gen/query/grabs/grabs.msw";
import { getListWorksMockHandler } from "../../api/gen/query/works/works.msw";
import { downloading, imported } from "../../test/grabs";
import type { LiveMap } from "../../test/handlers";
import { orchard, orchardId } from "../../test/library";
import { renderApp } from "../../test/render-app";

const harvestId = "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b";

const harvest: WorksListedWork = {
  id: harvestId,
  contentType: "cinema",
  key: "9000002",
  label: "Orchard Harvest",
  sortLabel: "Orchard Harvest",
  addedAt: "2026-10-11T09:00:00Z",
  refreshedAt: "2026-10-11T09:00:00Z",
  fields: [],
};

const failed: GrabsGrab = {
  ...downloading,
  id: "9e1b3d5f-7a9c-4e1b-8d3f-5a7c9e1b3d5f",
  workId: harvestId,
  releaseTitle: "Orchard.Harvest.2019.720p.HDTV",
  stage: "failed",
  history: [
    { stage: "fetching", at: "2026-10-11T09:01:00Z" },
    { stage: "failed", at: "2026-10-11T09:01:01Z" },
  ],
  failure: {
    stage: "fetching",
    code: "torrent_unreachable",
    message: "the Indexer answered 404 for the .torrent",
  },
};

function activityServer() {
  let list: GrabsGrabList = { grabs: [downloading, failed] };
  const clients: Parameters<Parameters<typeof sse<LiveMap>>[1]>[0]["client"][] = [];
  const handlers = [
    getListWorksMockHandler({
      works: [
        harvest,
        {
          id: orchardId,
          contentType: orchard.contentType,
          key: orchard.key,
          label: orchard.label,
          sortLabel: orchard.sortLabel,
          addedAt: orchard.addedAt,
          refreshedAt: orchard.refreshedAt,
          fields: orchard.fields,
        },
      ],
    }),
    getListGrabsMockHandler(() => list),
    sse<LiveMap>("*/api/v1/events", ({ client }) => {
      clients.push(client);
      client.send({ event: "live.resync", data: {} });
    }),
  ];
  return {
    handlers,
    grabMoves(next: GrabsGrab) {
      list = { grabs: [next, failed] };
      clients[0]?.send({ event: "grabs.changed", data: { grabId: next.id, workId: next.workId } });
    },
  };
}

const grabs = () => page.getByRole("list", { name: "Grabs" }).getByRole("listitem");

test("Activity lists each Grab with its Work, release, stage, progress and failure", async () => {
  renderApp({ path: "/activity", handlers: activityServer().handlers });
  await expect.element(page.getByRole("heading", { name: "Activity" })).toBeVisible();

  await expect
    .element(grabs().first())
    .toHaveTextContent("The Quiet OrchardThe.Quiet.Orchard.2021.1080p.WEB-DLDownloading");
  await expect
    .element(grabs().last())
    .toHaveTextContent(
      "Orchard HarvestOrchard.Harvest.2019.720p.HDTVFailedthe Indexer answered 404 for the .torrent",
    );
  const bar = page.getByRole("progressbar", {
    name: "Downloading The.Quiet.Orchard.2021.1080p.WEB-DL",
  });
  await expect.element(bar).toHaveAttribute("value", "183500800");
  await expect.element(bar).toHaveAttribute("max", "734003200");
  expect(page.getByRole("progressbar").all()).toHaveLength(1);
  await expect
    .element(page.getByRole("link", { name: "Orchard Harvest" }))
    .toHaveAttribute("href", `/works/${harvestId}`);
});

test("grabs.changed moves a Grab to Imported without a reload", async () => {
  const server = activityServer();
  renderApp({ path: "/activity", handlers: server.handlers });
  await expect
    .element(grabs().first())
    .toHaveTextContent("The Quiet OrchardThe.Quiet.Orchard.2021.1080p.WEB-DLDownloading");

  server.grabMoves(imported);

  await expect
    .element(grabs().first())
    .toHaveTextContent("The Quiet OrchardThe.Quiet.Orchard.2021.1080p.WEB-DLImported");
  expect(page.getByRole("progressbar").all()).toHaveLength(0);
});

test("Activity with no Grabs says so", async () => {
  renderApp({
    path: "/activity",
    handlers: [getListGrabsMockHandler({ grabs: [] })],
  });
  await expect.element(page.getByText("No Grabs yet.")).toBeVisible();
});
