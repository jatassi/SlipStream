import type {
  DownloadclientsTorrentList,
  DownloadclientsTorrentRow,
  WorksListedWork,
} from "../api/gen/model";
import { getListTorrentsMockHandler } from "../api/gen/query/downloadclients/downloadclients.msw";
import { getListCutsMockHandler } from "../api/gen/query/targets/targets.msw";
import {
  getListContentTypesMockHandler,
  getListWorksMockHandler,
} from "../api/gen/query/works/works.msw";
import { cinema, orchard, orchardId } from "./library";
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
  fields: [],
};

const zines = { ...cinema, id: "zines", name: "Zines" };

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

/**
 * The Controller's Torrents and what the view reads beside them. `rows` is
 * what GET /torrents answers from now on; `asked` records each ids splice.
 */
export function torrentsServer(initial: readonly Row[]) {
  let rows = [...initial];
  const asked: string[][] = [];
  return {
    handlers: [
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
      nodesAre(home, seedbox),
      getListContentTypesMockHandler({ contentTypes: [cinema, zines] }),
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
