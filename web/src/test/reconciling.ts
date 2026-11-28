import { HttpResponse, http } from "msw";

import type {
  ReconcilingNodeRow,
  ReconcilingPage,
  ReconcilingRehomeRequest,
  ReconcilingReviewed,
  ReconcilingRootRow,
  ShellBanner,
} from "../api/gen/model";
import {
  ForgetReconcilingTorrentsBody,
  ReaddReconcilingTorrentsBody,
  RehomeReconcilingBody,
} from "../api/gen/zod/reconciling/reconciling.zod";
import { getGetCountsMockHandler } from "../api/gen/query/shell/shell.msw";
import type { liveServer } from "./live-server";
import { homeId, seedboxId } from "./nodes";
import { filmsRoot } from "./roots";
import { worker } from "./worker";

const since = "2026-11-14T10:12:00Z";

export const restore = {
  backup: "slipstream-20261114T040000Z-2.1.0-nightly-7k3m.tar.zst.age",
  takenAt: "2026-11-14T04:00:00Z",
  startedAt: since,
  newMachine: true,
  rehome: false,
} as const;

export const nodeRow = (
  row: Partial<ReconcilingNodeRow> & Pick<ReconcilingNodeRow, "id" | "name">,
): ReconcilingNodeRow => ({
  since,
  held: 0,
  reported: false,
  adopted: false,
  continued: false,
  granted: null,
  refused: null,
  link: "in-step",
  linkSince: since,
  ...row,
});

export const homeRow = nodeRow({
  id: homeId,
  name: "home",
  held: 12,
  reported: true,
  adopted: true,
});

/** seedbox, a Worker still paired with the old Controller. */
export const seedboxRow = nodeRow({ id: seedboxId, name: "seedbox", link: "offline" });

export const filmsStep: ReconcilingRootRow = {
  id: filmsRoot.id,
  label: "Films library root on home",
  saved: true,
  scanned: false,
  continued: false,
};

export const reconcilingPage = (page: Partial<ReconcilingPage> = {}): ReconcilingPage => ({
  restore,
  nodes: [homeRow, seedboxRow],
  roots: [filmsStep],
  unknown: [],
  review: [],
  ...page,
});

/** The shell's counts member for a page, as the Controller derives it. */
export function bannerOf(page: ReconcilingPage): ShellBanner | null {
  if (page.restore === null) {
    return null;
  }
  return {
    rehome: page.restore.rehome,
    nodes: page.nodes.filter((row) => !row.adopted && !row.continued).map(({ name }) => name),
    roots: page.roots.filter((row) => !row.continued && !(row.saved && row.scanned)).length,
  };
}

const countsOf = (page: ReconcilingPage) =>
  getGetCountsMockHandler({
    activity: 0,
    downloads: { done: 0, wanted: 0, allPaused: false },
    alerts: { nodes: null, health: null, tasks: null, conversions: null },
    pendingImports: 0,
    reconciling: bannerOf(page),
  });

export type Sent = {
  continued: string[];
  rehomed: unknown[];
  readd: unknown[];
  forget: unknown[];
  pages: (string | null)[];
};

type World = { page: ReconcilingPage };

type Changed = (page: ReconcilingPage) => void;

function reviewPage(torrents: readonly ReconcilingReviewed[], sent: Sent, request: Request) {
  const query = new URL(request.url).searchParams;
  const after = query.get("after");
  sent.pages.push(after);
  const limit = Number(query.get("limit") ?? 100);
  const group = torrents.filter(
    (torrent) => torrent.reason === query.get("reason") && torrent.found === query.get("found"),
  );
  const from = after === null ? 0 : group.findIndex(({ id }) => id === after) + 1;
  const rows = group.slice(from, from + limit);
  const last = rows.at(-1);
  const next = from + limit < group.length && last !== undefined ? last.id : null;
  return HttpResponse.json({ torrents: rows, next });
}

function rehomed(world: World, body: ReconcilingRehomeRequest): ReconcilingPage {
  const ids = new Set(body.roots.map(({ rootId }) => rootId));
  return {
    ...world.page,
    restore: world.page.restore && {
      ...world.page.restore,
      rehome: world.page.restore.rehome && !body.done,
    },
    roots: world.page.roots.map((row) => (ids.has(row.id) ? { ...row, saved: true } : row)),
  };
}

const writesOf = (world: World, sent: Sent, changed: Changed) => [
  http.post("*/api/v1/reconciling/nodes/:nodeId/continue", ({ params }) => {
    const id = String(params["nodeId"]);
    sent.continued.push(id);
    changed({
      ...world.page,
      nodes: world.page.nodes.map((row) => (row.id === id ? { ...row, continued: true } : row)),
    });
    return new HttpResponse(null, { status: 204 });
  }),
  http.post("*/api/v1/reconciling/rehome", async ({ request }) => {
    const body = RehomeReconcilingBody.parse(await request.json());
    sent.rehomed.push(body);
    changed(rehomed(world, body));
    return HttpResponse.json({
      roots: body.roots.map(({ rootId }) => ({ rootId, refused: null, missing: null })),
    });
  }),
  http.post("*/api/v1/reconciling/torrents/readd", async ({ request }) => {
    sent.readd.push(ReaddReconcilingTorrentsBody.parse(await request.json()));
    return new HttpResponse(null, { status: 204 });
  }),
  http.post("*/api/v1/reconciling/torrents/forget", async ({ request }) => {
    sent.forget.push(ForgetReconcilingTorrentsBody.parse(await request.json()));
    return new HttpResponse(null, { status: 204 });
  }),
];

/** A Join Code the Controller makes for a Node to pair again. */
export const mintedCode = http.post("*/api/v1/join-codes", () =>
  HttpResponse.json(
    { id: "c1", code: "K7Q2-M4XP-9TRA", expiresAt: "2026-11-21T10:35:00Z" },
    { status: 201 },
  ),
);

/**
 * Serves Reconciling from one page a test changes, and the review list from
 * its torrents in pages of the limit asked. Each write sends
 * reconciling.changed over the test's live stream, as the Controller does.
 */
export function serveReconciling(
  start: ReconcilingPage,
  {
    torrents = [],
    live,
  }: { torrents?: readonly ReconcilingReviewed[]; live?: ReturnType<typeof liveServer> } = {},
) {
  const world: World = { page: start };
  const sent: Sent = { continued: [], rehomed: [], readd: [], forget: [], pages: [] };
  const changed: Changed = (page) => {
    world.page = page;
    worker.use(countsOf(page));
    live?.clients[0]?.send({ event: "reconciling.changed", data: {} });
  };
  worker.use(
    countsOf(start),
    http.get("*/api/v1/reconciling", () => HttpResponse.json(world.page)),
    http.get("*/api/v1/reconciling/torrents", ({ request }) => reviewPage(torrents, sent, request)),
    ...writesOf(world, sent, changed),
  );
  return { world, sent };
}
