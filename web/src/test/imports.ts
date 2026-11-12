import { HttpResponse, http } from "msw";

import type {
  ImportsLead,
  ImportsOutcome,
  WorksCandidate,
  WorksItem,
  WorksLibraryRow,
  WorksWorkView,
} from "../api/gen/model";
import {
  ActOnRootImportBody,
  AddRootImportsBody,
  IgnoreImportsBody,
} from "../api/gen/zod/imports/imports.zod";
import type { RootEntry } from "../imports/root-entry";
import type { liveServer } from "./live-server";
import { libraryRow, orchard } from "./library";
import {
  brazil,
  filmsLead,
  filmsShelf,
  filmsVault,
  halloween,
  halloween1978,
  halloween2018,
  heat,
  orchardLeft,
  severanceEntry,
  severanceId,
  shows,
  showsLead,
  showsShelf,
} from "./pending-imports";
import { worker } from "./worker";

export const severanceRow: WorksLibraryRow = libraryRow({
  id: severanceId,
  key: "371980",
  label: "Severance",
  contentType: shows.id,
  rootId: showsShelf.id,
});

export const brazilRow: WorksLibraryRow = libraryRow({
  id: "0199f0c3-0000-7000-8000-000000001985",
  key: "68",
  label: "Brazil",
  rootId: filmsShelf.id,
});

export const vaultRow: WorksLibraryRow = libraryRow({
  id: "0199f0c3-0000-7000-8000-000000000949",
  key: "949",
  label: "Heat",
  rootId: filmsVault.id,
});

const episodeId = (season: number, episode: number) =>
  `0199f0d4-0000-7000-8000-0000000${String(season).padStart(2, "0")}${String(episode).padStart(3, "0")}`;

const brazilItem: WorksItem = {
  id: "0199f0d4-0000-7000-8000-000000001985",
  levelId: null,
  key: "film",
  label: "Brazil",
  position: 1,
  withdrawnAt: null,
  monitored: true,
  aside: false,
  fields: [],
  ratings: [],
  artwork: [],
  availability: null,
};

const episode = (season: number, number: number, label: string) => ({
  ...brazilItem,
  id: episodeId(season, number),
  levelId: `season-${season}`,
  key: `${season}x${number}`,
  label,
  position: number,
  fields: [
    {
      key: "code",
      kind: "text" as const,
      text: `S${String(season).padStart(2, "0")}E${String(number).padStart(2, "0")}`,
    },
  ],
});

export const goodNews = episode(1, 1, "Good News About Hell");
export const halfLoop = episode(1, 2, "Half Loop");

export const severance: WorksWorkView = {
  ...orchard,
  id: severanceId,
  contentType: shows.id,
  key: "371980",
  label: "Severance",
  sortLabel: "Severance",
  fields: [],
  items: [goodNews, halfLoop],
};

export const brazilWork: WorksWorkView = {
  ...orchard,
  id: brazilRow.id,
  key: brazilRow.key,
  label: "Brazil",
  sortLabel: "Brazil",
  items: [brazilItem],
};

export type ImportsWorld = {
  entries: RootEntry[];
  leads: ImportsLead[];
  candidates: WorksCandidate[];
  rows: WorksLibraryRow[];
  works: WorksWorkView[];
  /** Publishes imports.changed, as the Controller does after each change. */
  changed: (rootIds: string[]) => void;
};

type Sent = {
  added: string[][];
  matched: { id: string; key: string }[];
  imported: { id: string; workId: string; files: { position: number; itemIds: string[] }[] }[];
  ignored: string[][];
  restored: string[][];
  searched: string[];
};

const problemType = { "Content-Type": "application/problem+json" };

const refused = (status: number, code: string, title: string) =>
  HttpResponse.json({ status, code, title }, { status, headers: problemType });

const needsYou = (entry: RootEntry) => entry.ignoredAt === null;

const failed = (id: string, code: string, title: string): ImportsOutcome => ({
  id,
  outcome: "failed",
  problem: { code, title },
});

function outcomeOf(world: ImportsWorld, entry: RootEntry): ImportsOutcome {
  const { id, suggestion } = entry;
  if (suggestion === null) {
    return failed(id, "import_no_suggestion", "The Import has no suggested Work");
  }
  if (entry.files.some(({ reason }) => reason.code === "elsewhere")) {
    return failed(id, "import_elsewhere", "The Work lives in another library root");
  }
  world.entries = world.entries.filter((candidate) => candidate.id !== id);
  return {
    id,
    outcome: "added",
    workId: suggestion.workId ?? `0199f0c3-9999-7000-8000-${id.slice(-12)}`,
  };
}

function matchedOf(entry: RootEntry, candidate: WorksCandidate): RootEntry {
  const year = candidate.fields.find((field) => field.key === "year");
  return {
    ...entry,
    suggestion: {
      key: candidate.key,
      label: candidate.label,
      year: year?.kind === "number" ? year.number : null,
      posterUrl: candidate.posterUrl,
      workId: candidate.workId,
      confident: false,
    },
  };
}

const readsOf = (world: ImportsWorld, sent: Sent) => [
  http.get("*/api/v1/imports", ({ request }) => {
    const ignored = new URL(request.url).searchParams.get("ignored") === "true";
    return HttpResponse.json({
      leads: world.leads,
      imports: world.entries.filter((entry) => needsYou(entry) !== ignored),
    });
  }),
  http.get("*/api/v1/content-types/:id/catalog", ({ request }) => {
    sent.searched.push(new URL(request.url).searchParams.get("query") ?? "");
    return HttpResponse.json({ candidates: world.candidates });
  }),
  http.get("*/api/v1/library", () => HttpResponse.json({ rows: world.rows })),
  http.get("*/api/v1/works/:workId", ({ params }) => {
    const work = world.works.find(({ id }) => id === params["workId"]);
    return work === undefined
      ? refused(404, "work_not_found", "No Work has that id")
      : HttpResponse.json(work);
  }),
];

const marksOf = (world: ImportsWorld, sent: Sent) => [
  http.post("*/api/v1/imports/ignore", async ({ request }) => {
    const { importIds } = IgnoreImportsBody.parse(await request.json());
    sent.ignored.push(importIds);
    return marked(world, importIds, "2026-11-08T10:00:00Z");
  }),
  http.post("*/api/v1/imports/unignore", async ({ request }) => {
    const { importIds } = IgnoreImportsBody.parse(await request.json());
    sent.restored.push(importIds);
    return marked(world, importIds, null);
  }),
];

const writesOf = (world: ImportsWorld, sent: Sent) => [
  http.post("*/api/v1/imports/add", async ({ request }) => {
    const { importIds } = AddRootImportsBody.parse(await request.json());
    sent.added.push(importIds);
    const named = world.entries.filter(({ id }) => importIds.includes(id));
    const outcomes = named.map((entry) => outcomeOf(world, entry));
    world.changed(named.map(({ root }) => root.id));
    return HttpResponse.json({ outcomes });
  }),
  http.post("*/api/v1/imports/:id", async ({ params, request }) => {
    const id = String(params["id"]);
    const body = ActOnRootImportBody.parse(await request.json());
    const entry = world.entries.find((found) => found.id === id);
    if (body.action === "match") {
      sent.matched.push({ id, key: body.key });
      const candidate = world.candidates.find((found) => found.key === body.key);
      if (candidate === undefined || entry === undefined) {
        return refused(
          404,
          "metadata_not_found",
          "The metadata source has no record with that key",
        );
      }
      const matched = matchedOf(entry, candidate);
      world.entries = world.entries.map((found) => (found.id === id ? matched : found));
      world.changed([entry.root.id]);
      return HttpResponse.json(matched);
    }
    sent.imported.push({ id, workId: body.workId, files: body.files });
    if (entry === undefined) {
      return refused(404, "import_not_found", "No Import has that id");
    }
    const chosen = new Set(body.files.map(({ position }) => position));
    const files = entry.files.filter(({ position }) => !chosen.has(position));
    const left: RootEntry = { ...entry, stage: files.length === 0 ? "done" : "waiting", files };
    world.entries =
      files.length === 0
        ? world.entries.filter((found) => found.id !== id)
        : world.entries.map((found) => (found.id === id ? left : found));
    world.changed([entry.root.id]);
    return HttpResponse.json(left);
  }),
];

function marked(world: ImportsWorld, ids: readonly string[], ignoredAt: string | null) {
  const roots = world.entries.filter(({ id }) => ids.includes(id)).map(({ root }) => root.id);
  world.entries = world.entries.map((entry) =>
    ids.includes(entry.id) ? { ...entry, ignoredAt } : entry,
  );
  world.changed(roots);
  return new HttpResponse(null, { status: 204 });
}

/** Serves Pending imports from one mutable world, recording what the page sent. */
export function serveImports(start: Partial<ImportsWorld> = {}) {
  const world: ImportsWorld = {
    entries: [halloween, brazil, heat, severanceEntry, orchardLeft],
    leads: [filmsLead, showsLead],
    candidates: [halloween1978, halloween2018],
    rows: [brazilRow, vaultRow, severanceRow],
    works: [brazilWork, severance],
    changed: () => {},
    ...start,
  };
  const sent: Sent = {
    added: [],
    matched: [],
    imported: [],
    ignored: [],
    restored: [],
    searched: [],
  };
  worker.use(...readsOf(world, sent), ...marksOf(world, sent), ...writesOf(world, sent));
  return { world, sent };
}

/** Sends imports.changed to the page over a test's live stream. */
export const changedOver =
  (live: ReturnType<typeof liveServer>) =>
  (rootIds: string[]): void => {
    live.clients[0]?.send({ event: "imports.changed", data: { rootIds } });
  };
