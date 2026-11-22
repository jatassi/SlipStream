import { HttpResponse, http } from "msw";

import type {
  DownloadclientsClientsList,
  MovesMovePreview,
  MovesStartedMoves,
  NodesNode,
  RootsRoot,
  RootsRootList,
  RootsVolumeList,
  WorksContentTypeList,
} from "../api/gen/model";
import { homeCard, seedboxCard, thresholds } from "./download-clients";
import { cinema } from "./library";
import { fromFilms, toArchive } from "./moves";
import { home, homeId, nodesAre, seedbox, seedboxId } from "./nodes";
import { renderApp } from "./render-app";
import { board, boardIs, homeSeat, outcome, seedboxSeat } from "./roles";
import { filmsRoot, mediaVolume, notScanned } from "./roots";

export const seedboxFilms: RootsRoot = {
  ...filmsRoot,
  id: "0199f0a1-1111-7000-8000-0000000000d1",
  nodeId: seedboxId,
  path: "/srv/films",
  default: true,
  scan: notScanned,
};

export const homeFilms: RootsRoot = {
  ...filmsRoot,
  id: "0199f0a1-1111-7000-8000-0000000000e1",
  nodeId: homeId,
  path: "/mnt/films",
  default: false,
  volumeId: "sdb1",
};

const fullerFilms: RootsRoot = {
  ...homeFilms,
  id: "0199f0a1-1111-7000-8000-0000000000e2",
  path: "/mnt/full",
  volumeId: "sdc1",
};

const onlineSeedbox: NodesNode = {
  ...seedbox,
  standing: "in-step",
  link: { state: "up", since: "2026-10-12T10:00:00Z" },
};

const seedboxNamed = { id: seedboxId, name: "seedbox" };

const drains = outcome({
  effects: [
    {
      kind: "drains",
      node: seedboxNamed,
      role: "storage",
      held: { units: 2, noun: "films", bytes: 8 * 1024 ** 3, endlessUnits: 0 },
      words: "seedbox takes no new Storage work. It keeps its 2 films until you move them.",
    },
    {
      kind: "drains",
      node: seedboxNamed,
      role: "downloading",
      held: { units: 1, noun: "torrents", bytes: 4 * 1024 ** 3, endlessUnits: 1 },
      words: "seedbox takes no new Downloading work. It keeps its 1 torrent until you move it.",
    },
  ],
});

const preview: MovesMovePreview = {
  token: "drain-1",
  expiresAt: "2099-01-01T00:00:00Z",
  recommendation: "Move 2 of 2 Works and torrents to home.",
  button: "Move 2 of 2",
  movable: 2,
  total: 2,
  homed: 3,
  units: [
    {
      key: "work:1",
      kind: "work",
      name: "The Quiet Orchard",
      workId: "0199f0c1-1111-7000-8000-000000000001",
      from: fromFilms,
      to: toArchive,
      verdict: "green",
      bytes: 4 * 1024 ** 3,
      reasons: [],
    },
    {
      key: "torrent:1",
      kind: "torrent",
      name: "Harbor.Lights.2019.1080p",
      workId: "",
      torrentId: "0199f0d1-1111-7000-8000-000000000001",
      from: fromFilms,
      to: toArchive,
      verdict: "amber",
      bytes: 4 * 1024 ** 3,
      reasons: [
        {
          check: "torrent",
          code: "downloading",
          color: "amber",
          words: "seedbox is finishing its work. This torrent moves once it finishes downloading.",
        },
      ],
    },
  ],
};

const started: MovesStartedMoves = {
  batchId: "0199f0b1-1111-7000-8000-000000000009",
  started: 2,
  dropped: [],
};

const served = (path: string, body: object) => http.get(path, () => HttpResponse.json(body));

/** seedbox draining Storage and Downloading on its Roles page, with every Move request it was sent. */
export function drainWorld() {
  const sent = { previews: [] as unknown[], starts: [] as unknown[], order: [] as string[] };
  renderApp({
    path: `/nodes/${seedboxId}/roles`,
    handlers: [
      nodesAre(home, onlineSeedbox),
      boardIs(board(homeSeat, seedboxSeat)),
      http.post("*/api/v1/nodes/roles/plan", () => HttpResponse.json(drains)),
      http.put("*/api/v1/nodes/roles", () => {
        sent.order.push("roles");
        return HttpResponse.json(outcome({ basis: "basis-2" }));
      }),
      served("*/api/v1/roots", {
        roots: [seedboxFilms, homeFilms, fullerFilms],
      } satisfies RootsRootList),
      served("*/api/v1/volumes", {
        nodes: [
          {
            nodeId: homeId,
            live: true,
            volumes: [
              { ...mediaVolume, id: "sdb1", free: 2 * 1024 ** 4, rootIds: [homeFilms.id] },
              { ...mediaVolume, id: "sdc1", free: 1024 ** 3, rootIds: [fullerFilms.id] },
            ],
          },
        ],
      } satisfies RootsVolumeList),
      served("*/api/v1/download-clients", {
        cards: [homeCard, seedboxCard],
        thresholds,
        orphans: [],
        kinds: [],
      } satisfies DownloadclientsClientsList),
      served("*/api/v1/content-types", { contentTypes: [cinema] } satisfies WorksContentTypeList),
      http.post("*/api/v1/moves/preview", async ({ request }) => {
        sent.previews.push(await request.json());
        return HttpResponse.json(preview);
      }),
      http.post("*/api/v1/moves", async ({ request }) => {
        sent.order.push("move");
        sent.starts.push(await request.json());
        return HttpResponse.json(started, { status: 202 });
      }),
    ],
  });
  return sent;
}
