import { HttpResponse, http, sse } from "msw";

import type {
  FilesCellList,
  GrabsGrab,
  GrabsSearchList,
  IndexersIndexer,
  IndexersWaiting,
  TargetsWants,
  WorksWorkView,
} from "../api/gen/model";
import {
  getGetWorkCellsMockHandler,
  getListWorkFilesMockHandler,
} from "../api/gen/query/files/files.msw";
import {
  getListGrabsMockHandler,
  getListSearchesMockHandler,
} from "../api/gen/query/grabs/grabs.msw";
import { getListIndexersMockHandler } from "../api/gen/query/indexers/indexers.msw";
import {
  getListRootsMockHandler,
  getListVolumesMockHandler,
} from "../api/gen/query/roots/roots.msw";
import { getGetWorkWantsMockHandler } from "../api/gen/query/targets/targets.msw";
import {
  getGetWorkMockHandler,
  getListContentTypesMockHandler,
} from "../api/gen/query/works/works.msw";
import { downloading, grabId, imported } from "./grabs";
import type { LiveMap } from "./handlers";
// oxlint-disable-next-line import/max-dependencies -- the Work page reads every section's queries, the Nodes', Targets' and roots' included
import { cinema, orchard, orchardId, orchardItemId } from "./library";
import { home, nodesAre, seedbox, seedboxId } from "./nodes";
import { profileHandlers, ultra } from "./profiles";
import { filmsRoot, mediaVolumes } from "./roots";
import { targetHandlers } from "./targets";
import { filesOf, missing } from "./work-cells";

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

/** The Quiet Orchard on Ultra, wanting 2160p HDR and 1080p. */
export const orchardOnUltra: WorksWorkView = {
  ...orchard,
  profile: { id: ultra.id, name: ultra.name },
};

/** The Quiet Orchard wants what Ultra wants, Edition list included. */
export const fromUltra: TargetsWants = {
  profileId: ultra.id,
  profileName: ultra.name,
  editions: ultra.editions,
  from: "profile",
};

export const unavailable: GrabsSearchList = {
  searches: [
    {
      id: searchId,
      itemId: orchardItemId,
      indexerId: downloading.indexerId,
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

const fixtureSite: IndexersIndexer = {
  id: downloading.indexerId,
  name: "Fixture Site",
  protocol: "torznab",
  url: "http://indexer.test",
  homeNodeId: seedboxId,
  apiKeySet: true,
  searchModes: [],
  waiting: null,
  createdAt: "2026-10-12T09:00:00Z",
};

const fixedHandlers = () => [
  getListContentTypesMockHandler({ contentTypes: [cinema] }),
  getGetWorkMockHandler(orchardOnUltra),
  getGetWorkWantsMockHandler(fromUltra),
  ...profileHandlers(),
  ...targetHandlers(),
  getListRootsMockHandler({ roots: [filmsRoot] }),
  getListVolumesMockHandler({ nodes: [mediaVolumes] }),
  nodesAre(home, seedbox),
];

function liveStream() {
  const clients: Parameters<Parameters<typeof sse<LiveMap>>[1]>[0]["client"][] = [];
  const send: (typeof clients)[number]["send"] = (event) => {
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
  const searched: string[] = [];
  const { handler, send } = liveStream();
  const handlers = [
    ...fixedHandlers(),
    getGetWorkCellsMockHandler(() => cells),
    getListWorkFilesMockHandler(() => filesOf(cells)),
    getListSearchesMockHandler(() => searches),
    getListGrabsMockHandler(() => ({ grabs })),
    getListIndexersMockHandler(() => ({ indexers: [indexer] })),
    http.post("*/api/v1/works/:workId/search", ({ params }) => {
      searched.push(String(params["workId"]));
      return HttpResponse.json({ searchIds: [searchId] }, { status: 202 });
    }),
    handler,
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
    indexerWaits(waiting: IndexersWaiting | null) {
      indexer = { ...indexer, waiting };
      send({ event: "nodes.changed", data: { nodeId: seedboxId } });
    },
    grabMoves(next: GrabsGrab, nextCells = cells) {
      grabs = [next, otherWork];
      cells = nextCells;
      send({ event: "grabs.changed", data: { grabId, workId: orchardId, progress: false } });
    },
  };
}
