import { HttpResponse, http } from "msw";

import type {
  ActInputBody,
  DownloadclientsActResult,
  DownloadclientsPreview,
  DownloadclientsTorrentDetail,
  DownloadclientsTorrentList,
  DownloadclientsTorrentRow,
  WorksContentType,
  WorksListedWork,
} from "../api/gen/model";
import { getListTorrentsMockHandler } from "../api/gen/query/downloadclients/downloadclients.msw";
import { getListCutsMockHandler } from "../api/gen/query/targets/targets.msw";
import {
  getListContentTypesMockHandler,
  getListWorksMockHandler,
} from "../api/gen/query/works/works.msw";
import { artworkHandlers, artworkUrl } from "./ambient/fixtures";
import { cinema, orchardId, orchardListed } from "./library";
import { home, homeId, nodesAre, seedbox, seedboxId } from "./nodes";
import { targetHandlers } from "./targets";

type Row = DownloadclientsTorrentRow;

const states = [
  "removing",
  "failed",
  "tracker-error",
  "paused",
  "checking",
  "queued",
  "stalled",
  "downloading",
  "seeding",
] as const;
const tabs = ["downloading", "seeding", "problems", "all"] as const;

export const harvestId = "5d7f9b1c-3e5a-4c7e-9f1b-3d5f7a9c1e2b";

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
  artwork: [],
};

const zines = { ...cinema, id: "zines", name: "Zines" };

const posteredCinema: WorksContentType = {
  ...cinema,
  artwork: [{ key: "poster", label: "Poster", shape: "poster", tier: "work" }],
};

export const orchardPoster = artworkUrl("red");

const posteredOrchard: WorksListedWork = {
  ...orchardListed,
  artwork: [{ kind: "poster", url: orchardPoster }],
};

let serial = 0;

export function torrent(row: Partial<Row> & Pick<Row, "state">): Row {
  serial += 1;
  const tail = serial.toString(16).padStart(12, "0");
  return {
    id: `7a000000-0000-4000-8000-${tail}`,
    nodeId: homeId,
    name: `Release.${serial}.1080p.WEB-DL`,
    contentType: "cinema",
    severity: "none",
    tabs: ["all"],
    size: 1_000_000_000,
    done: 0,
    uploaded: 0,
    ratio: 0,
    seedingSeconds: 0,
    addedAt: `2026-10-12T09:${String(serial % 60).padStart(2, "0")}:00Z`,
    ...row,
  };
}

export function listOf(rows: readonly Row[]): DownloadclientsTorrentList {
  const nodes = [...new Set(rows.map((row) => row.nodeId))];
  return {
    rows: [...rows],
    counts: {
      byState: states.map((state) => ({
        state,
        torrents: rows.filter((row) => row.state === state).length,
      })),
      byTab: tabs.map((tab) => ({
        tab,
        torrents: rows.filter((row) => row.tabs.includes(tab)).length,
      })),
      byNode: nodes.map((nodeId) => ({
        nodeId,
        torrents: rows.filter((row) => row.nodeId === nodeId).length,
      })),
    },
  };
}

/** A row's detail as GET /torrents/{id} answers it while its Node is in step. */
export function detailOf(
  row: Row,
  more: Partial<DownloadclientsTorrentDetail> = {},
): DownloadclientsTorrentDetail {
  return {
    ...row,
    downloaded: row.done,
    activeSeconds: 3600,
    infoHash: "0102030405060708090a0b0c0d0e0f1011121314",
    saveFolder: `/srv/downloads/Movies/0102030405060708090a0b0c0d0e0f1011121314`,
    pieceSize: 4 * 1024 * 1024,
    private: true,
    trackers: [],
    files: [{ index: 0, path: `${row.name}/${row.name}.mkv`, size: row.size, done: row.done }],
    grabs: [],
    live: true,
    ...more,
  };
}

/**
 * POST /torrents/actions as the Controller answers it: each action done
 * unless `refuse` names a refusal for its id, and a dry run's preview.
 */
export function actionsServer({
  preview = { cancels: [], stopSeeding: [], frees: 0, linked: false },
  refuse,
}: {
  preview?: DownloadclientsPreview;
  refuse?: (id: string, body: ActInputBody) => DownloadclientsActResult["refusal"];
} = {}) {
  const bodies: ActInputBody[] = [];
  return {
    bodies,
    handler: http.post("*/api/v1/torrents/actions", async ({ request }) => {
      // oxlint-disable-next-line typescript/no-unsafe-type-assertion -- the generated client sends the body it types
      const body = (await request.json()) as ActInputBody;
      bodies.push(body);
      return HttpResponse.json({
        results: body.ids.map((id) => {
          const refusal = refuse?.(id, body);
          return refusal === undefined ? { id, done: true } : { id, done: false, refusal };
        }),
        ...(body.dryRun === true && { preview }),
      });
    }),
  };
}

// GET /torrents/:id: the detail of a row the list holds, else not found.
function detailHandler(
  rows: () => readonly Row[],
  details: (row: Row) => DownloadclientsTorrentDetail,
) {
  return http.get("*/api/v1/torrents/:id", ({ params }) => {
    const row = rows().find(({ id }) => id === params["id"]);
    return row === undefined
      ? HttpResponse.json({ code: "torrent_not_found", title: "No such torrent" }, { status: 404 })
      : HttpResponse.json(details(row));
  });
}

/**
 * The Controller's Torrents and what the view reads beside them. `rows` is
 * what GET /torrents answers from now on; `asked` records each ids splice.
 */
export function torrentsServer(
  initial: readonly Row[],
  details: (row: Row) => DownloadclientsTorrentDetail = (row) => detailOf(row),
) {
  let rows = [...initial];
  const asked: string[][] = [];
  return {
    handlers: [
      detailHandler(() => rows, details),
      getListTorrentsMockHandler(({ request }) => {
        const ids = new URL(request.url).searchParams.get("ids");
        if (ids === null) {
          return listOf(rows);
        }
        const named = ids.split(",");
        asked.push(named);
        return { ...listOf(rows), rows: rows.filter((row) => named.includes(row.id)) };
      }),
      getListWorksMockHandler({
        works: [harvest, posteredOrchard],
      }),
      nodesAre(home, seedbox),
      getListContentTypesMockHandler({ contentTypes: [posteredCinema, zines] }),
      ...artworkHandlers,
      getListCutsMockHandler({
        cuts: [
          { key: "standard", name: "Standard", aliases: [], origin: "builtin" },
          { key: "directors", name: "Director's Cut", aliases: [], origin: "builtin" },
        ],
      }),
      ...targetHandlers(),
    ],
    asked,
    become(next: readonly Row[]) {
      rows = [...next];
    },
  };
}

export { homeId, orchardId, seedboxId };
