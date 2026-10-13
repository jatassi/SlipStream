import { http, sse } from "msw";
import { expect, test } from "vitest";
import { page } from "vitest/browser";

import type { FilesCellList } from "../../../api/gen/model";
import { getGetWorkCellsMockHandler } from "../../../api/gen/query/files/files.msw";
import {
  getGetWorkMockHandler,
  getListContentTypesMockHandler,
} from "../../../api/gen/query/works/works.msw";
import { type LiveMap, problem } from "../../../test/handlers";
import { cinema, orchard, orchardId, orchardItemId } from "../../../test/library";
import { renderApp } from "../../../test/render-app";

const hd = "1f3e5a7c-9b2d-4f6e-8a0c-2e4f6a8c0b1d";
const uhd = "7b9d1f3a-5c7e-4a9b-8d1f-3a5c7e9b1d2f";
const base = { itemId: orchardItemId, edition: "standard" } as const;

const missing: FilesCellList = {
  cells: [
    { ...base, targetId: hd, targetName: "1080p", wanted: true, state: "missing" },
    { ...base, targetId: uhd, targetName: "2160p", wanted: true, state: "missing" },
  ],
};

const filled: FilesCellList = {
  cells: [
    {
      ...base,
      targetId: hd,
      targetName: "1080p",
      wanted: false,
      state: "met",
      file: {
        id: "2a4c6e8f-0b1d-4e3f-9a5c-7e9b1d3f5a7c",
        path: "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv",
        size: 734_003_200,
        probe: { width: 1920, height: 1080 },
      },
    },
    { ...base, targetId: uhd, targetName: "2160p", wanted: true, state: "downloading" },
  ],
};

function workServer() {
  let cells = missing;
  const clients: Parameters<Parameters<typeof sse<LiveMap>>[1]>[0]["client"][] = [];
  const handlers = [
    getListContentTypesMockHandler({ contentTypes: [cinema] }),
    getGetWorkMockHandler(orchard),
    getGetWorkCellsMockHandler(() => cells),
    sse<LiveMap>("*/api/v1/events", ({ client }) => {
      clients.push(client);
      client.send({ event: "live.resync", data: {} });
    }),
  ];
  return {
    handlers,
    fileArrives(next: FilesCellList) {
      cells = next;
      clients[0]?.send({ event: "files.changed", data: { workId: orchardId } });
    },
  };
}

const row = () => page.getByRole("row", { name: /^The Quiet Orchard /u });

test("the Work page shows its fields and one cell per Target", async () => {
  renderApp({ path: `/works/${orchardId}`, handlers: workServer().handlers });
  await expect.element(page.getByRole("heading", { name: "The Quiet Orchard" })).toBeVisible();
  expect(
    page
      .getByRole("term")
      .all()
      .map((term) => term.element().textContent),
  ).toEqual(["Year", "Released", "Runtime"]);
  expect(
    page
      .getByRole("definition")
      .all()
      .map((value) => value.element().textContent),
  ).toEqual(["2021", "2021-04-09", "1h 52m"]);
  expect(
    page
      .getByRole("columnheader")
      .all()
      .map((header) => header.element().textContent),
  ).toEqual(["Film", "1080p", "2160p"]);
  await expect.element(row()).toHaveTextContent("The Quiet OrchardMissingMissing");
  await expect.element(page.getByText("No files yet.")).toBeVisible();
});

test("files.changed shows the new file without a reload", async () => {
  const server = workServer();
  renderApp({ path: `/works/${orchardId}`, handlers: server.handlers });
  await expect.element(row()).toHaveTextContent("The Quiet OrchardMissingMissing");

  server.fileArrives(filled);

  await expect
    .element(row())
    .toHaveTextContent("The Quiet OrchardMetThe Quiet Orchard (2021) - 1080p.mkvDownloading");
  await expect
    .element(page.getByRole("main").getByRole("listitem"))
    .toHaveTextContent(
      "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv 1920×1080",
    );
});

test("a Work that is gone says so", async () => {
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [
      http.get("*/api/v1/works/:workId", () =>
        problem(404, "work_not_found", "No Work has that id"),
      ),
    ],
  });
  await expect.element(page.getByRole("alert")).toHaveTextContent("No Work has that id");
});
