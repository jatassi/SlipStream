import { HttpResponse, http, sse } from "msw";

import type {
  FilesCellList,
  GrabsGrab,
  GrabsSearchList,
  IndexersIndexer,
  NodesNode,
  TargetsWants,
} from "../api/gen/model";
import { getGetWorkCellsMockHandler } from "../api/gen/query/files/files.msw";
import {
  getListGrabsMockHandler,
  getListSearchesMockHandler,
} from "../api/gen/query/grabs/grabs.msw";
import { getListIndexersMockHandler } from "../api/gen/query/indexers/indexers.msw";
import { getGetWorkWantsMockHandler } from "../api/gen/query/targets/targets.msw";
import {
  getGetWorkMockHandler,
  getListContentTypesMockHandler,
} from "../api/gen/query/works/works.msw";
import { grabId, imported, indexerId } from "./grabs";
import type { LiveMap } from "./handlers";
import { cinema, orchard, orchardId, orchardItemId } from "./library";
// oxlint-disable-next-line import/max-dependencies -- the Work page reads every section's queries, the Nodes' and the Profiles' included
import { home, nodesFrom, seedbox, seedboxId } from "./nodes";
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
export const searching: GrabsSearchList = {
  searches: [
    {
      id: searchId,
      itemId: orchardItemId,
      indexerId,
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
      indexerId,
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

/** The Quiet Orchard wants what Films wants, Edition list included. */
export const fromFilms: TargetsWants = {
  profileId: films.id,
  profileName: films.name,
  editions: films.editions,
  from: "profile",
};

export const unavailable: GrabsSearchList = {
  searches: [
    {
      id: searchId,
      itemId: orchardItemId,
      indexerId,
      startedAt: "2026-10-12T09:00:58Z",
      finishedAt: "2026-10-12T09:01:00Z",
      outcome: "indexer_unavailable",
      candidates: 0,
      refused: [],
      grabId: null,
      notes: [],
    },
  ],
};

const limitUnused = { count: 0, per: "hour", used: 0, freeAt: null } as const;

const fixtureSite: IndexersIndexer = {
  id: indexerId,
  name: "Fixture Site",
  kind: "torznab",
  homeNodeId: seedboxId,
  on: true,
  automaticSearch: true,
  rss: true,
  allowFailover: false,
  priority: 25,
  categories: null,
  searches: limitUnused,
  grabs: limitUnused,
  status: {
    state: "ok",
    cause: null,
    text: null,
    until: null,
    since: null,
    nodeId: null,
    solverTried: false,
    rung: 0,
    lastFailure: null,
    successAt: "2026-10-12T09:00:00Z",
  },
  sentence: "Fixture Site is working.",
  fix: null,
  signsIn: false,
  trackerIds: [],
  waits: null,
  session: null,
  feedHost: "indexer.test",
  searchModes: [],
  revision: 1,
  createdAt: "2026-10-12T09:00:00Z",
};

/** How seedbox stands while the Indexer homed on it waits for it. */
export type HomeAway = { readonly standing: NodesNode["standing"]; readonly since: string };

const waitsFor = (away: HomeAway | null): IndexersIndexer["waits"] =>
  away === null
    ? null
    : {
        kind: "waiting",
        releases: 0,
        until: null,
        since: away.since,
        nodeId: seedboxId,
        lapsed: false,
        cause: null,
        text: null,
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
  let indexer = fixtureSite;
  let box = seedbox;
  const searched: string[] = [];
  const live = liveChannel();
  const { send } = live;
  const handlers = [
    ...fixedHandlers(),
    getGetWorkCellsMockHandler(() => cells),
    getListSearchesMockHandler(() => searches),
    getListGrabsMockHandler(() => ({ grabs })),
    getListIndexersMockHandler(() => ({ indexers: [indexer] })),
    nodesFrom(() => [home, box]),
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
    indexerWaits(away: HomeAway | null) {
      box = away === null ? seedbox : { ...seedbox, standing: away.standing };
      indexer = { ...indexer, waits: waitsFor(away) };
      send({ event: "nodes.changed", data: { nodeId: seedboxId } });
      send({ event: "indexers.status-changed", data: { indexerId: indexer.id } });
    },
    grabMoves(next: GrabsGrab, nextCells = cells) {
      grabs = [next, otherWork];
      cells = nextCells;
      send({ event: "grabs.changed", data: { grabId, workId: orchardId } });
    },
  };
}
