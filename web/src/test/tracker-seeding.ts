import { HttpResponse, http } from "msw";

import type {
  DownloadclientsTorrentRow,
  TrackersAttention,
  TrackersGoalsPreview,
  TrackersWindow,
} from "../api/gen/model";
import { cadenceTask, catalogRefresh, healthCheck } from "./catalog-server";
import { aitherTrackerId, torrentLeechTrackerId } from "./indexers";
import { taskServer } from "./tasks";
import { seedingOf, torrent } from "./torrents";

export const seedpoolId = "4d5e6f7a-8b9c-4d0e-9f1a-3b4c5d6e7f80";

/** seedpool's three torrents and one of TorrentLeech's. Hold Fast owes 3 days 4 h. */
export const seedpoolRows: DownloadclientsTorrentRow[] = [
  torrent({
    state: "seeding",
    name: "Hold.Fast.2024.1080p.WEB-DL",
    tabs: ["seeding", "all"],
    tracker: { id: seedpoolId, name: "seedpool" },
    seeding: seedingOf({
      clears: "time",
      state: "owing",
      time: { value: 597_600, rule: 871_200, margin: 7200, goal: 1_209_600 },
      ratio: { value: 0.4, goal: 2 },
      owes: { seconds: 273_600, bytes: 0 },
    }),
  }),
  torrent({
    state: "seeding",
    name: "Low.Tide.2023.1080p.WEB-DL",
    tabs: ["seeding", "all"],
    tracker: { id: seedpoolId, name: "seedpool" },
  }),
  torrent({
    state: "paused",
    name: "Salt.Line.2022.1080p.WEB-DL",
    tabs: ["all"],
    tracker: { id: seedpoolId, name: "seedpool" },
  }),
  torrent({
    state: "downloading",
    name: "Elsewhere.2021.1080p.WEB-DL",
    tabs: ["downloading", "all"],
    tracker: { id: torrentLeechTrackerId, name: "TorrentLeech" },
  }),
];

const [holdFast, lowTide, saltLine, elsewhere] = seedpoolRows;

/** What getTracker says needs a look on each Tracker's page. */
export const attentionOf: { readonly [id: string]: TrackersAttention[] } = {
  [seedpoolId]: [
    {
      level: "warn",
      text: "2 torrents are seeding. 2 still owe seedpool, and 1 of those isn't seeding.",
    },
  ],
  [aitherTrackerId]: [
    {
      level: "info",
      text: "Aither's Releases don't state a seeding rule. Set one so its seeding goal can run.",
    },
  ],
};

const tenDays = { clears: "time", ratio: null, seedSeconds: 864_000 } as const;

const windowOf = (
  hash: string,
  more: Partial<TrackersWindow> & Pick<TrackersWindow, "name" | "state">,
) =>
  ({
    infoHash: hash.repeat(40),
    torrents: [],
    rule: tenDays,
    openedAt: "2026-10-01T09:00:00Z",
    metAt: null,
    closedAt: null,
    regrabbedAt: null,
    owed: null,
    owes: null,
    atRisk: "no",
    quietSince: null,
    ...more,
  }) satisfies TrackersWindow;

/**
 * seedpool's windows, newest opened first: two open, Salt Line missed and
 * grabbed again, then one met, one missed and one released.
 */
export const seedpoolWindows: TrackersWindow[] = [
  windowOf("a", {
    name: "Hold.Fast.2024.1080p.WEB-DL",
    state: "owing",
    torrents: [holdFast?.id ?? ""],
    owes: { seconds: 273_600, bytes: 0 },
  }),
  windowOf("b", {
    name: "Salt.Line.2022.1080p.WEB-DL",
    state: "owing",
    torrents: [saltLine?.id ?? ""],
    owes: { seconds: 432_000, bytes: 0 },
    atRisk: "yes",
    quietSince: "2026-10-12T08:00:00Z",
    closedAt: "2026-10-05T10:00:00Z",
    regrabbedAt: "2026-10-06T10:00:00Z",
    owed: { seconds: 518_400, bytes: 0 },
  }),
  windowOf("c", {
    name: "Low.Tide.2023.1080p.WEB-DL",
    state: "met",
    metAt: "2026-10-03T12:00:00Z",
  }),
  windowOf("d", {
    name: "Ash.Road.2020.1080p.WEB-DL",
    state: "missed",
    closedAt: "2026-09-20T18:30:00Z",
    owed: { seconds: 86_400, bytes: 0 },
  }),
  windowOf("e", {
    name: "Gone.Quiet.2019.1080p.WEB-DL",
    state: "released",
    closedAt: "2026-09-10T07:15:00Z",
  }),
];

/** A torrent row as the goals preview names it. */
export const planned = (row: DownloadclientsTorrentRow | undefined) => ({
  id: row?.id ?? "",
  name: row?.name ?? "",
  tracker: row?.tracker ?? null,
});

/**
 * The goals preview: Low Tide pauses, TorrentLeech's Elsewhere is removed,
 * Hold Fast is removed keeping its download copy, and Salt Line waits on its Grab.
 */
export const goalsPreview: TrackersGoalsPreview = {
  acts: [
    { action: "pause", torrent: planned(lowTide) },
    { action: "remove", torrent: planned(elsewhere) },
  ],
  kept: [{ why: "not-imported", torrent: planned(holdFast) }],
  waits: [{ why: "grab-importing", torrent: planned(saltLine) }],
};

/** The Seeding goals task, every 15 minutes, beside the Indexer tasks. */
export const seedingGoalsTask = cadenceTask("seeding-goals", "Seeding goals", 900, "Every 15 min");

const windowsPage = 2;

const inList: { readonly [list: string]: (window: TrackersWindow) => boolean } = {
  open: (window) => window.state === "owing",
  history: (window) => window.state !== "owing" || window.closedAt !== null,
};

/** GET /trackers/{id}/windows over seedpoolWindows, two to a page, in the list asked for. */
const windowsHandler = http.get("*/api/v1/trackers/:id/windows", ({ params, request }) => {
  const query = new URL(request.url).searchParams;
  const offset = Number(query.get("offset") ?? 0);
  const holds = inList[query.get("list") ?? ""] ?? (() => true);
  const all = (params["id"] === seedpoolId ? seedpoolWindows : []).filter((window) =>
    holds(window),
  );
  const end = offset + windowsPage;
  return HttpResponse.json({
    windows: all.slice(offset, end),
    nextOffset: end < all.length ? end : null,
  });
});

/**
 * The windows, the goals preview and the tasks with Seeding goals among them;
 * taskWrites records each task change.
 */
export function readHandlers(preview: TrackersGoalsPreview) {
  const tasks = taskServer({ tasks: [seedingGoalsTask, catalogRefresh, healthCheck] });
  return {
    handlers: [
      http.get("*/api/v1/torrents/goals/preview", () => HttpResponse.json(preview)),
      windowsHandler,
      ...tasks.handlers,
    ],
    taskWrites: tasks.writes,
  };
}
