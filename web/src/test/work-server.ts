import { HttpResponse, http, sse } from "msw";

import type { FilesCellList, GrabsGrab, GrabsSearchList } from "../api/gen/model";
import { getGetWorkCellsMockHandler } from "../api/gen/query/files/files.msw";
import {
  getListGrabsMockHandler,
  getListSearchesMockHandler,
} from "../api/gen/query/grabs/grabs.msw";
import {
  getGetWorkMockHandler,
  getListContentTypesMockHandler,
} from "../api/gen/query/works/works.msw";
import { downloading, grabId, imported } from "./grabs";
import type { LiveMap } from "./handlers";
import { cinema, orchard, orchardId, orchardItemId } from "./library";

const hd = "1f3e5a7c-9b2d-4f6e-8a0c-2e4f6a8c0b1d";
const uhd = "7b9d1f3a-5c7e-4a9b-8d1f-3a5c7e9b1d2f";
const base = { itemId: orchardItemId, edition: "standard" } as const;

const missing: FilesCellList = {
  cells: [
    {
      ...base,
      targetId: hd,
      targetName: "1080p",
      wanted: true,
      state: "missing",
    },
    {
      ...base,
      targetId: uhd,
      targetName: "2160p",
      wanted: true,
      state: "missing",
    },
  ],
};

export const filled: FilesCellList = {
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
        quality: "webdl-1080p",
        probe: { width: 1920, height: 1080 },
      },
    },
    {
      ...base,
      targetId: uhd,
      targetName: "2160p",
      wanted: true,
      state: "downloading",
    },
  ],
};

export const upgradable: FilesCellList = {
  cells: [
    {
      ...base,
      targetId: hd,
      targetName: "1080p",
      wanted: false,
      state: "upgradable",
      file: {
        id: "4c6e8a0b-2d4f-4a6c-8e0b-2d4f6a8c0e1f",
        path: "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv",
        size: 524_288_000,
        quality: "hdtv-1080p",
        probe: { width: 1920, height: 1080 },
      },
    },
    {
      ...base,
      targetId: uhd,
      targetName: "2160p",
      wanted: true,
      state: "missing",
    },
  ],
};

const otherWork: GrabsGrab = {
  ...imported,
  id: "9e1b3d5f-7a9c-4e1b-8d3f-5a7c9e1b3d5f",
  workId: "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b",
  releaseTitle: "Orchard.Harvest.2019.720p.HDTV",
};

const searchId = "2b4d6f8a-0c2e-4a6c-8e0a-2c4e6a8c0e2a";
export const searching: GrabsSearchList = {
  searches: [
    {
      id: searchId,
      itemId: orchardItemId,
      indexerId: downloading.indexerId,
      startedAt: "2026-10-12T09:00:58Z",
      finishedAt: null,
      outcome: null,
      candidates: 0,
      refused: [],
      grabId: null,
      notes: [],
    },
  ],
};
export const grabbed: GrabsSearchList = {
  searches: [
    {
      id: searchId,
      itemId: orchardItemId,
      indexerId: downloading.indexerId,
      startedAt: "2026-10-12T09:00:58Z",
      finishedAt: "2026-10-12T09:01:00Z",
      outcome: "grabbed",
      candidates: 2,
      refused: [],
      grabId,
      notes: [],
    },
  ],
};

export function workServer() {
  let cells = missing;
  let searches: GrabsSearchList = { searches: [] };
  let grabs: GrabsGrab[] = [otherWork];
  const searched: string[] = [];
  const clients: Parameters<Parameters<typeof sse<LiveMap>>[1]>[0]["client"][] = [];
  const send: (typeof clients)[number]["send"] = (event) => {
    clients[0]?.send(event);
  };
  const handlers = [
    getListContentTypesMockHandler({ contentTypes: [cinema] }),
    getGetWorkMockHandler(orchard),
    getGetWorkCellsMockHandler(() => cells),
    getListSearchesMockHandler(() => searches),
    getListGrabsMockHandler(() => ({ grabs })),
    http.post("*/api/v1/works/:workId/search", ({ params }) => {
      searched.push(String(params["workId"]));
      return HttpResponse.json({ searchIds: [searchId] }, { status: 202 });
    }),
    sse<LiveMap>("*/api/v1/events", ({ client }) => {
      clients.push(client);
      client.send({ event: "live.resync", data: {} });
    }),
  ];
  return {
    handlers,
    searched,
    fileArrives(next: FilesCellList) {
      cells = next;
      send({ event: "files.changed", data: { workId: orchardId } });
    },
    searchRuns(next: GrabsSearchList) {
      searches = next;
      send({
        event: "grabs.search-changed",
        data: { searchId, workId: orchardId },
      });
    },
    grabMoves(next: GrabsGrab, nextCells = cells) {
      grabs = [next, otherWork];
      cells = nextCells;
      send({ event: "grabs.changed", data: { grabId, workId: orchardId } });
    },
  };
}
