import { HttpResponse, http, sse } from "msw";
import { expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";

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
  goneUpstreamAt: null,
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
          goneUpstreamAt: null,
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

const sentence = () => page.getByRole("paragraph").filter({ hasText: "moving" });
const quietOrchard = () => page.getByRole("row", { name: /^The Quiet Orchard /u });
const harvestCard = () => page.getByRole("article", { name: "Orchard Harvest" });

test("Activity leads with what moves and what failed, then the failure, then each job", async () => {
  renderApp({ path: "/activity", handlers: activityServer().handlers });
  await expect.element(page.getByRole("heading", { name: "Activity" })).toBeVisible();

  await expect.element(sentence()).toHaveTextContent("1 job moving · 1 failure");
  await expect
    .element(page.getByRole("link", { name: "1 failure" }))
    .toHaveAttribute("href", `#failed-${failed.id}`);
  expect(harvestCard().element().closest(`#failed-${failed.id}`)).not.toBeNull();

  await expect.element(harvestCard()).toMatchTextContent(/^Failed at Downloading/u);
  await expect
    .element(harvestCard())
    .toMatchTextContent(
      "Orchard Harvest" +
        "Cause" +
        "The Indexer couldn't hand over the release's .torrent file." +
        "Recovery" +
        "Search again to pick another release." +
        "Release" +
        "Orchard.Harvest.2019.720p.HDTV" +
        "Detail" +
        "the Indexer answered 404 for the .torrent",
    );
  await expect
    .element(harvestCard().getByRole("img", { name: "Orchard Harvest failed at Downloading" }))
    .toBeVisible();

  await expect
    .element(quietOrchard())
    .toMatchTextContent(
      "The Quiet OrchardThe.Quiet.Orchard.2021.1080p.WEB-DLDownloading25%175 MB of 700 MB",
    );
  await expect
    .element(page.getByRole("link", { name: "The Quiet Orchard" }))
    .toHaveAttribute("href", `/works/${orchardId}`);
  await expect
    .element(
      page.getByRole("progressbar", { name: "Downloading The.Quiet.Orchard.2021.1080p.WEB-DL" }),
    )
    .toHaveAttribute("aria-valuenow", "25");
  await expect
    .element(page.getByRole("progressbar", { name: "Downloading The Quiet Orchard" }))
    .toHaveAttribute("aria-valuetext", "Downloading");
  expect(page.getByRole("progressbar").all()).toHaveLength(2);
});

test("grabs.changed moves a Grab to Imported without a reload", async () => {
  const server = activityServer();
  renderApp({ path: "/activity", handlers: server.handlers });
  await expect
    .element(quietOrchard())
    .toMatchTextContent("The Quiet OrchardThe.Quiet.Orchard.2021.1080p.WEB-DLDownloading");

  server.grabMoves(imported);

  await expect
    .element(quietOrchard())
    .toHaveTextContent("The Quiet OrchardThe.Quiet.Orchard.2021.1080p.WEB-DLImported");
  await expect.element(sentence()).toHaveTextContent("Nothing moving · 1 failure · 1 imported");
  expect(page.getByRole("progressbar").all()).toHaveLength(0);
});

test("Search again on a failed Grab searches its Work and says so", async () => {
  const searched: string[] = [];
  renderApp({
    path: "/activity",
    handlers: [
      http.post("*/api/v1/works/:workId/search", ({ params }) => {
        searched.push(String(params["workId"]));
        return HttpResponse.json({ searchIds: [failed.id] }, { status: 202 });
      }),
      ...activityServer().handlers,
    ],
  });
  await userEvent.click(harvestCard().getByRole("button", { name: "Search again" }));
  await expect
    .element(page.getByRole("region", { name: "Notifications" }).getByText("Search started"))
    .toBeVisible();
  expect(searched).toEqual([harvestId]);
});

test("Activity with no Grabs says what to do next", async () => {
  renderApp({
    path: "/activity",
    handlers: [getListGrabsMockHandler({ grabs: [] })],
  });
  await expect
    .element(
      page.getByText(
        "Nothing is downloading. Open a Work and choose Search to find a release for it.",
      ),
    )
    .toBeVisible();
  await expect
    .element(page.getByRole("main").getByRole("button", { name: "Open the Library" }))
    .toHaveAttribute("href", "/library");
});

test("a failure that a later Grab for its Item replaced no longer needs you", async () => {
  const retried: GrabsGrab = { ...downloading, createdAt: "2026-10-12T10:00:00Z" };
  const replaced: GrabsGrab = { ...failed, workId: orchardId, createdAt: "2026-10-12T09:00:00Z" };
  renderApp({
    path: "/activity",
    handlers: [
      getListGrabsMockHandler({ grabs: [retried, replaced] }),
      ...activityServer().handlers,
    ],
  });
  await expect.element(sentence()).toHaveTextContent("1 job moving");
  await expect.element(page.getByRole("article")).not.toBeInTheDocument();
});
