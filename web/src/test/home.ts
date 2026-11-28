import { HttpResponse, type RequestHandler, http } from "msw";
import { page } from "vitest/browser";

import type {
  DownloadclientsNodeTorrents,
  FilesWaitingNeed,
  GrabsGrab,
  HomeArrival,
  HomeJob,
  HomeNeed,
  HomeSummary,
  HomeWork,
  SettingCountField,
  ShellCounts,
} from "../api/gen/model";
import { getGetCountsMockHandler } from "../api/gen/query/shell/shell.msw";
import { getListContentTypesMockHandler } from "../api/gen/query/works/works.msw";
import { SettingsInputBody } from "../api/gen/zod/slipStream.schemas.zod";
import { type Artwork, artworkHandlers, artworkUrl } from "./ambient/fixtures";
import { downloading, fetchFailure } from "./grabs";
import { cinema } from "./library";
import { liveServer } from "./live-server";
// oxlint-disable-next-line import/max-dependencies -- Home's fixtures build on the Grab, torrent, Node and live fixtures
import { homeId } from "./nodes";
import { renderApp } from "./render-app";
import type { Row } from "../torrents/model";

let serial = 0;

const nextId = (prefix: string) => {
  serial += 1;
  return `${prefix}-0000-4000-8000-${serial.toString(16).padStart(12, "0")}`;
};

/** A Work as Home names it, with the ambient test images as its art. */
export function homeWork(
  label: string,
  more: { backdrop?: Artwork; poster?: Artwork; failed?: boolean; contentType?: string } = {},
): HomeWork {
  return {
    id: nextId("3c000000"),
    label,
    contentType: more.contentType ?? "cinema",
    posterUrl: more.poster === undefined ? null : artworkUrl(more.poster),
    backdropUrl: more.backdrop === undefined ? null : artworkUrl(more.backdrop),
    failed: more.failed ?? false,
  };
}

/** A moving Grab of the Work, its own id and Work id set. */
export function job(grab: GrabsGrab, work: HomeWork, torrent: Row | null = null): HomeJob {
  return { grab: { ...grab, id: nextId("6c000000"), workId: work.id }, work, torrent };
}

export function need(grab: GrabsGrab, work: HomeWork, wait: FilesWaitingNeed | null): HomeNeed {
  const id = nextId("6d000000");
  return {
    grab: { ...grab, id, workId: work.id },
    work,
    wait: wait === null ? null : { ...wait, grabId: id },
  };
}

export const waitOf = (
  code: FilesWaitingNeed["code"],
  more: Partial<FilesWaitingNeed> = {},
): FilesWaitingNeed => ({
  kind: "waiting",
  grabId: "",
  code,
  clientError: null,
  importWait: null,
  ...more,
});

export function arrival(work: HomeWork, words: string, at: string): HomeArrival {
  return { importId: nextId("1a000000"), work, words, at };
}

export const nodeTorrents = (
  nodeId: string,
  more: Partial<DownloadclientsNodeTorrents> = {},
): DownloadclientsNodeTorrents => ({
  nodeId,
  torrents: 0,
  downloading: 0,
  seeding: 0,
  failures: 0,
  warnings: 0,
  ...more,
});

export function summaryOf({
  jobs = [],
  moving = jobs.length,
  needs = [],
  failed = needs.filter(({ wait }) => wait === null).length,
  waiting = needs.filter(({ wait }) => wait !== null).length,
  arrivals = [],
  today = { imported: 0, upgraded: 0 },
}: {
  jobs?: HomeJob[];
  moving?: number;
  needs?: HomeNeed[];
  failed?: number;
  waiting?: number;
  arrivals?: HomeArrival[];
  today?: HomeSummary["recent"]["today"];
} = {}): HomeSummary {
  return {
    jobs: { rows: jobs, moving },
    needsYou: { rows: needs, moves: [], failed, waiting },
    recent: { rows: arrivals, today },
  };
}

export const rowsField = (value: number): SettingCountField => ({
  kind: "count",
  key: "home.rows",
  label: "Rows on Home",
  value,
  running: value,
  default: 8,
  min: 3,
  max: 50,
  unit: { one: "entry", many: "entries" },
  source: "database",
  applies: "live",
  isDefault: value === 8,
  pendingRestart: false,
});

/**
 * A Controller's Home over the live stream: the summary it answers, which
 * a test changes, and the Torrent counts. `since` keeps each summary read's
 * midnight.
 */
export function homeServer(
  initial: HomeSummary,
  counted: readonly DownloadclientsNodeTorrents[] = [],
) {
  let summary = initial;
  let rows = 8;
  const since: string[] = [];
  const live = liveServer();
  const handlers = [
    ...live.handlers,
    ...artworkHandlers,
    getListContentTypesMockHandler({ contentTypes: [cinema] }),
    http.get("*/api/v1/home", ({ request }) => {
      since.push(new URL(request.url).searchParams.get("since") ?? "");
      return HttpResponse.json(summary);
    }),
    http.get("*/api/v1/torrents/counts", () => HttpResponse.json({ nodes: counted })),
    http.get("*/api/v1/home-settings", () => HttpResponse.json({ settings: [rowsField(rows)] })),
    http.patch("*/api/v1/home-settings", async ({ request }) => {
      const { values } = SettingsInputBody.parse(await request.json());
      rows = Number(values["home.rows"]);
      return HttpResponse.json({ settings: [rowsField(rows)] });
    }),
  ];
  return {
    handlers,
    since,
    clients: live.clients,
    become(next: HomeSummary) {
      summary = next;
    },
  };
}

export const quietCounts: ShellCounts = {
  activity: 0,
  downloads: { done: 0, wanted: 0, allPaused: false },
  alerts: { nodes: null, health: null, tasks: null, conversions: null },
  pendingImports: 0,
  reconciling: null,
};

export const countsAre = (more: Partial<ShellCounts>) =>
  getGetCountsMockHandler({ ...quietCounts, ...more });

export const orchardWork = homeWork("The Quiet Orchard", { poster: "red", backdrop: "red" });
export const harvestWork = homeWork("Orchard Harvest", {
  poster: "two-tone",
  backdrop: "two-tone",
});
export const bareWork = homeWork("Bare Branches");

/** A Grab that failed fetching its .torrent at `at`, which needs the Admin. */
export const failedGrab = (at: string): GrabsGrab => ({
  ...downloading,
  stage: "failed",
  inFlight: false,
  history: [{ stage: "failed", at }],
  failure: fetchFailure,
});

/** Home at /, with home as its one Downloading Node unless handlers say otherwise. */
export function renderHome(summary: HomeSummary, handlers: RequestHandler[] = []) {
  const server = homeServer(summary, [nodeTorrents(homeId, { torrents: 1, downloading: 1 })]);
  const app = renderApp({ path: "/", handlers: [...handlers, ...server.handlers] });
  return { app, server };
}

export const jobRow = (label: string) =>
  page.getByRole("row", { name: new RegExp(`^${label} `, "u") });

export const homeSentence = () => page.getByRole("main").getByRole("paragraph").first();
