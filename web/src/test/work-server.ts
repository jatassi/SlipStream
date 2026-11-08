import { HttpResponse, http, sse } from "msw";

import type {
  FilesCellList,
  FilesFileList,
  GrabsGrab,
  GrabsSearch,
  GrabsSearchIndexer,
  GrabsSearchList,
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
import {
  getListRootsMockHandler,
  getListVolumesMockHandler,
} from "../api/gen/query/roots/roots.msw";
import { getGetWorkWantsMockHandler } from "../api/gen/query/targets/targets.msw";
import { ForgetFilesBody } from "../api/gen/zod/files/files.zod";
import {
  getGetWorkMockHandler,
  getListContentTypesMockHandler,
} from "../api/gen/query/works/works.msw";
import { anthologies } from "./anthology";
import { grabId, imported, indexerId } from "./grabs";
// oxlint-disable-next-line import/max-dependencies -- the Work page reads every section's queries, the Nodes', Targets' and roots' included
import { type LiveMap, problem } from "./handlers";
import { cinema, orchard, orchardId, orchardItemId } from "./library";
import { home, nodesAre, seedbox } from "./nodes";
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
          "Isn't responding. SlipStream retries at 14:20.",
          "failing",
        ),
        line(
          "Gale",
          "6a7b8c9d-0e1f-4a2b-8c3d-5e6f7a8b9c0d",
          "Stopped because the username or password was rejected.",
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

const fixedHandlers = (work: WorksWorkView) => [
  getListContentTypesMockHandler({ contentTypes: [cinema, anthologies] }),
  getGetWorkMockHandler(work),
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

/** A Work's files, the ones its cells hold unless files names them, and Forget over them. */
function filesWorld(
  files: FilesFileList | undefined,
  cellsNow: () => FilesCellList,
  changed: () => void,
) {
  let listed = files;
  const forgot: string[][] = [];
  const now = () => listed ?? filesOf(cellsNow());
  return {
    forgot,
    handlers: [
      getListWorkFilesMockHandler(() => now()),
      http.post("*/api/v1/files/forget", async ({ request }) => {
        const { fileIds } = ForgetFilesBody.parse(await request.json());
        forgot.push(fileIds);
        const known = now();
        if (!fileIds.every((id) => known.files.some((file) => file.id === id))) {
          return problem(404, "file_not_found", "No file has that id");
        }
        listed = { ...known, files: known.files.filter((file) => !fileIds.includes(file.id)) };
        changed();
        return new HttpResponse(null, { status: 204 });
      }),
    ],
  };
}

/**
 * The Work page's reads, of The Quiet Orchard on Ultra unless work names
 * another Work and its cells. Its files are the ones its cells hold, unless
 * files names them, as for gone files a scan found.
 */
export function workServer({
  work = orchardOnUltra,
  startCells = missing,
  files,
}: { work?: WorksWorkView; startCells?: FilesCellList; files?: FilesFileList } = {}) {
  let cells = startCells;
  let searches: GrabsSearchList = { searches: [] };
  let grabs: GrabsGrab[] = [otherWork];
  const searched: string[] = [];
  const { handler, send } = liveStream();
  const changed = () => {
    send({ event: "files.changed", data: { workId: work.id } });
  };
  const filed = filesWorld(files, () => cells, changed);
  const handlers = [
    ...fixedHandlers(work),
    getGetWorkCellsMockHandler(() => cells),
    ...filed.handlers,
    getListSearchesMockHandler(() => searches),
    getListGrabsMockHandler(() => ({ grabs })),
    http.post("*/api/v1/works/:workId/search", ({ params }) => {
      searched.push(String(params["workId"]));
      return HttpResponse.json({ searchIds: [searchId] }, { status: 202 });
    }),
    handler,
  ];
  return {
    handlers,
    searched,
    forgot: filed.forgot,
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
      send({ event: "grabs.changed", data: { grabId, workId: orchardId, progress: false } });
    },
  };
}
