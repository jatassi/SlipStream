import { HttpResponse, http, sse } from "msw";

import type {
  FilesCellList,
  GrabsGrab,
  GrabsSearch,
  GrabsSearchIndexer,
  GrabsSearchList,
  TargetsWants,
} from "../api/gen/model";
import { getGetWorkCellsMockHandler } from "../api/gen/query/files/files.msw";
import {
  getListGrabsMockHandler,
  getListSearchesMockHandler,
} from "../api/gen/query/grabs/grabs.msw";
import { getGetWorkWantsMockHandler } from "../api/gen/query/targets/targets.msw";
import {
  getGetWorkMockHandler,
  getListContentTypesMockHandler,
} from "../api/gen/query/works/works.msw";
import { grabId, imported, indexerId } from "./grabs";
import type { LiveMap } from "./handlers";
import { cinema, orchard, orchardId, orchardItemId } from "./library";
import { home, nodesFrom, seedbox } from "./nodes";
// oxlint-disable-next-line import/max-dependencies -- the Work page reads every section's queries, the Nodes' and the Profiles' included
import { films, profileHandlers } from "./profiles";

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
  items: [{ itemId: orchardItemId, status: "missing" }],
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
  items: [{ itemId: orchardItemId, status: "downloading" }],
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
  items: [{ itemId: orchardItemId, status: "missing" }],
};

const otherWork: GrabsGrab = {
  ...imported,
  id: "9e1b3d5f-7a9c-4e1b-8d3f-5a7c9e1b3d5f",
  workId: "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b",
  releaseTitle: "Orchard.Harvest.2019.720p.HDTV",
};

const searchId = "2b4d6f8a-0c2e-4a6c-8e0a-2c4e6a8c0e2a";
const mirrorId = "8a0c2e4f-6b8d-4f0a-9c2e-4f6a8b0d2e4f";

const run: Pick<GrabsSearch, "id" | "itemId" | "startedAt" | "refused" | "notes"> = {
  id: searchId,
  itemId: orchardItemId,
  startedAt: "2026-10-12T09:00:58Z",
  refused: [],
  notes: [],
};

const line = (
  name: string,
  id: string | null,
  sentence: string,
  outcome: GrabsSearchIndexer["outcome"],
  releases = 0,
): GrabsSearchIndexer => ({
  indexerId: id,
  name,
  outcome,
  releases,
  sentence,
});

export const searching: GrabsSearchList = {
  searches: [
    {
      ...run,
      indexers: [
        line("Fixture Mirror", mirrorId, "Searching", null),
        line("Fixture Site", indexerId, "Searching", null),
      ],
      finishedAt: null,
      outcome: null,
      candidates: 0,
      grabId: null,
    },
  ],
};
export const grabbed: GrabsSearchList = {
  searches: [
    {
      ...run,
      indexers: [
        line("Fixture Mirror", mirrorId, "Found 2 Releases", "found", 2),
        line("Fixture Site", indexerId, "Found 2 Releases", "found", 2),
      ],
      finishedAt: "2026-10-12T09:01:00Z",
      outcome: "grabbed",
      candidates: 2,
      grabId,
    },
  ],
};

/** A run whose every line ended another way, with the server's sentences. */
export const everyOutcome: GrabsSearchList = {
  searches: [
    {
      ...run,
      indexers: [
        line("Alder", "0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d", "Found 1 Release", "found", 1),
        line("Birch", "1b2c3d4e-5f6a-4b7c-9d8e-0f1a2b3c4d5e", "No Releases", "no_releases"),
        line(
          "Cedar",
          "2c3d4e5f-6a7b-4c8d-8e9f-1a2b3c4d5e6f",
          "Limit reached until 14:05",
          "limited",
        ),
        line(
          "Dune",
          "3d4e5f6a-7b8c-4d9e-9f0a-2b3c4d5e6f7a",
          "Limit reached until 15:00",
          "limit_reached",
        ),
        line("Elm", "4e5f6a7b-8c9d-4e0f-8a1b-3c4d5e6f7a8b", "Waiting for seedbox", "waiting"),
        line(
          "Fir",
          "5f6a7b8c-9d0e-4f1a-9b2c-4d5e6f7a8b9c",
          "Fir isn't responding. SlipStream retries at 14:20.",
          "failing",
        ),
        line(
          "Gale",
          "6a7b8c9d-0e1f-4a2b-8c3d-5e6f7a8b9c0d",
          "Stopped because Gale rejected the username or password.",
          "stopped",
        ),
        line(
          "Hazel",
          "7b8c9d0e-1f2a-4b3c-9d4e-6f7a8b9c0d1e",
          "Skipped because it has no movie categories",
          "no_categories",
        ),
        line("Ivy", "8c9d0e1f-2a3b-4c4d-8e5f-7a8b9c0d1e2f", "Didn't finish in time", "late"),
        line("Juniper", null, "Deleted since this search", "removed"),
      ],
      finishedAt: "2026-10-12T09:01:00Z",
      outcome: "nothing_acceptable",
      candidates: 1,
      grabId: null,
    },
  ],
};

/** The Quiet Orchard wants what Films wants, Edition list included. */
export const fromFilms: TargetsWants = {
  profileId: films.id,
  profileName: films.name,
  editions: films.editions,
  from: "profile",
};

const fixedHandlers = () => [
  getListContentTypesMockHandler({ contentTypes: [cinema] }),
  getGetWorkMockHandler(orchard),
  getGetWorkWantsMockHandler(fromFilms),
  ...profileHandlers(),
];

type LiveClient = Parameters<Parameters<typeof sse<LiveMap>>[1]>[0]["client"];

/** The events stream, and a send to its first client. */
function liveChannel() {
  const clients: LiveClient[] = [];
  const send: LiveClient["send"] = (event) => {
    clients[0]?.send(event);
  };
  const handler = sse<LiveMap>("*/api/v1/events", ({ client }) => {
    clients.push(client);
    client.send({ event: "live.resync", data: {} });
  });
  return { handler, send };
}

export function workServer() {
  let cells = missing;
  let searches: GrabsSearchList = { searches: [] };
  let grabs: GrabsGrab[] = [otherWork];
  const searched: string[] = [];
  const live = liveChannel();
  const { send } = live;
  const handlers = [
    ...fixedHandlers(),
    getGetWorkCellsMockHandler(() => cells),
    getListSearchesMockHandler(() => searches),
    getListGrabsMockHandler(() => ({ grabs })),
    nodesFrom(() => [home, seedbox]),
    http.post("*/api/v1/works/:workId/search", ({ params }) => {
      searched.push(String(params["workId"]));
      return HttpResponse.json({ searchIds: [searchId] }, { status: 202 });
    }),
    live.handler,
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
