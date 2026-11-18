import { HttpResponse, http, sse } from "msw";

import type {
  SubtitlesAttempt,
  SubtitlesCandidates,
  SubtitlesFileSubtitles,
  SubtitlesProvider,
  SubtitlesWantView,
  SubtitlesWorkSubtitles,
  TargetsWantedLanguage,
} from "../api/gen/model";
import {
  getGetWorkSubtitlesMockHandler,
  getListSubtitleProvidersMockHandler,
} from "../api/gen/query/subtitles/subtitles.msw";
import { page } from "vitest/browser";

import type { LiveMap } from "./handlers";
import { orchardId } from "./library";
import { renderApp } from "./render-app";
import { scannedFiles } from "./scanned-files";
import { workServer } from "./work-server";

const hour = 3_600_000;

/** An instant the given hours from now, so sentences read it as today, tonight or tomorrow. */
export const hoursFromNow = (hours: number) => new Date(Date.now() + hours * hour).toISOString();

const provider = (
  id: string,
  name: string,
  position: number,
  kinds: SubtitlesProvider["kinds"],
  keyed: boolean,
): SubtitlesProvider => ({
  id,
  name,
  position,
  kinds,
  needsKey: keyed,
  hasKey: keyed,
  on: true,
  account: null,
  plan: null,
  allowance: null,
  keyPage: null,
  state: { kind: "ready", since: null, until: null, words: null },
});

export const subdl = provider("subdl", "SubDL", 1, ["feature", "episode"], true);
export const providers: readonly SubtitlesProvider[] = [
  subdl,
  provider("subsource", "SubSource", 2, ["feature", "episode"], true),
  provider("gestdown", "Gestdown", 3, ["episode"], false),
];

export const language = (
  code: string,
  change: Partial<TargetsWantedLanguage> = {},
): TargetsWantedLanguage => ({
  language: code,
  forced: false,
  hearingImpaired: "allow",
  audio: "always",
  ...change,
});

const noState = {
  missing: null,
  next: null,
  sidecar: null,
  track: null,
  why: null,
} as const;

export const met = (
  code: string,
  sidecar: Partial<NonNullable<SubtitlesWantView["state"]["sidecar"]>> = {},
): SubtitlesWantView => ({
  want: code,
  wanted: language(code),
  last: null,
  state: {
    ...noState,
    kind: "met",
    sidecar: {
      name: `${code}.srt`,
      source: "provider",
      provider: "subdl",
      release: "The.Quiet.Orchard.2021.2160p.UHD.BluRay.REMUX-FLUX",
      final: false,
      fit: null,
      hindiWarning: false,
      stray: null,
      timingDue: false,
      ...sidecar,
    },
  },
});

export const missing = (
  code: string,
  why: NonNullable<SubtitlesWantView["state"]["missing"]>,
  change: Partial<Pick<SubtitlesWantView, "last">> & { next?: string | null } = {},
): SubtitlesWantView => ({
  want: code,
  wanted: language(code),
  last: change.last ?? null,
  state: { ...noState, kind: "missing", missing: why, next: change.next ?? null },
});

const [uhd, hd] = scannedFiles.files;

/** The Quiet Orchard's 2160p file with these wants and tracks, and nothing beside it. */
export const uhdSubtitles = (
  wants: SubtitlesWantView[],
  change: Partial<SubtitlesFileSubtitles> = {},
): SubtitlesFileSubtitles => ({
  fileId: uhd?.id ?? "",
  path: uhd?.path ?? "",
  season: null,
  episodes: [],
  embedded: [],
  companions: [],
  wants,
  ...change,
});

export const hdSubtitles = (
  wants: SubtitlesWantView[],
  change: Partial<SubtitlesFileSubtitles> = {},
): SubtitlesFileSubtitles => ({
  ...uhdSubtitles(wants, change),
  fileId: hd?.id ?? "",
  path: hd?.path ?? "",
  ...change,
});

export const uhdId = uhd?.id ?? "";

type Answer = () => Response | Promise<Response>;

export type SubtitlesWorld = {
  readonly work: SubtitlesWorkSubtitles;
  readonly search?: Answer;
  readonly start?: Answer;
  readonly candidates?: () => SubtitlesCandidates | Response;
  readonly pick?: Answer;
  readonly attempt?: () => SubtitlesAttempt | Response;
  readonly upload?: Answer;
  readonly keep?: Answer;
  readonly markBad?: Answer;
};

const searchId = "4a6c8e0a-2c4e-4a6c-8e0a-2c4e6a8c0e2b";
export const attemptId = "6c8e0a2c-4e6a-4c8e-9a2c-4e6a8c0e2a4d";

export const started = () => HttpResponse.json({ searchId }, { status: 202 });
export const attempting = () => HttpResponse.json({ attemptId }, { status: 202 });
const accepted = () => new HttpResponse(null, { status: 202 });
const kept = () => new HttpResponse(null, { status: 204 });

const nothingFound: SubtitlesCandidates = { state: "done", outcomes: [], candidates: [] };
const checking: SubtitlesAttempt = {
  step: "checking",
  provider: "subdl",
  outcome: null,
  uploadId: null,
};

type Recorded = { readonly calls: string[]; readonly uploads: unknown[] };

/** Every subtitle action, answering what the world names or else accepting it. */
function actionHandlers(world: SubtitlesWorld, { calls, uploads }: Recorded) {
  const act = (name: string, answer: Answer) => () => {
    calls.push(name);
    return answer();
  };
  return [
    http.post(
      "*/api/v1/files/:fileId/subtitles/:want/search",
      act("search", world.search ?? accepted),
    ),
    http.post(
      "*/api/v1/files/:fileId/subtitles/:want/candidates",
      act("candidates", world.start ?? started),
    ),
    http.get("*/api/v1/files/:fileId/subtitles/:want/candidates/:searchId", () => {
      const read = world.candidates?.() ?? nothingFound;
      return read instanceof Response ? read : HttpResponse.json(read);
    }),
    http.post("*/api/v1/files/:fileId/subtitles/:want/candidates/:searchId/:key", ({ params }) =>
      act(`pick ${String(params["key"])}`, world.pick ?? attempting)(),
    ),
    http.get("*/api/v1/files/:fileId/subtitles/attempts/:attemptId", () => {
      const read = world.attempt?.() ?? checking;
      return read instanceof Response ? read : HttpResponse.json(read);
    }),
    http.post("*/api/v1/files/:fileId/subtitles/:want/upload", async ({ request }) => {
      uploads.push(await request.json());
      return act("upload", world.upload ?? attempting)();
    }),
    http.post(
      "*/api/v1/files/:fileId/subtitles/uploads/:uploadId/keep",
      act("keep", world.keep ?? kept),
    ),
    http.post(
      "*/api/v1/files/:fileId/subtitles/:want/mark-bad",
      act("mark bad", world.markBad ?? accepted),
    ),
    http.put("*/api/v1/files/:fileId/subtitles/tracks/:stream/ignored", ({ params }) => {
      calls.push(`ignore ${String(params["stream"])}`);
      return new HttpResponse(null, { status: 204 });
    }),
  ];
}

/**
 * The Work page's subtitle reads of The Quiet Orchard and every subtitle
 * action, each accepted unless the world names an answer. It records each action's call and sends live events of its own.
 */
export function subtitlesServer(world: SubtitlesWorld) {
  let work = world.work;
  const calls: string[] = [];
  const uploads: unknown[] = [];
  const clients: Parameters<Parameters<typeof sse<LiveMap>>[1]>[0]["client"][] = [];
  const send: (typeof clients)[number]["send"] = (event) => {
    for (const client of clients) {
      client.send(event);
    }
  };
  const handlers = [
    sse<LiveMap>("*/api/v1/events", ({ client }) => {
      clients.push(client);
      client.send({ event: "live.resync", data: {} });
    }),
    getGetWorkSubtitlesMockHandler(() => work),
    getListSubtitleProvidersMockHandler({ providers: [...providers] }),
    ...actionHandlers(world, { calls, uploads }),
  ];
  return {
    handlers,
    calls,
    uploads,
    send,
    /** The subtitles change, as a Write would, and the page hears of it. */
    change(next: SubtitlesWorkSubtitles) {
      work = next;
      send({ event: "subtitles.changed", data: { workIds: [orchardId], fileIds: [] } });
    },
    attemptMoves() {
      send({ event: "subtitles.attempt", data: { attemptId, fileId: uhdId } });
    },
  };
}

/** The Quiet Orchard's Work page, its files the scanned ones, over this subtitles world. */
export function openOrchard(world: SubtitlesWorld) {
  const subtitles = subtitlesServer(world);
  renderApp({
    path: `/works/${orchardId}`,
    handlers: [...subtitles.handlers, ...workServer({ files: scannedFiles }).handlers],
  });
  return subtitles;
}

/** A subtitle row by its language's name. */
export const subtitleRow = (name: string) =>
  page.getByRole("list", { name: "Subtitles" }).getByRole("listitem").filter({ hasText: name });

export const notifications = () => page.getByRole("region", { name: "Messages" });
